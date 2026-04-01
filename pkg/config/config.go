// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds Cloudflare API configuration.
// API Token can be stored in target config or read from environment variables.
type Config struct {
	// Stored in target config
	AccountId string `json:"AccountId"` // Cloudflare Account ID

	// Can be stored in target config or read from environment
	ApiToken string `json:"ApiToken"` // From CLOUDFLARE_API_TOKEN
}

// FromTargetConfig extracts Cloudflare configuration from target config JSON.
// Credentials are read from environment variables if not in target config.
func FromTargetConfig(targetConfig json.RawMessage) (*Config, error) {
	var cfg Config

	if len(targetConfig) > 0 {
		if err := json.Unmarshal(targetConfig, &cfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal target config: %w", err)
		}
	}

	// Fall back to environment variables
	if cfg.AccountId == "" {
		cfg.AccountId = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	}

	// API Token is always read from environment if not set
	if cfg.ApiToken == "" {
		cfg.ApiToken = os.Getenv("CLOUDFLARE_API_TOKEN")
	}

	return &cfg, nil
}

// Validate checks that required Cloudflare API fields are set
func (c *Config) Validate() error {
	if c.ApiToken == "" {
		return fmt.Errorf("CLOUDFLARE_API_TOKEN environment variable is required")
	}
	return nil
}

// ValidateWithAccountId checks that account ID is also set
func (c *Config) ValidateWithAccountId() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.AccountId == "" {
		return fmt.Errorf("CLOUDFLARE_ACCOUNT_ID is required for this operation")
	}
	return nil
}
