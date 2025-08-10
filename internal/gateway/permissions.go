package gateway

import (
	"context"
	"log"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
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

		var claims *Claims
		if h.tokenValidation {
			token := c.Request().Header.Get("x-id-token")
			if token == "" {
				return echo.ErrUnauthorized
			}

			// If token validation is enabled, we need to validate the token
			claims, err = h.validateToken(c.Request().Context(), c.Request().Header.Get("x-id-token"))
			if err != nil {
				log.Print(err)
				return echo.ErrUnauthorized
			}
		} else {
			log.Printf("Token validation is disabled, using mock claims for testing purposes")
			// Mock the claims for testing purposes
			claims = &Claims{
				Groups: []string{"group1", "group2"},
				Email:  "user@example.com",
				Name:   "User",
				Roles:  []string{"role1", "role2"},
			}
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

func (h *Handler) validateToken(ctx context.Context, token string) (*Claims, error) {
	idToken, err := h.provider.Validate(ctx, token)
	if err != nil {
		return nil, err
	}

	claims := &Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, err
	}

	return claims, nil
}
