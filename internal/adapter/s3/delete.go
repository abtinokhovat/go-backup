package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"
)

// Delete deletes a file from S3
func (s *S3) Delete(ctx context.Context, bucket, key string) error {
	s.log.Info("Deleting file from S3",
		zap.String("bucket", bucket),
		zap.String("key", key))

	svc := s3.New(s.session)

	_, err := svc.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		s.log.Error("Error deleting file from S3",
			zap.String("bucket", bucket),
			zap.String("key", key),
			zap.Error(err))
		return fmt.Errorf("error deleting file %s: %v", key, err)
	}

	s.log.Info("File deleted successfully",
		zap.String("bucket", bucket),
		zap.String("key", key))
	return nil
}
