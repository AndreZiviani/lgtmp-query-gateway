package gateway

import (
	"log"
	"net/url"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CustomBalancer struct {
	targets map[string]*middleware.ProxyTarget
}

func NewCustomBalancer(destinations map[string]config.Destination) *CustomBalancer {
	targets := map[string]*middleware.ProxyTarget{}
	for host, dest := range destinations {
		if dest.Upstream == "" {
			log.Panic("missing upstream for destination " + host)
		}

		upstream, err := url.Parse(dest.Upstream)
		if err != nil {
			log.Panic(err)
		}

		targets[host] = &middleware.ProxyTarget{
			URL: upstream,
		}
	}

	return &CustomBalancer{
		targets: targets,
	}
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
func (b *CustomBalancer) checkTarget(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		host := c.Request().Host
		if _, ok := b.targets[host]; !ok {
			return echo.ErrNotFound
		}

		return next(c)
	}
}
