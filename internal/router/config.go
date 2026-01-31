package router

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RouteConfig defines a single route in the YAML config.
type RouteConfig struct {
	Path     string            `yaml:"path"`
	Headers  map[string]string `yaml:"headers,omitempty"`
	Backends []string          `yaml:"backends"`
}

// GatewayConfig is the top-level YAML configuration.
type GatewayConfig struct {
	Routes []RouteConfig `yaml:"routes"`
}

// LoadConfig reads and parses a YAML config file.
func LoadConfig(path string) (*GatewayConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	return ParseConfig(data)
}

// ParseConfig parses YAML bytes into a GatewayConfig.
func ParseConfig(data []byte) (*GatewayConfig, error) {
	var cfg GatewayConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateConfig checks that the config is semantically valid.
func validateConfig(cfg *GatewayConfig) error {
	if len(cfg.Routes) == 0 {
		return fmt.Errorf("config must have at least one route")
	}

	for i, route := range cfg.Routes {
		if route.Path == "" {
			return fmt.Errorf("route %d: path cannot be empty", i)
		}
		if len(route.Backends) == 0 {
			return fmt.Errorf("route %d (%s): must have at least one backend", i, route.Path)
		}
	}

	return nil
}
