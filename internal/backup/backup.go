package backup

import (
	"backup-agent/internal/pkg/encryption"
	"backup-agent/internal/pkg/logger"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
)

// Result represents a request for uploading a file to S3
type Result struct {
	FolderName string // Name of the folder in S3
	FilePath   string // Local file path
	FileName   string // File name
}

// Backup performs the backup operation for all configured databases
func Backup(dbConfigs []Config, encryptor *encryption.Encryptor) ([]Result, error) {
	log := logger.L()
	uploadRequests := make([]Result, 0)

	// Execute database backups
	for _, db := range dbConfigs {
		log.Info("Starting backup for database",
			zap.String("database", db.Name),
			zap.String("type", db.Type),
			zap.String("container", db.Container))

		backupFileName, err := backup(db)
		if err != nil {
			log.Error("Error backing up database",
				zap.String("database", db.Name),
				zap.Error(err))
			return nil, fmt.Errorf("error backing up %s: %v", db.Name, err)
		}
		log.Info("Backup completed for database",
			zap.String("database", db.Name),
			zap.String("backup_file", backupFileName))

		// Resolve the directory path, including handling "~" as the home directory
		absoluteDir, err := resolvePath(db.Directory)
		if err != nil {
			log.Error("Error resolving directory path",
				zap.String("database", db.Name),
				zap.String("directory", db.Directory),
				zap.Error(err))
			return nil, fmt.Errorf("error resolving directory path: %v", err)
		}
		log.Debug("Resolved directory path",
			zap.String("database", db.Name),
			zap.String("original_path", db.Directory),
			zap.String("absolute_path", absoluteDir))

		backupFilePath := filepath.Join(absoluteDir, db.Name, backupFileName)
		uploadFilePath := backupFilePath
		uploadFileName := backupFileName

		// Encrypt the backup file if encryption is enabled
		encryptedPath, err := encryptor.EncryptFile(backupFilePath)
		if err != nil {
			log.Error("Error encrypting backup file",
				zap.String("database", db.Name),
				zap.String("file", backupFilePath),
				zap.Error(err))
			return nil, fmt.Errorf("error encrypting backup file: %v", err)
		}

		if encryptedPath != backupFilePath {
			log.Info("Backup file encrypted",
				zap.String("database", db.Name),
				zap.String("original_path", backupFilePath),
				zap.String("encrypted_path", encryptedPath))
			uploadFilePath = encryptedPath
			uploadFileName = backupFileName + ".enc"
			// Remove the original unencrypted file
			if err := os.Remove(backupFilePath); err != nil {
				log.Warn("Error removing original backup file",
					zap.String("database", db.Name),
					zap.String("file", backupFilePath),
					zap.Error(err))
			} else {
				log.Debug("Original backup file removed",
					zap.String("database", db.Name),
					zap.String("file", backupFilePath))
			}
		}

		log.Debug("Adding upload request",
			zap.String("database", db.Name),
			zap.String("file_path", uploadFilePath),
			zap.String("file_name", uploadFileName))
		uploadRequests = append(uploadRequests, Result{
			FolderName: db.Name,
			FilePath:   uploadFilePath,
			FileName:   uploadFileName,
		})
	}

	return uploadRequests, nil
}
