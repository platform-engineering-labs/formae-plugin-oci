// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package config

import (
	"context"
	"encoding/json"

	"github.com/oracle/oci-go-sdk/v65/common"
	pkgmodel "github.com/platform-engineering-labs/formae/pkg/model"
)

type Config struct {
	Region         string `json:"Region"`
	Profile        string `json:"Profile"`
	ConfigFilePath string `json:"ConfigFilePath"`
}

// ToConfigProvider creates an OCI ConfigurationProvider from the config
func (c *Config) ToConfigProvider(ctx context.Context) (common.ConfigurationProvider, error) {
	if c.ConfigFilePath == "" && c.Profile == "" {
		return common.DefaultConfigProvider(), nil
	}

	if c.ConfigFilePath != "" && c.Profile == "" {
		return common.ConfigurationProviderFromFile(c.ConfigFilePath, "")
	}

	// If Profile is set (with or without ConfigFilePath)
	configPath := c.ConfigFilePath
	if configPath == "" {
		// Use default path with custom profile
		return common.CustomProfileConfigProvider("", c.Profile), nil
	}

	return common.ConfigurationProviderFromFileWithProfile(configPath, c.Profile, "")
}

// FromTarget extracts Config from a Target's raw config
// Deprecated: Use FromTargetConfig instead
func FromTarget(target *pkgmodel.Target) *Config {
	if target == nil || target.Config == nil {
		return &Config{}
	}
	config := &Config{}
	_ = json.Unmarshal(target.Config, config)

	return config
}

// FromTargetConfig extracts Config from raw JSON config
func FromTargetConfig(targetConfig json.RawMessage) *Config {
	if targetConfig == nil {
		return &Config{}
	}
	config := &Config{}
	_ = json.Unmarshal(targetConfig, config)

	return config
}
