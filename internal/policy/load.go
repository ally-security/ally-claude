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
	if c.InferenceProvider == "bedrock" {
		if c.InferenceBedrockRegion == "" {
			return fmt.Errorf("inferenceBedrockRegion is required when inferenceProvider=bedrock")
		}
		auths := 0
		if c.InferenceBedrockBearerToken != "" {
			auths++
		}
		if c.InferenceBedrockProfile != "" {
			auths++
		}
		if c.InferenceBedrockSsoStartUrl != "" {
			auths++
		}
		if c.InferenceCredentialHelper != "" {
			auths++
		}
		if auths == 0 {
			return fmt.Errorf("bedrock: provide one of inferenceBedrockBearerToken, inferenceBedrockProfile, inferenceBedrockSso*, or inferenceCredentialHelper")
		}
		if auths > 1 {
			return fmt.Errorf("bedrock: only one auth path may be set; got %d", auths)
		}
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
