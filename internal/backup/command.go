package backup

import (
	"backup-agent/internal/pkg/logger"
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	MySQL      = "mysql"
	PostgreSQL = "postgresql"
	InfluxDB   = "influxdb"
)

// Config represents a database configuration
type Config struct {
	Name      string `koanf:"name"`
	Type      string `koanf:"type"`
	Host      string `koanf:"host"`
	Port      int    `koanf:"port"`
	User      string `koanf:"user"`
	Password  string `koanf:"password"`
	Directory string `koanf:"directory"`
	Container string `koanf:"container,omitempty"`
}

func NewDBBackupCommand(db Config, backupFilePath string) (*exec.Cmd, error) {
	log := logger.L().With(
		zap.String("database", db.Name),
		zap.String("type", db.Type),
		zap.String("backup_path", backupFilePath),
	)

	baseCmd := ""

	switch db.Type {
	// mysql dump command
	case MySQL:
		baseCmd = fmt.Sprintf(`mysqldump -u %s --password="%s" --no-tablespaces %s > %s`,
			db.User, db.Password, db.Name, backupFilePath)
		log.Debug("Generated MySQL backup command", zap.String("command", strings.Replace(baseCmd, db.Password, "****", -1)))

	// postgresql dump command
	case PostgreSQL:
		baseCmd = fmt.Sprintf(`PGPASSWORD="%s" pg_dump -U %s -h %s%d %s > %s`,
			db.Password, db.User, db.Host, db.Port, db.Name, backupFilePath)
		log.Debug("Generated PostgreSQL backup command", zap.String("command", strings.Replace(baseCmd, db.Password, "****", -1)))

	// influxdb backup command
	case InfluxDB:
		// For InfluxDB, we need to create a directory for the backup
		backupDir := filepath.Dir(backupFilePath)
		// InfluxDB backup command requires a directory, not a file
		baseCmd = fmt.Sprintf(`influx backup -t %s -h %s:%d -o %s %s`,
			db.Password, // token
			db.Host,
			db.Port,
			db.User, // org
			backupDir)
		log.Debug("Generated InfluxDB backup command", zap.String("command", strings.Replace(baseCmd, db.Password, "****", -1)))

	default:
		log.Error("Unsupported database type", zap.String("type", string(db.Type)))
		return nil, fmt.Errorf("unsupported database type: %s", db.Type)
	}

	if db.Container != "" {
		baseCmd = fmt.Sprintf(`docker exec %s %s`, db.Container, baseCmd)
		log.Debug("Added container execution wrapper", zap.String("container", db.Container))
	}

	return exec.Command("sh", "-c", baseCmd), nil
}

func backup(db Config) (string, error) {
	log := logger.L().With(
		zap.String("database", db.Name),
		zap.String("type", db.Type),
	)

	backupFileName := fmt.Sprintf("%s_%s", db.Name, time.Now().Format("2006-01-02-15-04-05"))
	if db.Type == InfluxDB {
		backupFileName = backupFileName + ".influx"
	} else {
		backupFileName = backupFileName + ".sql"
	}

	// Resolve the directory path, including handling "~" as the home directory
	absoluteDir, err := resolvePath(db.Directory)
	if err != nil {
		log.Error("Error resolving directory path",
			zap.String("directory", db.Directory),
			zap.Error(err))
		return "", fmt.Errorf("error resolving directory path: %v", err)
	}

	backupFilePath := fmt.Sprintf("%s/%s/%s", absoluteDir, db.Name, backupFileName)

	// Ensure that the backup directory exists
	dir := filepath.Dir(backupFilePath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Error("Failed to create backup directory",
			zap.String("directory", dir),
			zap.Error(err))
		return "", fmt.Errorf("failed to create backup directory: %v", err)
	}

	// Get the appropriate backup command based on database type
	cmd, err := NewDBBackupCommand(db, backupFilePath)
	if err != nil {
		log.Error("Error creating backup command", zap.Error(err))
		return "", fmt.Errorf("error creating backup command: %v", err)
	}

	// For MySQL, check if mysqldump is available when not using a container
	if db.Type == MySQL && db.Container == "" {
		if err := checkMariadbDumpAvailability(); err != nil {
			log.Error("MySQL dump not available", zap.Error(err))
			return "", err
		}
	}

	// For InfluxDB, check if influx CLI is available when not using a container
	if db.Type == InfluxDB && db.Container == "" {
		if err := checkInfluxAvailability(); err != nil {
			log.Error("Influx CLI not available", zap.Error(err))
			return "", err
		}
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the backup command
	log.Info("Executing backup command")
	err = cmd.Run()
	if err != nil {
		log.Error("Error running backup command",
			zap.Error(err),
			zap.String("stderr", stderr.String()))
		return "", fmt.Errorf("error running backup command: %v, error message: %s", err, stderr.String())
	}

	log.Info("Backup command executed successfully")
	return backupFileName, nil
}

// Helper function to resolve paths, including expanding "~" to the home directory
func resolvePath(path string) (string, error) {
	log := logger.L().With(zap.String("path", path))

	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Error("Could not resolve home directory", zap.Error(err))
			return "", fmt.Errorf("could not resolve home directory: %v", err)
		}
		return filepath.Join(homeDir, path[1:]), nil
	}

	return path, nil
}

// checkInfluxAvailability checks if influx CLI is available on the system
func checkInfluxAvailability() error {
	log := logger.L()
	cmd := exec.Command("sh", "-c", "command -v influx")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Error("InfluxDB CLI not found", zap.Error(err), zap.String("stderr", stderr.String()))
		return fmt.Errorf("influx CLI is not installed or available on the system: %s", stderr.String())
	}

	return nil
}

// checkMariadbDumpAvailability checks if mariadb-dump is available on the system
func checkMariadbDumpAvailability() error {
	log := logger.L()
	cmd := exec.Command("sh", "-c", "command -v mysqldump")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Error("MySQL dump not found", zap.Error(err), zap.String("stderr", stderr.String()))
		return fmt.Errorf("mariadb-dump is not installed or available on the system: %s", stderr.String())
	}

	return nil
}
