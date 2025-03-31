package gateway

import (
	"context"
	"log"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/providers/entra"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/stacks/loki"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/stacks/mimir"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/urfave/cli/v3"
)

const (
	TenantIDHeader = "X-Scope-OrgID"
)

type Handler struct {
	provider        *entra.EntraProvider
	config          *config.Config
	tokenValidation bool
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

	config, err := config.LoadConfig(c.String("config"))
	if err != nil {
		log.Panic(err)
	}

	tokenValidation := !c.Bool("disable-token-validation")
	var provider *entra.EntraProvider
	if tokenValidation {
		provider, err = entra.New(&entra.AzureSettings{
			TenantID: c.String("tenant-id"),
			ClientID: c.String("client-id"),
		})
		if err != nil {
			log.Panic(err)
		}
	}

	handler := &Handler{
		provider:        provider,
		config:          config,
		tokenValidation: tokenValidation,
	}

	balancer := NewCustomBalancer(config.Destinations)

	e.Use(
		balancer.checkTarget,
		handler.checkPermissions,
		handler.handle,
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
func (h *Handler) handle(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		destination := c.Get("destination").(config.Destination)

		switch destination.Type {
		case config.StackLoki:
			err := loki.Handle(c)
			if err != nil {
				return err
			}

		case config.StackMimir, config.StackPrometheus:
			err := mimir.Handle(c)
			if err != nil {
				return err
			}

		case config.StackTempo:
			return echo.ErrNotImplemented
		case config.StackPyroscope:
			return echo.ErrNotImplemented
		default:
			return echo.ErrUnprocessableEntity
		}

		return next(c)
	}
}
