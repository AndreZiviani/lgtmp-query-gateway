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
	Server       Server                 `yaml:"server"`
	Profiles     map[string]Profile     `yaml:"profiles"`
	Destinations map[string]Destination `yaml:"destinations"`
}

// Server represents the server configuration
type Server struct {
	Port int `yaml:"port" validate:"required"`
}

// Profiles represents the profiles configuration
type Profile struct {
	Groups         []string `yaml:"groups" validate:"required"`
	EnforcedLabels []string `yaml:"enforcedLabels"`
	Matchers       []*labels.Matcher
}

// Destination represents a destination with a map of tenants
type Destination struct {
	Type           StackType         `yaml:"type" validate:"required"`
	Upstream       string            `yaml:"upstream" validate:"required"`
	AllowUndefined bool              `yaml:"allowUndefined"`
	Admins         []string          `yaml:"admins"`
	Tenants        map[string]Tenant `yaml:"tenants"`
}

// Tenant represents a tenant with a mode and a list of groups
type Tenant struct {
	Mode     Mode     `yaml:"mode" validate:"required,oneof=allowlist denylist"`
	Profiles []string `yaml:"profiles"`
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

func (c *Config) UnmarshalYAML(unmarshal func(any) error) error {
	// create an alias to avoid infinite recursion
	type Alias Config
	var aux Alias

	if err := unmarshal(&aux); err != nil {
		return err
	}

	c.Destinations = make(map[string]Destination)

	for name, dest := range aux.Destinations {
		if _, ok := c.Destinations[name]; ok {
			return fmt.Errorf("duplicate destination hostname: %s", name)
		}

		c.Destinations[name] = dest
	}

	c.Profiles = aux.Profiles
	c.Server = aux.Server

	return nil
}

func (p *Profile) UnmarshalYAML(unmarshal func(any) error) error {
	// create an alias to avoid infinite recursion
	type Alias Profile
	var aux Alias

	if err := unmarshal(&aux); err != nil {
		return err
	}

	//TODO: find a way to copy values from aux to p automatically
	p.Groups = aux.Groups
	p.EnforcedLabels = aux.EnforcedLabels
	p.Matchers = make([]*labels.Matcher, 0, len(aux.EnforcedLabels))

	for _, matcher := range aux.EnforcedLabels {
		str := matcher
		if !strings.HasPrefix(str, "{") {
			str = "{" + str + "}"
		}
		m, err := parser.ParseMetricSelector(str)
		if err != nil {
			log.Printf("failed to parse matcher %s: %v", matcher, err)
			return fmt.Errorf("failed to parse matcher %s: %w", matcher, err)
		}
		p.Matchers = append(p.Matchers, m...)
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
