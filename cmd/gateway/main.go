package main

import (
	"os"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/cli"
)

func main() {
	if cli.Run() != nil {
		os.Exit(1)
	}
}
