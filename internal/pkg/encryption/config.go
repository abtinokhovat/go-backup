package encryption

// Config holds the encryption configuration
type Config struct {
	Enabled bool   `koanf:"enabled"`
	Key     string `koanf:"key"` // Base64 encoded 32-byte key for AES-256
}

// NewConfig creates a new encryption configuration
func NewConfig(enabled bool, key string) *Config {
	return &Config{
		Enabled: enabled,
		Key:     key,
	}
} 