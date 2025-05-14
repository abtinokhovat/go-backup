package s3

import (
	"backup-agent/internal/pkg/logger"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

// Config holds the configuration for S3 adapter
type Config struct {
	AccessKey string `koanf:"access_key"`
	SecretKey string `koanf:"secret_key"`
	Endpoint  string `koanf:"endpoint"`
	Region    string `koanf:"region"`
	Bucket    string `koanf:"bucket"`
}

// S3 represents an S3 storage adapter
type S3 struct {
	config   Config
	uploader *s3manager.Uploader
	session  *session.Session
	log      *zap.Logger
}

// ListResponse represents the response from listing files in S3
type ListResponse struct {
	Files []FileInfo // List of file information
	Error error      // Any error that occurred during listing
}

// FileInfo represents information about a file in S3
type FileInfo struct {
	Key       string
	CreatedAt time.Time
	Size      int64
}

// New creates a new S3 adapter instance
func New(config Config) (*S3, error) {
	log := logger.L().With(
		zap.String("endpoint", config.Endpoint),
		zap.String("region", config.Region),
	)
	log.Debug("Initializing S3 adapter")

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(config.AccessKey, config.SecretKey, ""),
		Region:      aws.String(config.Region),
		Endpoint:    aws.String(config.Endpoint),
	})
	if err != nil {
		log.Error("Error creating AWS session", zap.Error(err))
		return nil, fmt.Errorf("error creating session: %v", err)
	}

	log.Debug("AWS session created successfully")
	return &S3{
		config:   config,
		uploader: s3manager.NewUploader(sess),
		session:  sess,
		log:      log,
	}, nil
}