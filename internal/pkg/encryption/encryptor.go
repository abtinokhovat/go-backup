package encryption

import (
	"backup-agent/internal/pkg/logger"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
)

// Encryptor handles file encryption and decryption
type Encryptor struct {
	config *Config
	key    []byte // Decoded key
	log    *zap.Logger
}

// NewEncryptor creates a new encryptor instance
func NewEncryptor(config *Config) (*Encryptor, error) {
	log := logger.L().With(zap.Bool("encryption_enabled", config.Enabled))
	log.Debug("Initializing encryptor")

	if !config.Enabled {
		log.Info("Encryption is disabled")
		return &Encryptor{
			config: config,
			log:    log,
		}, nil
	}

	// Decode the base64 key
	key, err := base64.StdEncoding.DecodeString(config.Key)
	if err != nil {
		log.Error("Error decoding encryption key", zap.Error(err))
		return nil, fmt.Errorf("error decoding encryption key: %v", err)
	}

	// Verify key length
	if len(key) != 32 {
		log.Error("Invalid key length",
			zap.Int("expected", 32),
			zap.Int("got", len(key)))
		return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d bytes", len(key))
	}

	log.Debug("Encryptor initialized successfully")
	return &Encryptor{
		config: config,
		key:    key,
		log:    log,
	}, nil
}

// EncryptFile encrypts a file using AES-256-GCM and returns the path to the encrypted file
func (e *Encryptor) EncryptFile(inputPath string) (string, error) {
	if !e.config.Enabled {
		return inputPath, nil
	}

	// Read the input file
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		e.log.Error("Error reading file",
			zap.String("file", inputPath),
			zap.Error(err))
		return "", fmt.Errorf("error reading file: %v", err)
	}

	// Generate a random nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		e.log.Error("Error generating nonce", zap.Error(err))
		return "", fmt.Errorf("error generating nonce: %v", err)
	}

	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		e.log.Error("Error creating cipher", zap.Error(err))
		return "", fmt.Errorf("error creating cipher: %v", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		e.log.Error("Error creating GCM", zap.Error(err))
		return "", fmt.Errorf("error creating GCM: %v", err)
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// Create output file path
	outputPath := inputPath + ".enc"

	// Write the encrypted data
	if err := os.WriteFile(outputPath, ciphertext, 0644); err != nil {
		e.log.Error("Error writing encrypted file",
			zap.String("file", outputPath),
			zap.Error(err))
		return "", fmt.Errorf("error writing encrypted file: %v", err)
	}

	e.log.Info("File encrypted successfully",
		zap.String("output_file", outputPath))
	return outputPath, nil
}

// DecryptFile decrypts an encrypted file using AES-256-GCM
func (e *Encryptor) DecryptFile(inputPath string) (string, error) {
	if !e.config.Enabled {
		return inputPath, nil
	}

	// Read the encrypted file
	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		e.log.Error("Error reading encrypted file",
			zap.String("file", inputPath),
			zap.Error(err))
		return "", fmt.Errorf("error reading encrypted file: %v", err)
	}

	// Extract nonce
	if len(ciphertext) < 12 {
		e.log.Error("Ciphertext too short",
			zap.Int("length", len(ciphertext)),
			zap.Int("minimum", 12))
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce := ciphertext[:12]
	ciphertext = ciphertext[12:]

	// Create cipher block
	block, err := aes.NewCipher(e.key)
	if err != nil {
		e.log.Error("Error creating cipher", zap.Error(err))
		return "", fmt.Errorf("error creating cipher: %v", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		e.log.Error("Error creating GCM", zap.Error(err))
		return "", fmt.Errorf("error creating GCM: %v", err)
	}

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		e.log.Error("Error decrypting data", zap.Error(err))
		return "", fmt.Errorf("error decrypting data: %v", err)
	}

	// Create output file path
	outputPath := strings.TrimSuffix(inputPath, ".enc")

	// Write the decrypted data
	if err := os.WriteFile(outputPath, plaintext, 0644); err != nil {
		e.log.Error("Error writing decrypted file",
			zap.String("file", outputPath),
			zap.Error(err))
		return "", fmt.Errorf("error writing decrypted file: %v", err)
	}

	e.log.Info("File decrypted successfully",
		zap.String("input_file", inputPath),
		zap.String("output_file", outputPath))
	return outputPath, nil
} 