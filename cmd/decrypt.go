package cmd

import (
	"backup-agent/internal/config"
	"backup-agent/internal/pkg/encryption"
	"backup-agent/internal/pkg/logger"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt [file]",
	Short: "Decrypt an encrypted backup file",
	Long:  `Decrypt an encrypted backup file using the encryption key from the configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		encryptedFile := args[0]

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("error loading configuration: %v", err)
		}

		// Initialize logger
		if err := logger.Init(cfg.LogLevel); err != nil {
			return fmt.Errorf("error initializing logger: %v", err)
		}
		defer logger.Sync()

		log := logger.L().With(
			zap.String("config_path", configPath),
			zap.String("encrypted_file", encryptedFile),
		)
		log.Info("Starting decryption process")

		// Initialize encryptor
		encryptor, err := encryption.NewEncryptor(cfg.Encryption)
		if err != nil {
			log.Error("Error initializing encryptor", zap.Error(err))
			return fmt.Errorf("error initializing encryptor: %v", err)
		}

		// Decrypt the file
		decryptedPath, err := encryptor.DecryptFile(encryptedFile)
		if err != nil {
			log.Error("Error decrypting file", zap.Error(err))
			return fmt.Errorf("error decrypting file: %v", err)
		}

		// Get the output filename without the .enc extension
		outputFile := filepath.Base(encryptedFile)
		if filepath.Ext(outputFile) == ".enc" {
			outputFile = outputFile[:len(outputFile)-4]
		}

		log.Info("File decrypted successfully",
			zap.String("encrypted_file", encryptedFile),
			zap.String("decrypted_file", decryptedPath))
		fmt.Printf("File decrypted successfully. Decrypted file: %s\n", decryptedPath)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(decryptCmd)
}
