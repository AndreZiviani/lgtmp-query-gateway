package gateway

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CustomBalancer struct {
	targets map[string]*middleware.ProxyTarget
}

// AddTarget implements the ProxyBalancer interface (required by Echo)
func (b *CustomBalancer) AddTarget(*middleware.ProxyTarget) bool {
	// Not used in this example, as we statically define targets.
	return true
}

// RemoveTarget implements the ProxyBalancer interface (required by Echo)
func (b *CustomBalancer) RemoveTarget(string) bool {
	// Not used in this example, as we statically define targets.
	return true
}

// Next selects the appropriate backend based on the Host header
// User must ensure that the target exists
func (b *CustomBalancer) Next(c echo.Context) *middleware.ProxyTarget {
	host := c.Request().Host
	target := b.targets[host]

	// Patch host header to match the target
	c.Request().Host = target.URL.Host

	return target
}

// CheckTarget is a middleware that checks if the target exists
func (b *CustomBalancer) CheckTarget(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		host := c.Request().Host
		if _, ok := b.targets[host]; !ok {
			return echo.ErrNotFound
		}

		return next(c)
	}
}
