package cmd

import (
	"backup-agent/internal/adapter/s3"
	"backup-agent/internal/backup"
	"backup-agent/internal/config"
	"backup-agent/internal/pkg/encryption"
	"backup-agent/internal/pkg/logger"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Perform database backups",
	Long:  `Perform backups of configured databases with optional encryption and S3 upload.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")

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
		)
		log.Info("Starting backup process")

		// Initialize encryptor
		encryptor, err := encryption.NewEncryptor(cfg.Encryption)
		if err != nil {
			log.Error("Error initializing encryptor", zap.Error(err))
			return fmt.Errorf("error initializing encryptor: %v", err)
		}
		log.Debug("Encryptor initialized", zap.Bool("encryption_enabled", cfg.Encryption.Enabled))

		log.Info("DBConfigs", zap.Any("DBConfigs", cfg.DBConfigs))

		// Perform database backups
		uploadRequests, err := backup.Backup(cfg.DBConfigs, encryptor)
		if err != nil {
			log.Error("Error backing up databases", zap.Error(err))
			return fmt.Errorf("error backing up databases: %v", err)
		}

		// Handle S3 upload if enabled
		if cfg.Upload.Enabled {
			log.Info("S3 upload enabled, initializing S3 adapter")
			s3Adapter, err := s3.New(s3.Config{
				AccessKey: cfg.S3.AccessKey,
				SecretKey: cfg.S3.SecretKey,
				Endpoint:  cfg.S3.Endpoint,
				Region:    "default",
			})
			if err != nil {
				log.Error("Error initializing S3 adapter", zap.Error(err))
				return fmt.Errorf("error initializing S3 adapter: %v", err)
			}

			// Convert upload requests to S3 adapter format
			s3Requests := make([]s3.UploadRequest, len(uploadRequests))
			for i, req := range uploadRequests {
				// Open the file for reading
				file, err := os.Open(req.FilePath)
				if err != nil {
					log.Error("Error opening file for upload",
						zap.String("file", req.FilePath),
						zap.Error(err))
					return fmt.Errorf("error opening file %s: %v", req.FilePath, err)
				}
				defer file.Close()

				s3Requests[i] = s3.UploadRequest{
					FolderName: req.FolderName,
					FileName:   req.FileName,
					Content:    file,
				}
			}

			// Upload files to S3
			log.Info("Starting S3 upload", zap.Int("file_count", len(s3Requests)))
			if err := s3Adapter.UploadMultiple(cfg.S3.Bucket, s3Requests); err != nil {
				log.Error("Error uploading to S3", zap.Error(err))
				return fmt.Errorf("error uploading to S3: %v", err)
			}
			log.Info("Successfully uploaded backups to S3")
		} else {
			log.Info("S3 upload is disabled, backups are stored locally only")
		}

		log.Info("Backup process completed successfully")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
}
