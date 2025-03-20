package cli

import (
	"context"
	"log"
	"os"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/gateway"
	"github.com/urfave/cli/v3"
)

func Run() error {

	ctx, _ := context.WithCancel(context.Background())

	flags := []cli.Flag{
		&cli.BoolFlag{Name: "verbose", Usage: "Log debug messages"},
	}

	app := &cli.Command{
		Flags:     flags,
		Name:      "lgtmp-query-gateway",
		Usage:     "https://github.com/AndreZiviani/lgtmp-query-gateway",
		UsageText: "", // TODO
		// Version:     version, // TODO
		HideVersion: false,
		Commands: []*cli.Command{
			gateway.Command(),
		},
		EnableShellCompletion: true,
	}

	if err := app.Run(ctx, os.Args); err != nil {
		log.Fatal(err)
	}

	return nil
}
