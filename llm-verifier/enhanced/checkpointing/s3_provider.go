package checkpointing

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

// S3Provider implements CloudBackupProvider for AWS S3
type S3Provider struct {
	session    *session.Session
	bucket     string
	region     string
	uploader   *s3manager.Uploader
	downloader *s3manager.Downloader
	s3Client   *s3.S3
}

// NewS3Provider creates a new S3 cloud backup provider
func NewS3Provider(bucket, region string) (*S3Provider, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	uploader := s3manager.NewUploader(sess)
	downloader := s3manager.NewDownloader(sess)
	s3Client := s3.New(sess)

	return &S3Provider{
		session:    sess,
		bucket:     bucket,
		region:     region,
		uploader:   uploader,
		downloader: downloader,
		s3Client:   s3Client,
	}, nil
}

// Upload uploads data to S3
func (s3p *S3Provider) Upload(ctx context.Context, key string, data []byte) error {
	_, err := s3p.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
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
	buf := &aws.WriteAtBuffer{}

	_, err := s3p.downloader.DownloadWithContext(ctx, buf, &s3.GetObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	return buf.Bytes(), nil
}

// List lists objects in S3 with the given prefix
func (s3p *S3Provider) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	err := s3p.s3Client.ListObjectsV2PagesWithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3p.bucket),
		Prefix: aws.String(prefix),
	}, func(page *s3.ListObjectsV2Output, lastPage bool) bool {
		for _, obj := range page.Contents {
			if obj.Key != nil {
				keys = append(keys, *obj.Key)
			}
		}
		return !lastPage
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	return keys, nil
}

// Delete deletes an object from S3
func (s3p *S3Provider) Delete(ctx context.Context, key string) error {
	_, err := s3p.s3Client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
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
	_, err := s3p.s3Client.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s3p.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
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
	_, err := s3p.s3Client.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(s3p.bucket),
		MaxKeys: aws.Int64(1),
	})
	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}
	return nil
}
