package gateway

import (
	"context"
	"log"
	"slices"
	"strings"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/labstack/echo/v4"
)

func (h *Handler) checkPermissions(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		destination, err := h.getDestination(c)
		if err != nil {
			log.Print(err)
			return err
		}

		tenantID := c.Request().Header.Get(TenantIDHeader)

		if tenantID == "" {
			return echo.ErrBadRequest
		}

		queryTenants := []string{tenantID}

		if strings.Contains(tenantID, "|") {
			// If the tenantID contains a pipe, this is a multi-tenant request
			// and we need to split it into multiple tenants
			// X-Scope-OrgID:Tenant1|Tenant2|Tenant3
			//
			// We dont support this for now...
			return echo.NewHTTPError(echo.ErrNotImplemented.Code, "multi-tenant requests are not supported yet")
			// queryTenants = strings.Split(tenantID, "|")
		}

		var claims *Claims
		if h.tokenValidation {
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

		for _, tenantID := range queryTenants {
			if tenant, ok := destination.Tenants[tenantID]; ok {
				found := slicesContains(tenant.Groups, claims.Groups)

				if (tenant.Mode == "allowlist" && !found) || (tenant.Mode == "denylist" && found) {
					return echo.ErrForbidden
				}
			} else if !destination.AllowUndefined {
				// Deny access if the tenant is not defined
				return echo.ErrForbidden
			}
		}

		c.Set("tenantNames", queryTenants)
		c.Set("groups", claims.Groups)
		c.Set("email", claims.Email)
		c.Set("destination", destination)

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

func slicesContains(groups []config.Group, claims []string) bool {
	for _, group := range groups {
		if slices.Contains(claims, group.Name) {
			return true
		}
	}

	return false
}

func (h *Handler) getDestination(c echo.Context) (config.Destination, error) {
	host := c.Request().Host
	tenantID := c.Request().Header.Get(TenantIDHeader)

	if host == "" || tenantID == "" {
		return config.Destination{}, echo.ErrBadRequest
	}

	// Previous middleware validates that the target exists
	destination := h.config.Destinations[host]

	return destination, nil
}
