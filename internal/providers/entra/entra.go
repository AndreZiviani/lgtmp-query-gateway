package entra

import (
	"context"
	"fmt"
	"net/http"

	oidc "github.com/coreos/go-oidc"
)

const (
	DiscoveryEndpoint string = "https://login.microsoftonline.com/%s/v2.0"
)

type AzureSettings struct {
	TenantID string
	ClientID string
}

type EntraProvider struct {
	oidcVerifier *oidc.IDTokenVerifier
	settings     *AzureSettings
	httpClient   *http.Client
	// authorizer           autorest.Authorizer
	// authorizerExpiration time.Time
	// lock                 sync.RWMutex
}

func New(settings *AzureSettings) (*EntraProvider, error) {
	discovery := fmt.Sprintf(DiscoveryEndpoint, settings.TenantID)
	provider, err := oidc.NewProvider(context.Background(), discovery)
	if err != nil {
		return nil, err
	}

	oidcVerifier := provider.Verifier(&oidc.Config{ClientID: settings.ClientID})

	return &EntraProvider{
		oidcVerifier: oidcVerifier,
		settings:     settings,
		httpClient:   &http.Client{},
	}, nil
}

func (p *EntraProvider) Validate(ctx context.Context, token string) (*oidc.IDToken, error) {
	idToken, err := p.oidcVerifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	return idToken, nil
}
