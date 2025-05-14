package cmd

import (
	"backup-agent/internal/adapter/s3"
	"backup-agent/internal/command"
	"backup-agent/internal/config"
	"backup-agent/internal/pkg/logger"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	dryRun bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete old backups based on retention rules",
	Long: `Delete old backups based on configured retention rules.
The deletion process follows these rules:
1. MaxAgeDays: Delete backups older than specified days
2. MaxCount: Keep only the specified number of most recent backups
3. Both rules can be applied simultaneously
4. Rules are applied per database folder independently

Example configuration:
deletion_rules:
  enabled: true
  max_age_days: 30
  max_count: 10`,
	RunE: ExecuteDelete,
}

func ExecuteDelete(cmd *cobra.Command, args []string) error {
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
		zap.Bool("dry_run", dryRun),
	)
	log.Info("Starting backup deletion process")

	if !cfg.DeletionRules.Enabled {
		log.Info("Backup deletion is disabled in configuration")
		return nil
	}

	// Initialize S3 client
	s3Client, err := s3.New(s3.Config{
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		Endpoint:  cfg.S3.Endpoint,
		Region:    cfg.S3.Region,
	})
	if err != nil {
		log.Error("Error initializing S3 client", zap.Error(err))
		return fmt.Errorf("error initializing S3 client: %v", err)
	}

	// Create and execute delete command
	deleteCmd := command.NewDeleteCommand(s3Client, cfg).WithDryRun(dryRun)
	stats, err := deleteCmd.Execute(context.Background())
	if err != nil {
		log.Error("Error executing delete command", zap.Error(err))
		return fmt.Errorf("error executing delete command: %v", err)
	}

	// Print summary to console
	fmt.Printf("\nOverall Deletion Summary:\n")
	fmt.Printf("------------------------\n")
	fmt.Printf("Total Files: %d\n", stats.TotalFiles)
	fmt.Printf("Files to Delete: %d\n", stats.DeletedFiles)
	fmt.Printf("Files to Retain: %d\n", stats.RetainedFiles)
	fmt.Printf("Deleted Size: %s\n", formatBytes(stats.DeletedSize))
	fmt.Printf("Retained Size: %s\n", formatBytes(stats.RetainedSize))
	if !stats.OldestRetained.IsZero() {
		fmt.Printf("Oldest Retained: %s\n", stats.OldestRetained.Format(time.RFC3339))
		fmt.Printf("Newest Retained: %s\n", stats.NewestRetained.Format(time.RFC3339))
	}

	// Print per-database statistics
	if len(stats.DatabaseStats) > 0 {
		fmt.Printf("\nPer-Database Statistics:\n")
		fmt.Printf("----------------------\n")

		// Sort database names for consistent output
		dbNames := make([]string, 0, len(stats.DatabaseStats))
		for dbName := range stats.DatabaseStats {
			dbNames = append(dbNames, dbName)
		}
		sort.Strings(dbNames)

		for _, dbName := range dbNames {
			dbStats := stats.DatabaseStats[dbName]
			fmt.Printf("\nDatabase: %s\n", dbName)
			fmt.Printf("%s\n", strings.Repeat("-", len(dbName)+11))
			fmt.Printf("Total Files: %d\n", dbStats.TotalFiles)
			fmt.Printf("Files to Delete: %d\n", dbStats.DeletedFiles)
			fmt.Printf("Files to Retain: %d\n", dbStats.RetainedFiles)
			fmt.Printf("Deleted Size: %s\n", formatBytes(dbStats.DeletedSize))
			fmt.Printf("Retained Size: %s\n", formatBytes(dbStats.RetainedSize))
			if !dbStats.OldestRetained.IsZero() {
				fmt.Printf("Oldest Retained: %s\n", dbStats.OldestRetained.Format(time.RFC3339))
				fmt.Printf("Newest Retained: %s\n", dbStats.NewestRetained.Format(time.RFC3339))
			}
		}
	}

	if dryRun {
		fmt.Printf("\nNote: This was a dry run - no files were actually deleted\n")
	}

	log.Info("Backup deletion process completed successfully")
	return nil
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVarP(&dryRun, "dry-run", "d", false, "Perform a dry run without actually deleting files")
}

// formatBytes formats a byte count into a human-readable string
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
