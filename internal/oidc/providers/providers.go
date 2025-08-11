package providers

import (
	"context"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/oidc/providers/entra"
	"github.com/coreos/go-oidc"
)

type Provider interface {
	Validate(context.Context, string) (*oidc.IDToken, error)
}

func AvailableProviders() []string {
	return []string{entra.ProviderName}
}
