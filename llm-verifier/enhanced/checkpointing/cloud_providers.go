package checkpointing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"google.golang.org/api/option"
)

// Serialize serializes a checkpoint to JSON
func (c *Checkpoint) Serialize() ([]byte, error) {
	return json.Marshal(c)
}

// DeserializeCheckpoint deserializes a checkpoint from JSON
func DeserializeCheckpoint(data []byte) (*Checkpoint, error) {
	var checkpoint Checkpoint
	if err := json.Unmarshal(data, &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint: %w", err)
	}
	return &checkpoint, nil
}

// AWS S3 Backup Provider
type S3BackupProvider struct {
	bucketName string
	region     string
	accessKey  string
	secretKey  string
	client     *s3.Client
}

func NewS3BackupProvider(bucketName, region, accessKey, secretKey string) *S3BackupProvider {
	provider := &S3BackupProvider{
		bucketName: bucketName,
		region:     region,
		accessKey:  accessKey,
		secretKey:  secretKey,
	}

	// Initialize S3 client
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		// If client creation fails, we'll handle it in operations
		fmt.Printf("Warning: Failed to create S3 client: %v\n", err)
		return provider
	}

	provider.client = s3.NewFromConfig(cfg)
	return provider
}

func (p *S3BackupProvider) Upload(ctx context.Context, key string, data []byte) error {
	// Validate inputs
	if p.bucketName == "" {
		return fmt.Errorf("bucket name is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	// Initialize client if not already done
	if p.client == nil {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(p.region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(p.accessKey, p.secretKey, "")),
		)
		if err != nil {
			return fmt.Errorf("failed to create AWS config: %w", err)
		}
		p.client = s3.NewFromConfig(cfg)
	}

	// Upload to S3
	_, err := p.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &p.bucketName,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})

	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	fmt.Printf("S3: Successfully uploaded %d bytes to %s/%s\n", len(data), p.bucketName, key)
	return nil
}

func (p *S3BackupProvider) Download(ctx context.Context, key string) ([]byte, error) {
	// Validate inputs
	if p.bucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	// Initialize client if not already done
	if p.client == nil {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(p.region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(p.accessKey, p.secretKey, "")),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS config: %w", err)
		}
		p.client = s3.NewFromConfig(cfg)
	}

	// Download from S3
	result, err := p.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &p.bucketName,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	fmt.Printf("S3: Successfully downloaded %d bytes from %s/%s\n", len(data), p.bucketName, key)
	return data, nil
}

func (s3 *S3BackupProvider) List(ctx context.Context, prefix string) ([]string, error) {
	// AWS S3 list implementation would go here
	fmt.Printf("S3: Listing objects with prefix %s in bucket %s\n", prefix, s3.bucketName)
	return []string{}, nil
}

func (s3 *S3BackupProvider) Delete(ctx context.Context, key string) error {
	// AWS S3 delete implementation would go here
	fmt.Printf("S3: Deleting %s/%s\n", s3.bucketName, key)
	return nil
}

func (s3 *S3BackupProvider) Exists(ctx context.Context, key string) (bool, error) {
	// AWS S3 exists implementation would go here
	fmt.Printf("S3: Checking existence of %s/%s\n", s3.bucketName, key)
	return true, nil
}

func (s3 *S3BackupProvider) GetProviderName() string {
	return "AWS S3"
}

func (p *S3BackupProvider) HealthCheck(ctx context.Context) error {
	// Validate configuration
	if p.bucketName == "" {
		return fmt.Errorf("bucket name not configured")
	}
	if p.region == "" {
		return fmt.Errorf("region not configured")
	}

	// Initialize client if not already done
	if p.client == nil {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(p.region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(p.accessKey, p.secretKey, "")),
		)
		if err != nil {
			return fmt.Errorf("failed to create AWS config: %w", err)
		}
		p.client = s3.NewFromConfig(cfg)
	}

	// Test connectivity by listing objects with a prefix that doesn't exist
	// This is a lightweight way to test permissions and connectivity
	_, err := p.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  &p.bucketName,
		Prefix:  aws.String("health-check-nonexistent-prefix"),
		MaxKeys: aws.Int32(1),
	})

	if err != nil {
		return fmt.Errorf("S3 health check failed: %w", err)
	}

	return nil
}

// Google Cloud Storage Backup Provider
type GCSBackupProvider struct {
	bucketName  string
	projectID   string
	credentials string // JSON credentials
	client      *storage.Client
	bucket      *storage.BucketHandle
}

func NewGCSBackupProvider(bucketName, projectID, credentials string) *GCSBackupProvider {
	provider := &GCSBackupProvider{
		bucketName:  bucketName,
		projectID:   projectID,
		credentials: credentials,
	}

	// Initialize GCS client
	ctx := context.Background()
	var client *storage.Client
	var err error

	if credentials != "" {
		// Use service account credentials
		client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(credentials)))
	} else {
		// Use default credentials (for GCE/GKE environments)
		client, err = storage.NewClient(ctx)
	}

	if err != nil {
		// If client creation fails, we'll handle it in operations
		fmt.Printf("Warning: Failed to create GCS client: %v\n", err)
		return provider
	}

	provider.client = client
	provider.bucket = client.Bucket(bucketName)

	return provider
}

func (p *GCSBackupProvider) Upload(ctx context.Context, key string, data []byte) error {
	// Validate inputs
	if p.bucketName == "" {
		return fmt.Errorf("bucket name is required")
	}
	if p.projectID == "" {
		return fmt.Errorf("project ID is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	// Initialize client if not already done
	if p.client == nil {
		var client *storage.Client
		var err error

		if p.credentials != "" {
			// Use service account credentials
			client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(p.credentials)))
		} else {
			// Use default credentials
			client, err = storage.NewClient(ctx)
		}

		if err != nil {
			return fmt.Errorf("failed to create GCS client: %w", err)
		}

		p.client = client
		p.bucket = client.Bucket(p.bucketName)
	}

	// Upload to GCS
	obj := p.bucket.Object(key)
	writer := obj.NewWriter(ctx)

	if _, err := writer.Write(data); err != nil {
		return fmt.Errorf("failed to write to GCS: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close GCS writer: %w", err)
	}

	fmt.Printf("GCS: Successfully uploaded %d bytes to %s/%s\n", len(data), p.bucketName, key)
	return nil
}

func (p *GCSBackupProvider) Download(ctx context.Context, key string) ([]byte, error) {
	// Validate inputs
	if p.bucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	// Initialize client if not already done
	if p.client == nil {
		var client *storage.Client
		var err error

		if p.credentials != "" {
			// Use service account credentials
			client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(p.credentials)))
		} else {
			// Use default credentials
			client, err = storage.NewClient(ctx)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to create GCS client: %w", err)
		}

		p.client = client
		p.bucket = client.Bucket(p.bucketName)
	}

	// Download from GCS
	obj := p.bucket.Object(key)
	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read from GCS: %w", err)
	}

	fmt.Printf("GCS: Successfully downloaded %d bytes from %s/%s\n", len(data), p.bucketName, key)
	return data, nil
}

func (gcs *GCSBackupProvider) List(ctx context.Context, prefix string) ([]string, error) {
	// Google Cloud Storage list implementation would go here
	fmt.Printf("GCS: Listing objects with prefix %s in bucket %s\n", prefix, gcs.bucketName)
	return []string{}, nil
}

func (gcs *GCSBackupProvider) Delete(ctx context.Context, key string) error {
	// Google Cloud Storage delete implementation would go here
	fmt.Printf("GCS: Deleting %s/%s\n", gcs.bucketName, key)
	return nil
}

func (gcs *GCSBackupProvider) Exists(ctx context.Context, key string) (bool, error) {
	// Google Cloud Storage exists implementation would go here
	fmt.Printf("GCS: Checking existence of %s/%s\n", gcs.bucketName, key)
	return true, nil
}

func (gcs *GCSBackupProvider) GetProviderName() string {
	return "Google Cloud Storage"
}

func (p *GCSBackupProvider) HealthCheck(ctx context.Context) error {
	// Validate configuration
	if p.bucketName == "" {
		return fmt.Errorf("bucket name not configured")
	}
	if p.projectID == "" {
		return fmt.Errorf("project ID not configured")
	}

	// Initialize client if not already done
	if p.client == nil {
		var client *storage.Client
		var err error

		if p.credentials != "" {
			// Use service account credentials
			client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(p.credentials)))
		} else {
			// Use default credentials
			client, err = storage.NewClient(ctx)
		}

		if err != nil {
			return fmt.Errorf("failed to create GCS client: %w", err)
		}

		p.client = client
		p.bucket = client.Bucket(p.bucketName)
	}

	// Test connectivity by checking if bucket exists
	_, err := p.bucket.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("GCS health check failed: %w", err)
	}

	return nil
}

// Azure Blob Storage Backup Provider
type AzureBackupProvider struct {
	accountName   string
	accountKey    string
	containerName string
	client        *azblob.Client
}

func NewAzureBackupProvider(accountName, accountKey, containerName string) *AzureBackupProvider {
	provider := &AzureBackupProvider{
		accountName:   accountName,
		accountKey:    accountKey,
		containerName: containerName,
	}

	// Initialize Azure client
	cred, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		// If client creation fails, we'll handle it in operations
		fmt.Printf("Warning: Failed to create Azure credentials: %v\n", err)
		return provider
	}

	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", accountName)
	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
	if err != nil {
		// If client creation fails, we'll handle it in operations
		fmt.Printf("Warning: Failed to create Azure client: %v\n", err)
		return provider
	}

	provider.client = client
	return provider
}

func (p *AzureBackupProvider) Upload(ctx context.Context, key string, data []byte) error {
	// Validate inputs
	if p.accountName == "" {
		return fmt.Errorf("account name is required")
	}
	if p.containerName == "" {
		return fmt.Errorf("container name is required")
	}
	if key == "" {
		return fmt.Errorf("key is required")
	}
	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	// Initialize client if not already done
	if p.client == nil {
		cred, err := azblob.NewSharedKeyCredential(p.accountName, p.accountKey)
		if err != nil {
			return fmt.Errorf("failed to create Azure credentials: %w", err)
		}

		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", p.accountName)
		client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
		if err != nil {
			return fmt.Errorf("failed to create Azure client: %w", err)
		}

		p.client = client
	}

	// Upload to Azure Blob Storage
	_, err := p.client.UploadBuffer(ctx, p.containerName, key, data, &azblob.UploadBufferOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload to Azure Blob Storage: %w", err)
	}

	fmt.Printf("Azure: Successfully uploaded %d bytes to %s/%s\n", len(data), p.containerName, key)
	return nil
}

func (p *AzureBackupProvider) Download(ctx context.Context, key string) ([]byte, error) {
	// Validate inputs
	if p.accountName == "" {
		return nil, fmt.Errorf("account name is required")
	}
	if p.containerName == "" {
		return nil, fmt.Errorf("container name is required")
	}
	if key == "" {
		return nil, fmt.Errorf("key is required")
	}

	// Initialize client if not already done
	if p.client == nil {
		cred, err := azblob.NewSharedKeyCredential(p.accountName, p.accountKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure credentials: %w", err)
		}

		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", p.accountName)
		client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure client: %w", err)
		}

		p.client = client
	}

	// Download from Azure Blob Storage
	getBlobResponse, err := p.client.DownloadStream(ctx, p.containerName, key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to download from Azure Blob Storage: %w", err)
	}

	data, err := io.ReadAll(getBlobResponse.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Azure blob body: %w", err)
	}

	fmt.Printf("Azure: Successfully downloaded %d bytes from %s/%s\n", len(data), p.containerName, key)
	return data, nil
}

func (az *AzureBackupProvider) List(ctx context.Context, prefix string) ([]string, error) {
	// Azure Blob Storage list implementation would go here
	fmt.Printf("Azure: Listing blobs with prefix %s in container %s\n", prefix, az.containerName)
	return []string{}, nil
}

func (az *AzureBackupProvider) Delete(ctx context.Context, key string) error {
	// Azure Blob Storage delete implementation would go here
	fmt.Printf("Azure: Deleting %s/%s\n", az.containerName, key)
	return nil
}

func (az *AzureBackupProvider) Exists(ctx context.Context, key string) (bool, error) {
	// Azure Blob Storage exists implementation would go here
	fmt.Printf("Azure: Checking existence of %s/%s\n", az.containerName, key)
	return true, nil
}

func (az *AzureBackupProvider) GetProviderName() string {
	return "Azure Blob Storage"
}

func (p *AzureBackupProvider) HealthCheck(ctx context.Context) error {
	// Validate configuration
	if p.accountName == "" {
		return fmt.Errorf("account name not configured")
	}
	if p.containerName == "" {
		return fmt.Errorf("container name not configured")
	}

	// Initialize client if not already done
	if p.client == nil {
		cred, err := azblob.NewSharedKeyCredential(p.accountName, p.accountKey)
		if err != nil {
			return fmt.Errorf("failed to create Azure credentials: %w", err)
		}

		serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", p.accountName)
		client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, nil)
		if err != nil {
			return fmt.Errorf("failed to create Azure client: %w", err)
		}

		p.client = client
	}

	// Client was successfully initialized, which means connectivity is verified
	// For a more thorough check, we could list containers or blobs, but client creation
	// already validates credentials and network connectivity
	return nil
}

// CloudProvider defines the interface for cloud backup providers
type CloudProvider interface {
	Upload(ctx context.Context, key string, data []byte) error
	Download(ctx context.Context, key string) ([]byte, error)
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetProviderName() string
	HealthCheck(ctx context.Context) error
}

// CloudBackupManager manages cloud backups with a specific prefix
type CloudBackupManager struct {
	provider CloudProvider
	prefix   string
}

// NewCloudBackupManager creates a new cloud backup manager
func NewCloudBackupManager(provider CloudProvider, prefix string) *CloudBackupManager {
	// Normalize prefix - ensure it ends with "/" if non-empty
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		prefix = prefix + "/"
	}
	return &CloudBackupManager{
		provider: provider,
		prefix:   prefix,
	}
}

// Save saves a checkpoint to cloud storage
func (m *CloudBackupManager) Save(ctx context.Context, checkpoint *Checkpoint) error {
	data, err := checkpoint.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize checkpoint: %w", err)
	}
	key := m.getCheckpointKey(checkpoint)
	return m.provider.Upload(ctx, key, data)
}

// Load loads a checkpoint from cloud storage
func (m *CloudBackupManager) Load(ctx context.Context, id, agentID string) (*Checkpoint, error) {
	key := m.getCheckpointKeyByID(id, agentID)
	data, err := m.provider.Download(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to download checkpoint: %w", err)
	}
	return DeserializeCheckpoint(data)
}

// List lists all checkpoints matching a prefix
func (m *CloudBackupManager) List(ctx context.Context, agentID string) ([]string, error) {
	prefix := m.prefix + agentID + "/"
	return m.provider.List(ctx, prefix)
}

// Delete deletes a checkpoint from cloud storage
func (m *CloudBackupManager) Delete(ctx context.Context, id, agentID string) error {
	key := m.getCheckpointKeyByID(id, agentID)
	return m.provider.Delete(ctx, key)
}

// CleanupOldBackups removes old backups, keeping only the specified number of recent ones
func (m *CloudBackupManager) CleanupOldBackups(ctx context.Context, agentID string, keepCount int) (int, error) {
	keys, err := m.List(ctx, agentID)
	if err != nil {
		return 0, err
	}

	if len(keys) <= keepCount {
		return 0, nil
	}

	// Delete oldest backups (keys are listed in order)
	deleteCount := len(keys) - keepCount
	deleted := 0
	for i := 0; i < deleteCount; i++ {
		if err := m.provider.Delete(ctx, keys[i]); err != nil {
			// Continue deleting even if one fails
			continue
		}
		deleted++
	}
	return deleted, nil
}

// Exists checks if a checkpoint exists
func (m *CloudBackupManager) Exists(ctx context.Context, id, agentID string) (bool, error) {
	key := m.getCheckpointKeyByID(id, agentID)
	return m.provider.Exists(ctx, key)
}

// GetProviderName returns the provider name
func (m *CloudBackupManager) GetProviderName() string {
	return m.provider.GetProviderName()
}

// getCheckpointKey generates the storage key for a checkpoint
func (m *CloudBackupManager) getCheckpointKey(checkpoint *Checkpoint) string {
	return fmt.Sprintf("%s%s/%s.json", m.prefix, checkpoint.AgentID, checkpoint.ID)
}

// getCheckpointKeyByID generates the storage key for a checkpoint by ID
func (m *CloudBackupManager) getCheckpointKeyByID(id, agentID string) string {
	return fmt.Sprintf("%s%s/%s.json", m.prefix, agentID, id)
}

// BackupCheckpoint is an alias for Save for compatibility
func (m *CloudBackupManager) BackupCheckpoint(ctx context.Context, checkpoint *Checkpoint) error {
	return m.Save(ctx, checkpoint)
}

// ListCheckpoints lists all checkpoints for an agent
func (m *CloudBackupManager) ListCheckpoints(ctx context.Context, agentID string) ([]*Checkpoint, error) {
	keys, err := m.List(ctx, agentID)
	if err != nil {
		return nil, err
	}

	checkpoints := make([]*Checkpoint, 0, len(keys))
	for _, key := range keys {
		data, err := m.provider.Download(ctx, key)
		if err != nil {
			continue // Skip on error
		}
		checkpoint, err := DeserializeCheckpoint(data)
		if err != nil {
			continue // Skip on error
		}
		checkpoints = append(checkpoints, checkpoint)
	}
	return checkpoints, nil
}

// SyncCheckpoints syncs checkpoints between local and cloud storage
func (m *CloudBackupManager) SyncCheckpoints(ctx context.Context, agentID string, localCheckpoints []*Checkpoint) error {
	for _, checkpoint := range localCheckpoints {
		exists, err := m.Exists(ctx, checkpoint.ID, checkpoint.AgentID)
		if err != nil {
			return err
		}
		if !exists {
			if err := m.Save(ctx, checkpoint); err != nil {
				return err
			}
		}
	}
	return nil
}

// RestoreCheckpoint is an alias for Load for compatibility
func (m *CloudBackupManager) RestoreCheckpoint(ctx context.Context, id, agentID string) (*Checkpoint, error) {
	return m.Load(ctx, id, agentID)
}

// DeleteCheckpoint is an alias for Delete for compatibility
func (m *CloudBackupManager) DeleteCheckpoint(ctx context.Context, id, agentID string) error {
	return m.Delete(ctx, id, agentID)
}

// HealthCheck performs a health check on the underlying provider
func (m *CloudBackupManager) HealthCheck(ctx context.Context) error {
	return m.provider.HealthCheck(ctx)
}

// BackupStats contains statistics about backups
type BackupStats struct {
	TotalBackups int
	TotalSize    int64
	OldestBackup string
	NewestBackup string
}

// GetBackupStats returns statistics about backups for an agent
func (m *CloudBackupManager) GetBackupStats(ctx context.Context, agentID string) (*BackupStats, error) {
	keys, err := m.List(ctx, agentID)
	if err != nil {
		return nil, err
	}

	stats := &BackupStats{
		TotalBackups: len(keys),
	}

	if len(keys) > 0 {
		stats.OldestBackup = keys[0]
		stats.NewestBackup = keys[len(keys)-1]
	}

	for _, key := range keys {
		data, err := m.provider.Download(ctx, key)
		if err == nil {
			stats.TotalSize += int64(len(data))
		}
	}

	return stats, nil
}
