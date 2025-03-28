package providers

import "github.com/AndreZiviani/lgtmp-query-gateway/internal/providers/entra"

func AvailableProviders() []string {
	return []string{entra.ProviderName}
}