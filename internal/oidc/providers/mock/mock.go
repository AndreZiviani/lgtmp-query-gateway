package mock

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coreos/go-oidc"
)

type idToken struct {
	Issuer       string                 `json:"iss"`
	Subject      string                 `json:"sub"`
	Audience     audience               `json:"aud"`
	Expiry       jsonTime               `json:"exp"`
	IssuedAt     jsonTime               `json:"iat"`
	NotBefore    *jsonTime              `json:"nbf"`
	Nonce        string                 `json:"nonce"`
	AtHash       string                 `json:"at_hash"`
	ClaimNames   map[string]string      `json:"_claim_names"`
	ClaimSources map[string]claimSource `json:"_claim_sources"`
}

type Provider struct{}

func (p *Provider) Validate(ctx context.Context, token string) (*oidc.IDToken, error) {
	var idToken idToken
	if err := json.Unmarshal([]byte(token), &idToken); err != nil {
		return nil, err
	}

	return &oidc.IDToken{
		Issuer:          idToken.Issuer,
		Subject:         idToken.Subject,
		Audience:        []string(idToken.Audience),
		Expiry:          time.Time(idToken.Expiry),
		IssuedAt:        time.Time(idToken.IssuedAt),
		Nonce:           idToken.Nonce,
		AccessTokenHash: idToken.AtHash,
	}, nil
}
