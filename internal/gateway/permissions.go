package gateway

import (
	"context"
	"log"
	"slices"

	"github.com/labstack/echo/v4"
)

func (h *Handler) checkPermissions(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		claims, err := h.validateToken(c.Request().Context(), c.Request().Header.Get("x-id-token"))
		if err != nil {
			log.Print(err)
			return echo.ErrUnauthorized
		}

		host := c.Request().Host
		tenantID := c.Request().Header.Get(TenantIDHeader)

		if host == "" || tenantID == "" {
			return echo.ErrBadRequest
		}

		// Previous middleware validates that the target exists
		destination := h.config.Destinations[host]

		if tenant, ok := destination.Tenants[tenantID]; ok {
			c.Set("tenant", tenant)
			c.Set("groups", claims.Groups)
			c.Set("email", claims.Email)
			c.Set("stack", destination.Type)

			found := slicesContains(tenant.Groups, claims.Groups)

			if (tenant.Mode == "allowlist" && !found) || (tenant.Mode == "denylist" && found) {
				return echo.ErrForbidden
			}
		} else if !destination.AllowUndefined {
			// Deny access if the tenant is not defined
			return echo.ErrForbidden
		}

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

func slicesContains(groups []Group, claims []string) bool {
	for _, group := range groups {
		if slices.Contains(claims, group.Name) {
			return true
		}
	}

	return false
}
