package s3

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"go.uber.org/zap"
)

// UploadRequest represents a request for uploading content to S3
type UploadRequest struct {
	FolderName string    // Name of the folder in S3
	FileName   string    // File name
	Content    io.Reader // Content to upload
}

// Upload uploads content to S3 and returns its URL
func (s *S3) Upload(bucket string, req UploadRequest) (string, error) {
	key := fmt.Sprintf("%s/%s", req.FolderName, req.FileName)
	s.log.Info("Starting S3 upload process",
		zap.String("bucket", bucket),
		zap.String("folder", req.FolderName),
		zap.String("file", req.FileName),
		zap.String("key", key))

	output, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   req.Content,
	})
	if err != nil {
		s.log.Error("Error during S3 upload",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return "", fmt.Errorf("error uploading %s: %v", req.FileName, err)
	}

	s.log.Info("Content uploaded successfully",
		zap.String("key", key),
		zap.String("url", output.Location))
	
	return output.Location, nil
}

// Upload uploads multiple files to S3
func (s *S3) UploadMultiple(bucket string, requests []UploadRequest) error {
	s.log.Info("Starting S3 upload process",
		zap.String("bucket", bucket),
		zap.Int("file_count", len(requests)))

	for _, req := range requests {
		key := fmt.Sprintf("%s/%s", req.FolderName, req.FileName)
		s.log.Debug("Processing upload request",
			zap.String("folder", req.FolderName),
			zap.String("file", req.FileName),
			zap.String("key", key))

		if err := s.uploadFile(bucket, req.Content, key); err != nil {
			s.log.Error("Error uploading file",
				zap.String("file", req.FileName),
				zap.String("key", key),
				zap.Error(err))
			return fmt.Errorf("error uploading %s: %v", req.FileName, err)
		}
		s.log.Info("File uploaded successfully",
			zap.String("file", req.FileName),
			zap.String("key", key))
	}

	s.log.Info("All files uploaded successfully",
		zap.String("bucket", bucket),
		zap.Int("file_count", len(requests)))
	return nil
}

// uploadFile uploads a single file to S3
func (s *S3) uploadFile(bucket string, content io.Reader, key string) error {
	s.log.Debug("Starting S3 upload",
		zap.String("bucket", bucket),
		zap.String("key", key))

	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   content,
	})
	if err != nil {
		s.log.Error("Error during S3 upload",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	s.log.Debug("S3 upload completed",
		zap.String("bucket", bucket),
		zap.String("key", key))
	return nil
}