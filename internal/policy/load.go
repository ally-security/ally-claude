package policy

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

func Load(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	if c.ID == "" {
		return fmt.Errorf("policy.id is required")
	}
	if c.InferenceProvider == "" {
		return fmt.Errorf("policy.inferenceProvider is required")
	}
	for i, b := range c.Plugins {
		if b.Name == "" || b.Source == "" {
			return fmt.Errorf("plugins[%d]: name and source are required", i)
		}
	}
	for i, b := range c.Extensions {
		if b.Name == "" || b.Source == "" {
			return fmt.Errorf("extensions[%d]: name and source are required", i)
		}
	}
	for i, conn := range c.Connectors {
		if conn.Name == "" {
			return fmt.Errorf("connectors[%d]: name is required", i)
		}
	}
	return nil
}
