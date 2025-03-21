package gateway

import (
	"github.com/urfave/cli/v3"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "serve",
		Usage:  "Run the gateway",
		Action: Serve,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "mode",
				Usage:   "Gateway mode (proxy, middleware)",
				Aliases: []string{"m"},
				Sources: cli.EnvVars("MODE"),
				Value:   "proxy",
			},
			&cli.StringFlag{
				Name:    "provider",
				Usage:   "Provider to use for authentication (entra)",
				Aliases: []string{"p"},
				Sources: cli.EnvVars("PROVIDER"),
				Value:   "entra",
			},
			&cli.StringFlag{
				Name:     "tenant-id",
				Usage:    "Azure Tenant ID",
				Aliases:  []string{"t"},
				Sources:  cli.EnvVars("TENANT_ID"),
				Required: true,
			},
			&cli.StringFlag{
				Name:     "client-id",
				Usage:    "Azure Client ID",
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
			&cli.UintFlag{
				Name:    "port",
				Usage:   "Port to listen on",
				Sources: cli.EnvVars("PORT"),
				Value:   9000,
			},
		},
	}
}
