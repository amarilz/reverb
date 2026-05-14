package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
)

// AppConfig holds all user-configurable settings for reverb.
type AppConfig struct {
	Voice             string `json:"voice"`
	Rate              string `json:"rate"`
	Pitch             string `json:"pitch"`
	PreferLinuxEngine string `json:"prefer_linux_engine"`
	TestText          string `json:"test_text"`
}

// DefaultConfig returns a safe baseline configuration.
func DefaultConfig() AppConfig {
	return AppConfig{
		Voice:             "",
		Rate:              "",
		Pitch:             "",
		PreferLinuxEngine: "spd-say",
		TestText:          "Hi, this is a test.",
	}
}

// LoadConfig reads the config file at path, falling back to defaults if the
// file does not exist. Returns an error only on I/O or JSON parse failures.
func LoadConfig(path string) (AppConfig, error) {
	if path == "" {
		path = "config.json"
	}

	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %q: %w", path, err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %q: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return cfg, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

// CreateDefaultConfig writes a default config.json to path.
func CreateDefaultConfig(path string) error {
	if path == "" {
		path = "config.json"
	}

	data, err := json.MarshalIndent(DefaultConfig(), "", "  ")
	if err != nil {
		return fmt.Errorf("serializing default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config %q: %w", path, err)
	}

	return nil
}

// validate checks that numeric fields, if set, are actually numeric.
func (c AppConfig) validate() error {
	if c.Rate != "" {
		if _, err := strconv.ParseFloat(c.Rate, 64); err != nil {
			return fmt.Errorf("rate %q is not a valid number", c.Rate)
		}
	}
	if c.Pitch != "" {
		if _, err := strconv.ParseFloat(c.Pitch, 64); err != nil {
			return fmt.Errorf("pitch %q is not a valid number", c.Pitch)
		}
	}
	return nil
}
