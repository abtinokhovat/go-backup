package command

import (
	"backup-agent/internal/adapter/s3"
	"backup-agent/internal/config"
	"backup-agent/internal/pkg/logger"
	"context"
	"fmt"
	"path"
	"sort"
	"time"

	"go.uber.org/zap"
)

// DeleteCommand handles the deletion of old backups based on configured rules
type DeleteCommand struct {
	s3Client *s3.S3
	cfg      *config.Config
	dryRun   bool
}

// DeleteStats holds statistics about the deletion operation
type DeleteStats struct {
	TotalFiles     int
	DeletedFiles   int
	RetainedFiles  int
	DeletedSize    int64
	RetainedSize   int64
	OldestRetained time.Time
	NewestRetained time.Time
	// Per database statistics
	DatabaseStats map[string]*DatabaseStats
}

// DatabaseStats holds statistics for a specific database
type DatabaseStats struct {
	TotalFiles     int
	DeletedFiles   int
	RetainedFiles  int
	DeletedSize    int64
	RetainedSize   int64
	OldestRetained time.Time
	NewestRetained time.Time
}

// NewDeleteCommand creates a new DeleteCommand instance
func NewDeleteCommand(s3Client *s3.S3, cfg *config.Config) *DeleteCommand {
	return &DeleteCommand{
		s3Client: s3Client,
		cfg:      cfg,
		dryRun:   false,
	}
}

// WithDryRun enables dry-run mode
func (c *DeleteCommand) WithDryRun(dryRun bool) *DeleteCommand {
	c.dryRun = dryRun
	return c
}

// Execute runs the deletion command based on configured rules
func (c *DeleteCommand) Execute(ctx context.Context) (*DeleteStats, error) {
	log := logger.L()
	stats := &DeleteStats{
		DatabaseStats: make(map[string]*DatabaseStats),
	}

	if !c.cfg.DeletionRules.Enabled {
		log.Info("backup deletion is disabled")
		return stats, nil
	}

	// List all backups (files in the bucket)
	listResp, err := c.s3Client.List(ctx, c.cfg.S3.Bucket, "")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	if len(listResp.Files) == 0 {
		log.Info("no files found to delete")
		return stats, nil
	}

	// Group files by database folder
	dbFiles := make(map[string][]s3.FileInfo)
	for _, file := range listResp.Files {
		// Get the database folder name (first part of the key)
		dbFolder := path.Dir(file.Key)
		dbFiles[dbFolder] = append(dbFiles[dbFolder], file)
	}

	// Process each database folder
	for dbFolder, files := range dbFiles {
		// Initialize database stats
		dbStats := &DatabaseStats{}
		stats.DatabaseStats[dbFolder] = dbStats

		// Sort files by creation time (newest first)
		sort.Slice(files, func(i, j int) bool {
			return files[i].CreatedAt.After(files[j].CreatedAt)
		})

		dbStats.TotalFiles = len(files)
		stats.TotalFiles += len(files)

		// Initialize sets for files to delete and retain
		filesToDelete := make(map[string]s3.FileInfo)
		filesToRetain := make(map[string]s3.FileInfo)

		// Apply time-based rule independently
		if c.cfg.DeletionRules.MaxAgeDays > 0 {
			cutoffTime := time.Now().AddDate(0, 0, -c.cfg.DeletionRules.MaxAgeDays)
			for _, file := range files {
				if file.CreatedAt.Before(cutoffTime) {
					filesToDelete[file.Key] = file
				} else {
					filesToRetain[file.Key] = file
				}
			}
			log.Info("applied time-based retention rule for database",
				zap.String("database", dbFolder),
				zap.Int("max_age_days", c.cfg.DeletionRules.MaxAgeDays),
				zap.Time("cutoff_time", cutoffTime),
				zap.Int("files_to_delete", len(filesToDelete)),
				zap.Int("files_to_retain", len(filesToRetain)))
		}

		// Apply count-based rule independently
		if c.cfg.DeletionRules.MaxCount > 0 {
			// If we have more files than max_count, mark the excess for deletion
			if len(files) > c.cfg.DeletionRules.MaxCount {
				// Keep only the most recent max_count files
				for i, file := range files {
					if i >= c.cfg.DeletionRules.MaxCount {
						filesToDelete[file.Key] = file
						delete(filesToRetain, file.Key)
					} else {
						filesToRetain[file.Key] = file
						delete(filesToDelete, file.Key)
					}
				}
			} else {
				// If we have fewer files than max_count, keep all of them
				for _, file := range files {
					filesToRetain[file.Key] = file
					delete(filesToDelete, file.Key)
				}
			}
			log.Info("applied count-based retention rule for database",
				zap.String("database", dbFolder),
				zap.Int("max_count", c.cfg.DeletionRules.MaxCount),
				zap.Int("files_to_delete", len(filesToDelete)),
				zap.Int("files_to_retain", len(filesToRetain)))
		}

		// Convert maps to slices for final processing
		var filesToDeleteSlice []s3.FileInfo
		for _, file := range filesToDelete {
			filesToDeleteSlice = append(filesToDeleteSlice, file)
		}

		var filesToRetainSlice []s3.FileInfo
		for _, file := range filesToRetain {
			filesToRetainSlice = append(filesToRetainSlice, file)
		}

		// Sort retained files by creation time for statistics
		sort.Slice(filesToRetainSlice, func(i, j int) bool {
			return filesToRetainSlice[i].CreatedAt.After(filesToRetainSlice[j].CreatedAt)
		})

		// Calculate database statistics
		dbStats.DeletedFiles = len(filesToDeleteSlice)
		dbStats.RetainedFiles = len(filesToRetainSlice)
		if len(filesToRetainSlice) > 0 {
			dbStats.OldestRetained = filesToRetainSlice[len(filesToRetainSlice)-1].CreatedAt
			dbStats.NewestRetained = filesToRetainSlice[0].CreatedAt
		}

		for _, file := range filesToDeleteSlice {
			dbStats.DeletedSize += file.Size
		}
		for _, file := range filesToRetainSlice {
			dbStats.RetainedSize += file.Size
		}

		// Update overall statistics
		stats.DeletedFiles += dbStats.DeletedFiles
		stats.RetainedFiles += dbStats.RetainedFiles
		stats.DeletedSize += dbStats.DeletedSize
		stats.RetainedSize += dbStats.RetainedSize

		// Update overall oldest/newest retained times
		if len(filesToRetainSlice) > 0 {
			if stats.OldestRetained.IsZero() || filesToRetainSlice[len(filesToRetainSlice)-1].CreatedAt.Before(stats.OldestRetained) {
				stats.OldestRetained = filesToRetainSlice[len(filesToRetainSlice)-1].CreatedAt
			}
			if stats.NewestRetained.IsZero() || filesToRetainSlice[0].CreatedAt.After(stats.NewestRetained) {
				stats.NewestRetained = filesToRetainSlice[0].CreatedAt
			}
		}

		// Log database deletion summary
		log.Info("deletion summary for database",
			zap.String("database", dbFolder),
			zap.Int("total_files", dbStats.TotalFiles),
			zap.Int("files_to_delete", dbStats.DeletedFiles),
			zap.Int("files_to_retain", dbStats.RetainedFiles),
			zap.Int64("deleted_size_bytes", dbStats.DeletedSize),
			zap.Int64("retained_size_bytes", dbStats.RetainedSize),
			zap.Time("oldest_retained", dbStats.OldestRetained),
			zap.Time("newest_retained", dbStats.NewestRetained),
			zap.Bool("dry_run", c.dryRun))

		if c.dryRun {
			log.Info("dry run mode - no files were actually deleted")
			continue
		}

		// Delete the files for this database
		if err := c.deleteFiles(ctx, filesToDeleteSlice); err != nil {
			return stats, err
		}
	}

	// Log overall deletion summary
	log.Info("overall deletion summary",
		zap.Int("total_files", stats.TotalFiles),
		zap.Int("files_to_delete", stats.DeletedFiles),
		zap.Int("files_to_retain", stats.RetainedFiles),
		zap.Int64("deleted_size_bytes", stats.DeletedSize),
		zap.Int64("retained_size_bytes", stats.RetainedSize),
		zap.Time("oldest_retained", stats.OldestRetained),
		zap.Time("newest_retained", stats.NewestRetained),
		zap.Bool("dry_run", c.dryRun))

	return stats, nil
}

// deleteFiles deletes the specified files and logs the operation
func (c *DeleteCommand) deleteFiles(ctx context.Context, files []s3.FileInfo) error {
	log := logger.L()
	for _, file := range files {
		log.Info("deleting file",
			zap.String("key", file.Key),
			zap.Time("created_at", file.CreatedAt),
			zap.Int64("size", file.Size))

		if err := c.s3Client.Delete(ctx, c.cfg.S3.Bucket, file.Key); err != nil {
			log.Error("failed to delete file",
				zap.String("key", file.Key),
				zap.Error(err))
			return fmt.Errorf("failed to delete file %s: %w", file.Key, err)
		}

		log.Info("successfully deleted file",
			zap.String("key", file.Key))
	}
	return nil
} 