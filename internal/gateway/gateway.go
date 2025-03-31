package gateway

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/otel"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/providers/entra"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/stacks/loki"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/stacks/mimir"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
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
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer stop()

	wg := &sync.WaitGroup{}

	otel.Initialize(ctx, wg)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(otelecho.Middleware("echo"))

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

	go func() {
		if err := e.Start(":" + c.String("port")); err != nil && err != http.ErrServerClosed {
			log.Fatalf("shutting down server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), c.Duration("drain-duration"))
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("shutting down server: %v", err)
	}
	wg.Wait()
	log.Println("server shut down gracefully")
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
