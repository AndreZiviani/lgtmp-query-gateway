package mimir

import (
	"log"
	"slices"
	"strings"

	"github.com/AndreZiviani/lgtmp-query-gateway/internal/config"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

const (
	RouteInstantQuery           = "/prometheus/api/v1/query"
	RouteRangeQuery             = "/prometheus/api/v1/query_range"
	RouteQueryExemplars         = "/prometheus/api/v1/query_exemplars"
	RouteSeries                 = "/prometheus/api/v1/series"
	RouteActiveSeries           = "/prometheus/api/v1/cardinality/active_series"
	RouteLabels                 = "/prometheus/api/v1/labels"
	RouteLabelValuesPrefix      = "/prometheus/api/v1/label/"
	RouteMetadata               = "/prometheus/api/v1/metadata"
	RouteRemoteRead             = "/prometheus/api/v1/read"
	RouteLabelNamesCardinality  = "/prometheus/api/v1/cardinality/label_names"
	RouteLabelValuesCardinality = "/prometheus/api/v1/cardinality/label_values"
	RouteFormatQuery            = "/prometheus/api/v1/format_query"
	RouteBuildInfo              = "/prometheus/api/v1/status/buildinfo"
)

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func Handle(c echo.Context) error {
	if c.Request().Method == "POST" {
		// TODO
		return nil
	}

	path := c.Request().URL.Path

	if strings.HasPrefix(path, RouteLabelValuesPrefix) {
		// We cant enforce LBAC here
		return nil
	}

	switch path {
	case RouteInstantQuery, RouteRangeQuery, RouteQueryExemplars, RouteFormatQuery:

		err := PatchQuery(c, "query")
		if err != nil {
			log.Println(err)
			return err
		}

		return nil

	case RouteSeries, RouteLabels:
		// This can be either a GET or POST request
		// query: match[]=<selector> (can be repeated)
		// You can URL-encode these parameters directly in the request body by using
		// the POST method and Content-Type: application/x-www-form-urlencoded header.
		// Get the query from the request
		err := PatchQuery(c, "match[]")
		if err != nil {
			log.Println(err)
			return err
		}

		return nil

	case RouteActiveSeries, RouteLabelNamesCardinality:
		// This can be either a GET or POST request
		// query: selector=<selector>
		err := PatchQuery(c, "selector")
		if err != nil {
			log.Println(err)
			return err
		}

		return nil

	case RouteMetadata:
		// query: metric=<metric name>
		// We cant enforce LBAC here
		return nil

	case RouteRemoteRead:
		// prometheus remote-read API
		return echo.ErrNotImplemented

	case RouteLabelValuesCardinality:
		// label_names[] - required - specifies labels for which cardinality must be provided.
		// selector - optional - specifies PromQL selector that will be used to filter series that must be analyzed.
		return echo.ErrNotImplemented

	default:
		return echo.ErrBadRequest
	}
}

func ParseQuery(query string) (parser.Expr, error) {
	// Parse the query using Loki's syntax parser
	return parser.ParseExpr(query)
}

func PatchQuery(c echo.Context, parameterName string) error {
	// Parse the query
	query := c.Request().URL.Query().Get(parameterName)
	expr, err := ParseQuery(query)
	if err != nil {
		log.Println(err)
		return echo.NewHTTPError(400, "invalid query")
	}

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

	err = EnforceLBAC(expr, enforcedLabels)
	if err != nil {
		log.Printf("failed to enforce LBAC: %v", err)
		return echo.NewHTTPError(400, "invalid query: %v", err)
	}

	// patch the query with the new one
	patchedQuery := c.Request().URL.Query() // this returns a copy and not a reference
	patchedQuery.Set(parameterName, expr.String())
	c.Request().URL.RawQuery = patchedQuery.Encode()

	return nil
}

func EnforceLBAC(e parser.Expr, lbac []*labels.Matcher) error {
	// must check if any labels are already set in the expression
	// if so, we must rewrite them instead of adding them

	selectors := getSelectors(e)

	for _, selector := range selectors {
	OUTER:
		for _, l := range lbac {
			for _, m := range selector.LabelMatchers {
				if m.Name == l.Name {
					// user already set this label, we must rewrite it instead of adding it
					m = l
					continue OUTER
				}
			}
			selector.LabelMatchers = append(selector.LabelMatchers, l)
		}

		log.Println(e.String())
	}
	return nil
}

// getSelector returns the selector from the expression
// an expression can have multiple levels of nesting
func getSelectors(e parser.Expr) []*parser.VectorSelector {
	var selectors []*parser.VectorSelector

	visitor := func(node parser.Node, _ []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if ok {
			selectors = append(selectors, vs)
		}
		return nil
	}

	parser.Inspect(e, visitor)

	return selectors
}
