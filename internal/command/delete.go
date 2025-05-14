package command

import (
	"backup-agent/internal/adapter/s3"
	"backup-agent/internal/config"
	"backup-agent/internal/pkg/logger"
	"context"
	"fmt"
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
	stats := &DeleteStats{}

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

	stats.TotalFiles = len(listResp.Files)

	// Sort files by creation time (newest first)
	sort.Slice(listResp.Files, func(i, j int) bool {
		return listResp.Files[i].CreatedAt.After(listResp.Files[j].CreatedAt)
	})

	var filesToDelete []s3.FileInfo
	var filesToRetain []s3.FileInfo

	// Apply count-based rule
	if c.cfg.DeletionRules.MaxCount > 0 {
		if len(listResp.Files) > c.cfg.DeletionRules.MaxCount {
			filesToDelete = append(filesToDelete, listResp.Files[c.cfg.DeletionRules.MaxCount:]...)
			filesToRetain = listResp.Files[:c.cfg.DeletionRules.MaxCount]
		} else {
			filesToRetain = listResp.Files
		}
		log.Info("applied count-based retention rule",
			zap.Int("max_count", c.cfg.DeletionRules.MaxCount),
			zap.Int("files_to_delete", len(filesToDelete)),
			zap.Int("files_to_retain", len(filesToRetain)))
	}

	// Apply time-based rule
	if c.cfg.DeletionRules.MaxAgeDays > 0 {
		cutoffTime := time.Now().AddDate(0, 0, -c.cfg.DeletionRules.MaxAgeDays)
		var timeBasedDelete []s3.FileInfo
		var timeBasedRetain []s3.FileInfo

		// If we already have files to delete from count-based rule, use those
		filesToCheck := filesToDelete
		if len(filesToDelete) == 0 {
			filesToCheck = listResp.Files
		}

		for _, file := range filesToCheck {
			if file.CreatedAt.Before(cutoffTime) {
				timeBasedDelete = append(timeBasedDelete, file)
			} else {
				timeBasedRetain = append(timeBasedRetain, file)
			}
		}

		// Update our lists based on time-based rule
		if len(filesToDelete) == 0 {
			filesToDelete = timeBasedDelete
			filesToRetain = timeBasedRetain
		} else {
			// Intersect the count-based and time-based rules
			timeBasedDeleteMap := make(map[string]bool)
			for _, f := range timeBasedDelete {
				timeBasedDeleteMap[f.Key] = true
			}
			var newFilesToDelete []s3.FileInfo
			for _, f := range filesToDelete {
				if timeBasedDeleteMap[f.Key] {
					newFilesToDelete = append(newFilesToDelete, f)
				} else {
					filesToRetain = append(filesToRetain, f)
				}
			}
			filesToDelete = newFilesToDelete
		}

		log.Info("applied time-based retention rule",
			zap.Int("max_age_days", c.cfg.DeletionRules.MaxAgeDays),
			zap.Time("cutoff_time", cutoffTime),
			zap.Int("files_to_delete", len(filesToDelete)),
			zap.Int("files_to_retain", len(filesToRetain)))
	}

	// Calculate statistics
	stats.DeletedFiles = len(filesToDelete)
	stats.RetainedFiles = len(filesToRetain)
	if len(filesToRetain) > 0 {
		stats.OldestRetained = filesToRetain[len(filesToRetain)-1].CreatedAt
		stats.NewestRetained = filesToRetain[0].CreatedAt
	}

	for _, file := range filesToDelete {
		stats.DeletedSize += file.Size
	}
	for _, file := range filesToRetain {
		stats.RetainedSize += file.Size
	}

	// Log deletion summary
	log.Info("deletion summary",
		zap.Int("total_files", stats.TotalFiles),
		zap.Int("files_to_delete", stats.DeletedFiles),
		zap.Int("files_to_retain", stats.RetainedFiles),
		zap.Int64("deleted_size_bytes", stats.DeletedSize),
		zap.Int64("retained_size_bytes", stats.RetainedSize),
		zap.Time("oldest_retained", stats.OldestRetained),
		zap.Time("newest_retained", stats.NewestRetained),
		zap.Bool("dry_run", c.dryRun))

	if c.dryRun {
		log.Info("dry run mode - no files were actually deleted")
		return stats, nil
	}

	// Delete the files
	if err := c.deleteFiles(ctx, filesToDelete); err != nil {
		return stats, err
	}

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