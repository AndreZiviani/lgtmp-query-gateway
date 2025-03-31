package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"gopkg.in/yaml.v3"
)

const (
	ModeAllowList Mode = "allowlist"
	ModeDenyList  Mode = "denylist"

	StackLoki       StackType = "loki"
	StackPrometheus StackType = "prometheus"
	StackMimir      StackType = "mimir"
	StackTempo      StackType = "tempo"
	StackPyroscope  StackType = "pyroscope"
)

type Mode string
type StackType string

// Config represents the root YAML structure
type Config struct {
	Destinations map[string]Destination `yaml:",inline"`
}

// Destination represents a destination with a map of tenants
type Destination struct {
	Type           StackType         `yaml:"type" validate:"required"`
	Upstream       string            `yaml:"upstream" validate:"required"`
	AllowUndefined bool              `yaml:"allowUndefined"`
	Tenants        map[string]Tenant `yaml:"tenants"`
}

// Tenant represents a tenant with a mode and a list of groups
type Tenant struct {
	Mode   Mode    `yaml:"mode" validate:"required,oneof=allowlist denylist"`
	Groups []Group `yaml:"groups"`
}

// Group represents a group
type Group struct {
	Name     string   `yaml:"name" validate:"required"`
	LBAC     []string `yaml:"enforcedLabels"`
	Matchers []*labels.Matcher
}

func LoadConfig(path string) (*Config, error) {
	// Load the configuration from the specified path
	// and return a Config struct

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	validate := validator.New(validator.WithRequiredStructEnabled())
	err = validate.Struct(config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (g *Group) UnmarshalYAML(unmarshal func(any) error) error {
	// create an alias to avoid infinite recursion
	type Alias Group
	var aux Alias

	if err := unmarshal(&aux); err != nil {
		return err
	}

	//TODO: find a way to copy values from aux to g automatically
	g.LBAC = aux.LBAC
	g.Name = aux.Name
	g.Matchers = make([]*labels.Matcher, 0, len(aux.LBAC))

	for _, matcher := range aux.LBAC {
		str := matcher
		if !strings.HasPrefix(str, "{") {
			str = "{" + str + "}"
		}
		m, err := parser.ParseMetricSelector(str)
		if err != nil {
			log.Printf("failed to parse matcher %s: %v", matcher, err)
			return fmt.Errorf("failed to parse matcher %s: %w", matcher, err)
		}
		g.Matchers = append(g.Matchers, m...)
	}
	return nil
}

func (m *Mode) UnmarshalYAML(unmarshal func(any) error) error {
	var mode string
	if err := unmarshal(&mode); err != nil {
		return err
	}
	switch mode {
	case string(ModeAllowList):
		*m = ModeAllowList
	case string(ModeDenyList):
		*m = ModeDenyList
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
	return nil
}

func (s *StackType) UnmarshalYAML(unmarshal func(any) error) error {
	var stack string
	if err := unmarshal(&stack); err != nil {
		return err
	}
	switch stack {
	case string(StackLoki):
		*s = StackLoki
	case string(StackPrometheus):
		*s = StackPrometheus
	case string(StackMimir):
		*s = StackMimir
	case string(StackTempo):
		*s = StackTempo
	case string(StackPyroscope):
		*s = StackPyroscope
	default:
		return fmt.Errorf("invalid stack type: %s", stack)
	}
	return nil
}
