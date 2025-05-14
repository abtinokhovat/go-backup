package config

import (
	"backup-agent/internal/adapter/s3"
	"backup-agent/internal/backup"
	"backup-agent/internal/pkg/encryption"
	"backup-agent/internal/pkg/logger"
)

// DeletionRules defines rules for automatic backup deletion
type DeletionRules struct {
	// MaxAgeDays defines the maximum age of backups in days before deletion
	MaxAgeDays int `koanf:"max_age_days"`
	// MaxCount defines the maximum number of backups to keep
	MaxCount int `koanf:"max_count"`
	// Enabled determines if automatic deletion is enabled
	Enabled bool `koanf:"enabled"`
}

// Config represents the application configuration
type Config struct {
	LogLevel logger.LogLevel `koanf:"log_level"`
	Upload   struct {
		Enabled bool `koanf:"enabled"`
	} `koanf:"upload"`
	S3         s3.Config          `koanf:"s3"`
	Encryption *encryption.Config `koanf:"encryption"`
	DBConfigs  []backup.Config    `koanf:"db_configs"`
	DeletionRules DeletionRules    `koanf:"deletion_rules"`
}
