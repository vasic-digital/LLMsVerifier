package checkpointing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Provider implements CloudBackupProvider for AWS S3
type S3Provider struct {
	client *s3.Client
	bucket string
	region string
}

// NewS3Provider creates a new S3 cloud backup provider
func NewS3Provider(bucket, region string) (*S3Provider, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &S3Provider{
		client: client,
		bucket: bucket,
		region: region,
	}, nil
}

// Upload uploads data to S3
func (s3p *S3Provider) Upload(ctx context.Context, key string, data []byte) error {
	_, err := s3p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}
	return nil
}

// Download downloads data from S3
func (s3p *S3Provider) Download(ctx context.Context, key string) ([]byte, error) {
	result, err := s3p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return data, nil
}

// List lists objects in S3 with the given prefix
func (s3p *S3Provider) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	paginator := s3.NewListObjectsV2Paginator(s3p.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3p.bucket),
		Prefix: aws.String(prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list S3 objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
	}

	return keys, nil
}

// Delete deletes an object from S3
func (s3p *S3Provider) Delete(ctx context.Context, key string) error {
	_, err := s3p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete from S3: %w", err)
	}
	return nil
}

// Exists checks if an object exists in S3
func (s3p *S3Provider) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s3p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		// Also check for NoSuchKey
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check S3 object existence: %w", err)
	}
	return true, nil
}

// GetProviderName returns the provider name
func (s3p *S3Provider) GetProviderName() string {
	return "AWS S3"
}

// HealthCheck performs a health check
func (s3p *S3Provider) HealthCheck(ctx context.Context) error {
	// Try to list objects (this will fail if bucket doesn't exist or permissions are wrong)
	_, err := s3p.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s3p.bucket),
		MaxKeys: aws.Int32(1),
	})
	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}
	return nil
}
