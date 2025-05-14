package config

import (
	"backup-agent/internal/pkg/logger"
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Koanf instance
var k = koanf.New(".")

// Load configuration using Koanf
func Load(filepath string) (*Config, error) {
	if filepath == "" {
		filepath = "config.yaml"
		logger.L().Info("using default configuration config.yml")
	}

	if err := k.Load(file.Provider(filepath), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("error loading config from file: %v", err)
	}

	if err := 	k.Load(env.Provider("BACKUP_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "BACKUP_")), "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading config from env: %v", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %v", err)
	}

	return &cfg, nil
}
