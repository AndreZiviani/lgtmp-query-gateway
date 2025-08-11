package gateway

import (
	"context"
	"encoding/json"
	"log"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/oidc/providers/mock"
	oidcTypes "github.com/AndreZiviani/lgtmp-query-gateway/internal/oidc/types"
	"github.com/AndreZiviani/lgtmp-query-gateway/internal/util"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/prometheus/model/labels"
)

func (h *Handler) checkPermissions(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var err error

		// All validations are done in the previous middleware
		destination := c.Get("destination").(config.Destination)
		tenantNames := c.Get("tenantNames").([]string)

		// If token validation is enabled, we need to validate the token
		claims, err := h.validateToken(c.Request().Context(), c.Request().Header.Get("x-id-token"))
		if err != nil {
			log.Print(err)
			return echo.ErrUnauthorized
		}

		enforcedLabels := make([]*labels.Matcher, 0)
		for _, tenantID := range tenantNames {
			if tenant, ok := destination.Tenants[tenantID]; ok {
				found := false
				for _, profile := range tenant.Profiles {
					if util.SlicesContains(claims.Groups, h.config.Profiles[profile].Groups) {
						enforcedLabels = append(enforcedLabels, h.config.Profiles[profile].Matchers...)
						found = true
					}
				}

				if (tenant.Mode == "allowlist" && !found) || (tenant.Mode == "denylist" && found) {
					return echo.ErrForbidden
				}
			} else if !destination.AllowUndefined {
				// Deny access if the tenant is not defined
				return echo.ErrForbidden
			}
		}

		c.Set("groups", claims.Groups)
		c.Set("email", claims.Email)
		c.Set("enforcedLabels", enforcedLabels)

		return next(c)
	}
}

func (h *Handler) validateToken(ctx context.Context, token string) (*oidcTypes.Claims, error) {
	if !h.tokenValidation {
		log.Printf("Token validation is disabled, using mock claims for testing purposes")
		// Mock the claims for testing purposes
		return &oidcTypes.Claims{
			Groups: []string{"group1", "group2"},
			Email:  "user@example.com",
			Name:   "User",
			Roles:  []string{"role1", "role2"},
		}, nil
	}

	// If we are using a mock provider, treat the token as Claims instead of OIDC
	// because it is hard to mock it
	if _, ok := h.provider.(*mock.Provider); ok {
		log.Printf("Using mock provider for testing purposes")
		c := oidcTypes.Claims{}
		err := json.Unmarshal([]byte(token), &c)
		if err != nil {
			return nil, err
		}
		return &c, nil
	}

	idToken, err := h.provider.Validate(ctx, token)
	if err != nil {
		return nil, err
	}

	claims := &oidcTypes.Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}
