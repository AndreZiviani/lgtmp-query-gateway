package loki

import (
	"fmt"
	"log"

	"github.com/grafana/loki/v3/pkg/logql/syntax"
	"github.com/prometheus/prometheus/model/labels"
)

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func ParseQuery(query string) (syntax.Expr, error) {

	// Parse the query using Loki's syntax parser
	return syntax.ParseExpr(query)

}

func EnforceLBAC(e syntax.Expr, lbac []*labels.Matcher) error {
	// must check if any labels are already set in the expression
	// if so, we must rewrite them instead of adding them

	var selector syntax.LogSelectorExpr
	switch e.(type) {
	case *syntax.RangeAggregationExpr:
		// this is a metrics query
		tmp := e.(*syntax.RangeAggregationExpr)
		selector, _ = tmp.Selector()
	case syntax.LogSelectorExpr:
		// this is a log query
		selector = e.(syntax.LogSelectorExpr)
	default:
		return fmt.Errorf("unsuported expression type: %T", e)
	}

	pipeline, ok := selector.(*syntax.PipelineExpr)
	if !ok {
		return fmt.Errorf("not a pipeline expression")
	}

	log.Println(e.String())

OUTER:
	for _, l := range lbac {
		for i, m := range pipeline.Left.Mts {
			if m.Name == l.Name {
				// user already set this label, we must rewrite it instead of adding it
				pipeline.Left.Mts[i] = l
				continue OUTER
			}
		}
		// if we didn't find the label, we must add it
		pipeline.Left.AppendMatchers([]*labels.Matcher{l})
	}

	log.Println(e.String())

	return nil
}
