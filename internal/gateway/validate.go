package gateway

import (
	"log"
	"strings"

	"github.com/labstack/echo/v4"
)

func (h *Handler) validateRequest(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		destination, ok := h.config.Destinations[c.Request().Host]
		if !ok {
			log.Printf("destination not found for host: %s", c.Request().Host)
			return echo.ErrBadRequest
		}

		tenantID := c.Request().Header.Get(TenantIDHeader)

		if tenantID == "" {
			log.Printf("tenant ID not found in request header: %s", TenantIDHeader)
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

		c.Set("tenantNames", queryTenants)
		c.Set("destination", destination)

		return next(c)
	}
}
