package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOConfig holds MinIO connection configuration
type MinIOConfig struct {
	Endpoint        string        `yaml:"endpoint" json:"endpoint"`
	AccessKey       string        `yaml:"access_key" json:"access_key"`
	SecretKey       string        `yaml:"secret_key" json:"secret_key"`
	UseSSL          bool          `yaml:"use_ssl" json:"use_ssl"`
	Region          string        `yaml:"region" json:"region"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
	RequestTimeout  time.Duration `yaml:"request_timeout" json:"request_timeout"`
	VerificationBucket string     `yaml:"verification_bucket" json:"verification_bucket"`
	ModelsBucket    string        `yaml:"models_bucket" json:"models_bucket"`
	LogsBucket      string        `yaml:"logs_bucket" json:"logs_bucket"`
}

// DefaultMinIOConfig returns default MinIO configuration
func DefaultMinIOConfig() *MinIOConfig {
	return &MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKey:       "minioadmin",
		SecretKey:       "minioadmin123",
		UseSSL:          false,
		Region:          "us-east-1",
		ConnectTimeout:  30 * time.Second,
		RequestTimeout:  60 * time.Second,
		VerificationBucket: "llmsverifier-results",
		ModelsBucket:    "llmsverifier-models",
		LogsBucket:      "llmsverifier-logs",
	}
}

// Validate validates the MinIO configuration
func (c *MinIOConfig) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("access_key is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}
	if c.ConnectTimeout <= 0 {
		return fmt.Errorf("connect_timeout must be positive")
	}
	return nil
}

// MinIOClient wraps the MinIO client for LLMsVerifier storage
type MinIOClient struct {
	config      *MinIOConfig
	client      *minio.Client
	mu          sync.RWMutex
	connected   bool
}

// NewMinIOClient creates a new MinIO client
func NewMinIOClient(config *MinIOConfig) (*MinIOClient, error) {
	if config == nil {
		config = DefaultMinIOConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &MinIOClient{
		config:    config,
		connected: false,
	}, nil
}

// Connect establishes connection to MinIO
func (c *MinIOClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	client, err := minio.New(c.config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(c.config.AccessKey, c.config.SecretKey, ""),
		Secure: c.config.UseSSL,
		Region: c.config.Region,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO client: %w", err)
	}

	c.client = client

	// Verify connection
	_, err = client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to MinIO: %w", err)
	}

	// Ensure buckets exist
	if err := c.ensureBuckets(ctx); err != nil {
		return fmt.Errorf("failed to ensure buckets: %w", err)
	}

	c.connected = true
	return nil
}

func (c *MinIOClient) ensureBuckets(ctx context.Context) error {
	buckets := []string{
		c.config.VerificationBucket,
		c.config.ModelsBucket,
		c.config.LogsBucket,
	}

	for _, bucket := range buckets {
		exists, err := c.client.BucketExists(ctx, bucket)
		if err != nil {
			return fmt.Errorf("failed to check bucket %s: %w", bucket, err)
		}
		if !exists {
			if err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{
				Region: c.config.Region,
			}); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}
	}
	return nil
}

// Close closes the client connection
func (c *MinIOClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	c.client = nil
	return nil
}

// IsConnected returns whether the client is connected
func (c *MinIOClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// HealthCheck performs a health check on MinIO
func (c *MinIOClient) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.client == nil {
		return fmt.Errorf("not connected to MinIO")
	}

	_, err := c.client.ListBuckets(ctx)
	return err
}

// VerificationResult represents a verification result to store
type VerificationResult struct {
	ID           string                 `json:"id"`
	ProviderID   string                 `json:"provider_id"`
	ModelID      string                 `json:"model_id"`
	Score        float64                `json:"score"`
	Passed       bool                   `json:"passed"`
	Details      map[string]interface{} `json:"details"`
	Timestamp    time.Time              `json:"timestamp"`
	Duration     time.Duration          `json:"duration"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// StoreVerificationResult stores a verification result
func (c *MinIOClient) StoreVerificationResult(ctx context.Context, result *VerificationResult) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.client == nil {
		return fmt.Errorf("not connected to MinIO")
	}

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	objectName := fmt.Sprintf("%s/%s/%s/%s.json",
		result.ProviderID,
		result.ModelID,
		result.Timestamp.Format("2006/01/02"),
		result.ID,
	)

	reader := strings.NewReader(string(data))
	_, err = c.client.PutObject(ctx, c.config.VerificationBucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	return nil
}

// GetVerificationResult retrieves a verification result
func (c *MinIOClient) GetVerificationResult(ctx context.Context, providerID, modelID, resultID string) (*VerificationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("not connected to MinIO")
	}

	// Search for the result (simplified - in production would use index)
	prefix := fmt.Sprintf("%s/%s/", providerID, modelID)
	objects := c.client.ListObjects(ctx, c.config.VerificationBucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objects {
		if obj.Err != nil {
			continue
		}
		if strings.HasSuffix(obj.Key, resultID+".json") {
			object, err := c.client.GetObject(ctx, c.config.VerificationBucket, obj.Key, minio.GetObjectOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get object: %w", err)
			}
			defer object.Close()

			data, err := io.ReadAll(object)
			if err != nil {
				return nil, fmt.Errorf("failed to read object: %w", err)
			}

			var result VerificationResult
			if err := json.Unmarshal(data, &result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal result: %w", err)
			}

			return &result, nil
		}
	}

	return nil, fmt.Errorf("result not found")
}

// ListVerificationResults lists verification results for a provider/model
func (c *MinIOClient) ListVerificationResults(ctx context.Context, providerID, modelID string, since time.Time) ([]*VerificationResult, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("not connected to MinIO")
	}

	prefix := fmt.Sprintf("%s/%s/", providerID, modelID)
	objects := c.client.ListObjects(ctx, c.config.VerificationBucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var results []*VerificationResult
	for obj := range objects {
		if obj.Err != nil {
			continue
		}
		if !strings.HasSuffix(obj.Key, ".json") {
			continue
		}
		if obj.LastModified.Before(since) {
			continue
		}

		object, err := c.client.GetObject(ctx, c.config.VerificationBucket, obj.Key, minio.GetObjectOptions{})
		if err != nil {
			continue
		}

		data, err := io.ReadAll(object)
		object.Close()
		if err != nil {
			continue
		}

		var result VerificationResult
		if err := json.Unmarshal(data, &result); err != nil {
			continue
		}

		results = append(results, &result)
	}

	return results, nil
}

// StoreModelArtifact stores a model artifact
func (c *MinIOClient) StoreModelArtifact(ctx context.Context, providerID, modelID, artifactName string, reader io.Reader, size int64) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.client == nil {
		return fmt.Errorf("not connected to MinIO")
	}

	objectName := fmt.Sprintf("%s/%s/%s", providerID, modelID, artifactName)
	_, err := c.client.PutObject(ctx, c.config.ModelsBucket, objectName, reader, size, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to store artifact: %w", err)
	}

	return nil
}

// GetModelArtifact retrieves a model artifact
func (c *MinIOClient) GetModelArtifact(ctx context.Context, providerID, modelID, artifactName string) (io.ReadCloser, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.client == nil {
		return nil, fmt.Errorf("not connected to MinIO")
	}

	objectName := fmt.Sprintf("%s/%s/%s", providerID, modelID, artifactName)
	object, err := c.client.GetObject(ctx, c.config.ModelsBucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	}

	return object, nil
}

// LogEntry represents a log entry to store
type LogEntry struct {
	ID        string                 `json:"id"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Component string                 `json:"component"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

// StoreLog stores a log entry
func (c *MinIOClient) StoreLog(ctx context.Context, entry *LogEntry) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.client == nil {
		return fmt.Errorf("not connected to MinIO")
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	objectName := fmt.Sprintf("%s/%s/%s.json",
		entry.Timestamp.Format("2006/01/02/15"),
		entry.Component,
		entry.ID,
	)

	reader := strings.NewReader(string(data))
	_, err = c.client.PutObject(ctx, c.config.LogsBucket, objectName, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/json",
	})
	if err != nil {
		return fmt.Errorf("failed to store log: %w", err)
	}

	return nil
}
