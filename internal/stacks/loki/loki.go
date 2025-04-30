package loki

import (
	"log"
	"slices"
	"strings"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/grafana/loki/v3/pkg/logql/syntax"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/prometheus/model/labels"
)

const (
	RouteInstantQuery      = "/loki/api/v1/query"
	RouteRangeQuery        = "/loki/api/v1/query_range"
	RouteLabels            = "/loki/api/v1/labels"
	RouteLabelValuesPrefix = "/loki/api/v1/label/"
	RouteSeries            = "/loki/api/v1/series"
	RouteIndexStats        = "/loki/api/v1/index/stats"
	RouteInstantLogVolume  = "/loki/api/v1/index/volume"
	RouteRangeLogVolume    = "/loki/api/v1/index/volume_range"
	RoutePattern           = "/loki/api/v1/patterns"
	RouteTailStream        = "/loki/api/v1/tail" // WebSocket
)

func Handle(c echo.Context) error {
	path := c.Request().URL.Path

	if strings.HasPrefix(path, RouteLabelValuesPrefix) {
		// We cant enforce LBAC here
		return nil
	}

	var query, field string

	switch path {
	case RouteLabels:
		// "query" parameter is optional, but if it is specified it must match something
		// queries require at least one regexp or equality matcher that does not have an
		// empty-compatible value. For instance, app=~".*" does not meet this requirement,
		// but app=~".+" will, which means that we can't just inject filters removing some
		// labels (e.g. app!="api").
		//
		// Allow user to view all labels, we will enforce LBAC on the matchers when querying the log

		return nil

	case RouteInstantQuery, RouteRangeQuery,
		RouteIndexStats, RouteInstantLogVolume, RouteRangeLogVolume, RoutePattern:

		field = "query"
		query = c.Request().URL.Query().Get(field)

	case RouteSeries:
		// This can be either a GET or POST request
		// query: match[]=<selector> (can be repeated)
		// You can URL-encode these parameters directly in the request body by using
		// the POST method and Content-Type: application/x-www-form-urlencoded header.
		// Get the query from the request
		if c.Request().Method == "POST" {
			// TODO
			return echo.ErrNotImplemented
		}

		field = "match"
		query = c.Request().URL.Query().Get(field)

	case RouteTailStream:
		// Also receives a query parameter "query", but this is a WebSocket
		// TODO
		return echo.ErrNotImplemented

	default:
		return echo.ErrBadRequest
	}

	expr, err := ParseQuery(query)
	if err != nil {
		log.Println(err)
		return err
	}

	err = PatchExpression(c, expr)
	if err != nil {
		log.Println(err)
		return err
	}

	// Patch the query with the new one
	PatchRequest(c, field, expr)

	return nil
}

func ParseQuery(query string) (syntax.Expr, error) {
	// Parse the query using Loki's syntax parser
	return syntax.ParseExpr(query)
}

func PatchExpression(c echo.Context, expr syntax.Expr) error {
	// Get the tenant from the request
	destination := c.Get("destination").(config.Destination)
	tenantNames := c.Get("tenantNames").([]string)

	// We dont support multi tenant requests for now
	tenant, ok := destination.Tenants[tenantNames[0]]
	if !ok {
		if destination.AllowUndefined {
			// Allow access if the tenant is not defined
			return nil
		}
		return echo.ErrBadRequest
	}

	userGroups := c.Get("groups").([]string)

	found := false
	enforcedLabels := make([]*labels.Matcher, 0)
	// A user can be part of multiple groups, so we need to check all of them
	// and see if any of them match any of the groups in the tenant
	for _, group := range tenant.Groups {
		if !slices.Contains(userGroups, group.Name) {
			continue
		}
		if len(group.Matchers) > 0 {
			// if we get here, it means that the user is part of a group
			// that has LBAC rules
			enforcedLabels = append(enforcedLabels, group.Matchers...)
		}
		found = true
	}

	if tenant.Mode == config.ModeAllowList {
		// This tenant requires that the user is part of at least one of the groups
		if !found {
			return echo.ErrForbidden
		}
	} else {
		// This tenant requires that the user is not part of any of the groups
		if found {
			return echo.ErrForbidden
		}
	}

	err := EnforceLBAC(expr, enforcedLabels)
	if err != nil {
		log.Printf("failed to enforce LBAC: %v", err)
		return echo.NewHTTPError(400, "invalid query: %v", err)
	}

	return nil
}

func EnforceLBAC(e syntax.Expr, lbac []*labels.Matcher) error {
	if len(lbac) == 0 {
		// no labels to enforce
		return nil
	}

	if e == nil {
		return echo.ErrBadRequest
	}

	// must check if any labels are already set in the expression
	// if so, we must rewrite them instead of adding them

	selector := getSelector(e)

OUTER:
	for _, l := range lbac {
		for _, m := range selector.Matchers() {
			if m.Name == l.Name {
				// user already set this label, we must rewrite it instead of adding it
				m = l
				continue OUTER
			}
		}
		appendMatcher(selector, l)
	}

	return nil
}

func appendMatcher(selector syntax.LogSelectorExpr, matcher *labels.Matcher) {
	visitor := &syntax.DepthFirstTraversal{
		VisitMatchersFn: func(_ syntax.RootVisitor, m *syntax.MatchersExpr) {
			m.AppendMatchers([]*labels.Matcher{matcher})
		},
	}
	selector.Accept(visitor)
}

// getSelector returns the selector from the expression
// an expression can have multiple levels of nesting
func getSelector(e syntax.Expr) syntax.LogSelectorExpr {
	var selector syntax.LogSelectorExpr

	visitor := &syntax.DepthFirstTraversal{
		VisitMatchersFn: func(_ syntax.RootVisitor, m *syntax.MatchersExpr) {
			selector = m
		},
	}
	e.Accept(visitor)

	return selector
}

func PatchRequest(c echo.Context, parameterName string, expr syntax.Expr) {
	// patch the query with the new one
	patchedQuery := c.Request().URL.Query() // this returns a copy and not a reference
	patchedQuery.Set(parameterName, expr.String())
	c.Request().URL.RawQuery = patchedQuery.Encode()
}
