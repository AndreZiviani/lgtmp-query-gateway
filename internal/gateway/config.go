package gateway

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the root YAML structure
type config struct {
	Destinations map[string]Destination `yaml:"destinations"`
}

// Destination represents a destination with a map of tenants
type Destination struct {
	Tenants  map[string]Tenant `yaml:"tenants"`
	Upstream string            `yaml:"upstream"`
}

// Tenant represents a tenant with a mode and a list of groups
type Tenant struct {
	Mode   string  `yaml:"mode"`
	Groups []Group `yaml:"groups"`
}

// Group represents a group
type Group struct {
	Name string `yaml:"name"`
}

func LoadConfig(path string) (*config, error) {
	// Load the configuration from the specified path
	// and return a Config struct

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
