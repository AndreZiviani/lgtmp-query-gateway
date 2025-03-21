package gateway

import (
	"context"
	"log"
	"net/url"
	"slices"
	"strconv"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/providers/entra"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/urfave/cli/v3"
)

const (
	TenantIDHeader = "X-Scope-OrgID"
)

type Handler struct {
	provider  *entra.EntraProvider
	config    *config
	mode      string
	upstreams map[string]middleware.ProxyTarget
}

type Claims struct {
	Groups    []string `json:"groups"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
	NotBefore int64    `json:"nbf"`
	Issuer    string   `json:"iss"`
}

func Serve(ctx context.Context, c *cli.Command) error {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	mode := c.String("mode")

	config, err := LoadConfig(c.String("config"))
	if err != nil {
		log.Panic(err)
	}

	p, err := entra.New(&entra.AzureSettings{
		TenantID: c.String("tenant-id"),
		ClientID: c.String("client-id"),
	})
	if err != nil {
		log.Panic(err)
	}

	handler := &Handler{
		provider: p,
		config:   config,
		mode:     mode,
	}

	e.Use(handler.checkPermissions)

	if mode == "proxy" {
		targets := map[string]*middleware.ProxyTarget{}
		for host, dest := range config.Destinations {
			if dest.Upstream == "" {
				log.Panic("missing upstream for destination " + host)
			}

			upstream, err := url.Parse(dest.Upstream)
			if err != nil {
				log.Panic(err)
			}

			targets[host] = &middleware.ProxyTarget{
				URL: upstream,
			}
		}

		balancer := &CustomBalancer{
			targets: targets,
		}

		e.Use(
			balancer.CheckTarget,
			middleware.ProxyWithConfig(
				middleware.ProxyConfig{
					Balancer: balancer,
				},
			),
		)
	}

	e.Logger.Fatal(
		e.Start(":"+strconv.Itoa(int(c.Uint("port")))),
	)

	return nil
}

func (h *Handler) validateToken(ctx context.Context, token string) (*Claims, error) {
	idToken, err := h.provider.Validate(ctx, token)
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

func (h *Handler) checkPermissions(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		token := c.Request().Header.Get("x-id-token")
		if token == "" {
			return c.String(401, "Unauthorized")
		}

		claims, err := h.validateToken(c.Request().Context(), token)
		if err != nil {
			log.Print(err)
			return c.String(401, "Unauthorized")
		}

		httpErr := h.handle(c, claims)
		if httpErr != nil {
			log.Print(err)
			return httpErr
		}

		return next(c)
	}
}

func (h *Handler) handle(c echo.Context, claims *Claims) *echo.HTTPError {
	host := c.Request().Host
	tenantID := c.Request().Header.Get(TenantIDHeader)

	if host == "" || tenantID == "" {
		return echo.ErrBadRequest
	}

	if dest, ok := h.config.Destinations[host]; ok {
		if tenant, ok := dest.Tenants[tenantID]; ok {
			found := slicesContains(tenant.Groups, claims.Groups)

			if (tenant.Mode == "allowlist" && !found) || (tenant.Mode == "denylist" && found) {
				return echo.ErrForbidden
			}

			return nil
		}
	}

	return echo.ErrUnprocessableEntity
}

func slicesContains(groups []Group, claims []string) bool {
	for _, group := range groups {
		if slices.Contains(claims, group.Name) {
			return true
		}
	}

	return false
}
