package gateway

import (
	"context"
	"log"
	"slices"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/providers/entra"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/stacks/loki"
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
	}

	balancer := NewCustomBalancer(config.Destinations)

	e.Use(
		balancer.CheckTarget,
		handler.checkPermissions,
		handler.enforceLBAC,
		middleware.ProxyWithConfig(
			middleware.ProxyConfig{
				Balancer: balancer,
			},
		),
	)

	e.Logger.Fatal(
		e.Start(":" + c.String("port")),
	)

	return nil
}

// MUST be called after the checkPermissions middleware
// This middleware enforces the LBAC rules for the user groups
// It modifies the query to include the matchers for the groups
func (h *Handler) enforceLBAC(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {

		tenant := c.Get("tenant").(*Tenant)
		userGroups := c.Get("groups").([]string)
		stack := c.Get("stack").(StackType)
		// email := c.Get("email").(string)

		if tenant == nil || userGroups == nil || len(userGroups) == 0 {
			// if we get here, it means that something went terribly wrong
			return echo.ErrInternalServerError
		}

		switch stack {
		case StackLoki:
			e, err := loki.ParseQuery(c.Request().URL.Query().Get("query"))
			if err != nil {
				return echo.ErrBadRequest
			}

			for _, group := range tenant.Groups {
				if !slices.Contains(userGroups, group.Name) {
					continue
				}
				loki.EnforceLBAC(e, group.Matchers)
			}

			// patch the query with the new one
			c.Request().URL.Query().Set("query", e.String())

		case StackMimir, StackPrometheus:
			return echo.ErrNotImplemented
		case StackTempo:
			return echo.ErrNotImplemented
		case StackPyroscope:
			return echo.ErrNotImplemented
		default:
			return echo.ErrUnprocessableEntity
		}

		return next(c)
	}
}
