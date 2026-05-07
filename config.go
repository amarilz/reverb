package main

import (
	"encoding/json"
	"errors"
	"os"
)

type AppConfig struct {
	Voice             string `json:"voice"`
	Rate              string `json:"rate"`
	Pitch             string `json:"pitch"`
	PreferLinuxEngine string `json:"prefer_linux_engine"`
	TestText          string `json:"test_text"`
}

func DefaultConfig() AppConfig {
	return AppConfig{
		Voice:             "",
		Rate:              "",
		Pitch:             "",
		PreferLinuxEngine: "spd-say",
		TestText:          "Hi, this is a test.",
	}
}

func LoadConfig(path string) (AppConfig, error) {
	config := DefaultConfig()

	if path == "" {
		path = "config.json"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}
		return config, err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return config, err
	}
	return config, nil
}

func CreateDefaultConfig(path string) error {
	if path == "" {
		path = "config.json"
	}

	config := DefaultConfig()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
