package config

import (
	"fmt"
	"os"

	"frigate-stitcher/internal/models"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads the configuration from a file
func LoadConfig(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg models.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults if needed
	if cfg.MQTT.Topic == "" {
		cfg.MQTT.Topic = "frigate/events"
	}
	if cfg.MQTT.ClientID == "" {
		cfg.MQTT.ClientID = "frigate-stitcher"
	}

	return &cfg, nil
}
