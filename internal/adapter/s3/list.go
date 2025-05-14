package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"go.uber.org/zap"
)

// List lists files in a specific folder/prefix in S3
func (s *S3) List(ctx context.Context, bucket, prefix string) (*ListResponse, error) {
	s.log.Info("Listing files in S3",
		zap.String("bucket", bucket),
		zap.String("prefix", prefix))

	svc := s3.New(s.session)
	var files []FileInfo

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	err := svc.ListObjectsV2PagesWithContext(ctx, input, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			files = append(files, FileInfo{
				Key:       *obj.Key,
				CreatedAt: *obj.LastModified,
				Size:      *obj.Size,
			})
		}
		return !lastPage
	})

	if err != nil {
		s.log.Error("Error listing files in S3",
			zap.String("bucket", bucket),
			zap.String("prefix", prefix),
			zap.Error(err))
		return nil, fmt.Errorf("error listing files: %v", err)
	}

	s.log.Info("Files listed successfully",
		zap.String("bucket", bucket),
		zap.String("prefix", prefix),
		zap.Int("file_count", len(files)))

	return &ListResponse{
		Files: files,
	}, nil
}