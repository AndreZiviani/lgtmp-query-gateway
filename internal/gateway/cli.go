package gateway

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/oidc/providers"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/oidc/providers/entra"
	"github.com/urfave/cli/v3"
)

const (
	ProxyMode      RunMode = "proxy"
	MiddlewareMode RunMode = "middleware"
)

type RunMode string

func Command() *cli.Command {
	return &cli.Command{
		Name:   "serve",
		Usage:  "Run the gateway",
		Action: Serve,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "provider",
				Usage:   "Provider to use for authentication",
				Aliases: []string{"p"},
				Sources: cli.EnvVars("PROVIDER"),
				Value:   entra.ProviderName,
				Action: func(ctx context.Context, c *cli.Command, v string) error {
					p := providers.AvailableProviders()
					if !slices.Contains(p, v) {
						return cli.Exit(fmt.Sprintf("Invalid provider, available options: %v", p), 1)
					}
					return nil
				},
			},
			&cli.StringFlag{
				Name:     "tenant-id",
				Usage:    "EntraID Tenant ID",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("TENANT_ID"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "client-id",
				Usage:    "EntraID Client ID",
				Aliases:  []string{"c"},
				Sources:  cli.EnvVars("CLIENT_ID"),
				Required: true,
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "Path to the configuration file",
				Aliases: []string{"f"},
				Sources: cli.EnvVars("CONFIG"),
				Value:   "config.yaml",
			},
			&cli.BoolFlag{
				Name:    "disable-token-validation",
				Usage:   "Disable OIDC Token validation",
				Sources: cli.EnvVars("DISABLE_OIDC_TOKEN_VALIDATION"),
				Value:   false,
			},
			&cli.DurationFlag{
				Name:    "drain-duration",
				Usage:   "Duration to wait before shutting down the server",
				Sources: cli.EnvVars("DRAIN_DURATION"),
				Value:   30 * time.Second,
			},
		},
	}
}
