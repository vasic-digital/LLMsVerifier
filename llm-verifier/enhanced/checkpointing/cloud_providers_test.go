package checkpointing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test S3BackupProvider

func TestNewS3BackupProvider(t *testing.T) {
	provider := NewS3BackupProvider("test-bucket", "us-east-1", "access-key", "secret-key")
	require.NotNil(t, provider)
	assert.Equal(t, "test-bucket", provider.bucketName)
	assert.Equal(t, "us-east-1", provider.region)
	assert.Equal(t, "access-key", provider.accessKey)
	assert.Equal(t, "secret-key", provider.secretKey)
}

func TestS3BackupProvider_GetProviderName(t *testing.T) {
	provider := NewS3BackupProvider("test-bucket", "us-east-1", "access-key", "secret-key")
	assert.Equal(t, "AWS S3", provider.GetProviderName())
}

func TestS3BackupProvider_Upload_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty bucket name", func(t *testing.T) {
		provider := &S3BackupProvider{
			bucketName: "",
			region:     "us-east-1",
		}
		err := provider.Upload(ctx, "key", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name is required")
	})

	t.Run("empty key", func(t *testing.T) {
		provider := &S3BackupProvider{
			bucketName: "bucket",
			region:     "us-east-1",
		}
		err := provider.Upload(ctx, "", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key is required")
	})

	t.Run("empty data", func(t *testing.T) {
		provider := &S3BackupProvider{
			bucketName: "bucket",
			region:     "us-east-1",
		}
		err := provider.Upload(ctx, "key", []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data cannot be empty")
	})
}

func TestS3BackupProvider_Download_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty bucket name", func(t *testing.T) {
		provider := &S3BackupProvider{
			bucketName: "",
		}
		_, err := provider.Download(ctx, "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name is required")
	})

	t.Run("empty key", func(t *testing.T) {
		provider := &S3BackupProvider{
			bucketName: "bucket",
		}
		_, err := provider.Download(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key is required")
	})
}

func TestS3BackupProvider_HealthCheck_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty bucket name", func(t *testing.T) {
		provider := &S3BackupProvider{
			bucketName: "",
		}
		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name not configured")
	})

	t.Run("empty region", func(t *testing.T) {
		provider := &S3BackupProvider{
			bucketName: "bucket",
			region:     "",
		}
		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region not configured")
	})
}

func TestS3BackupProvider_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping S3 integration test in short mode")  // SKIP-OK: #short-mode
	}
	// These tests require actual AWS credentials
	t.Skip("Skipping S3 integration test - requires AWS credentials")  // SKIP-OK: #integration-mode-only
}

func TestS3BackupProvider_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping S3 integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping S3 integration test - requires AWS credentials")  // SKIP-OK: #integration-mode-only
}

func TestS3BackupProvider_Exists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping S3 integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping S3 integration test - requires AWS credentials")  // SKIP-OK: #integration-mode-only
}

// Test GCSBackupProvider

func TestNewGCSBackupProvider(t *testing.T) {
	provider := NewGCSBackupProvider("test-bucket", "test-project", "")
	require.NotNil(t, provider)
	assert.Equal(t, "test-bucket", provider.bucketName)
	assert.Equal(t, "test-project", provider.projectID)
}

func TestGCSBackupProvider_GetProviderName(t *testing.T) {
	provider := NewGCSBackupProvider("test-bucket", "test-project", "")
	assert.Equal(t, "Google Cloud Storage", provider.GetProviderName())
}

func TestGCSBackupProvider_Upload_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty bucket name", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "",
			projectID:  "project",
		}
		err := provider.Upload(ctx, "key", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name is required")
	})

	t.Run("empty project ID", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "bucket",
			projectID:  "",
		}
		err := provider.Upload(ctx, "key", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project ID is required")
	})

	t.Run("empty key", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "bucket",
			projectID:  "project",
		}
		err := provider.Upload(ctx, "", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key is required")
	})

	t.Run("empty data", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "bucket",
			projectID:  "project",
		}
		err := provider.Upload(ctx, "key", []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data cannot be empty")
	})
}

func TestGCSBackupProvider_Download_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty bucket name", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "",
		}
		_, err := provider.Download(ctx, "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name is required")
	})

	t.Run("empty key", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "bucket",
		}
		_, err := provider.Download(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key is required")
	})
}

func TestGCSBackupProvider_HealthCheck_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty bucket name", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "",
		}
		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bucket name not configured")
	})

	t.Run("empty project ID", func(t *testing.T) {
		provider := &GCSBackupProvider{
			bucketName: "bucket",
			projectID:  "",
		}
		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "project ID not configured")
	})
}

func TestGCSBackupProvider_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GCS integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping GCS integration test - requires GCP credentials")  // SKIP-OK: #integration-mode-only
}

func TestGCSBackupProvider_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GCS integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping GCS integration test - requires GCP credentials")  // SKIP-OK: #integration-mode-only
}

func TestGCSBackupProvider_Exists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GCS integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping GCS integration test - requires GCP credentials")  // SKIP-OK: #integration-mode-only
}

// Test AzureBackupProvider

func TestNewAzureBackupProvider(t *testing.T) {
	provider := NewAzureBackupProvider("testaccount", "testkey", "testcontainer")
	require.NotNil(t, provider)
	assert.Equal(t, "testaccount", provider.accountName)
	assert.Equal(t, "testkey", provider.accountKey)
	assert.Equal(t, "testcontainer", provider.containerName)
}

func TestAzureBackupProvider_GetProviderName(t *testing.T) {
	provider := NewAzureBackupProvider("testaccount", "testkey", "testcontainer")
	assert.Equal(t, "Azure Blob Storage", provider.GetProviderName())
}

func TestAzureBackupProvider_Upload_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty account name", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "",
			containerName: "container",
		}
		err := provider.Upload(ctx, "key", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account name is required")
	})

	t.Run("empty container name", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "account",
			containerName: "",
		}
		err := provider.Upload(ctx, "key", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "container name is required")
	})

	t.Run("empty key", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "account",
			containerName: "container",
		}
		err := provider.Upload(ctx, "", []byte("data"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key is required")
	})

	t.Run("empty data", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "account",
			containerName: "container",
		}
		err := provider.Upload(ctx, "key", []byte{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data cannot be empty")
	})
}

func TestAzureBackupProvider_Download_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty account name", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "",
			containerName: "container",
		}
		_, err := provider.Download(ctx, "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account name is required")
	})

	t.Run("empty container name", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "account",
			containerName: "",
		}
		_, err := provider.Download(ctx, "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "container name is required")
	})

	t.Run("empty key", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "account",
			containerName: "container",
		}
		_, err := provider.Download(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "key is required")
	})
}

func TestAzureBackupProvider_HealthCheck_Validation(t *testing.T) {
	ctx := context.Background()

	t.Run("empty account name", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName: "",
		}
		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account name not configured")
	})

	t.Run("empty container name", func(t *testing.T) {
		provider := &AzureBackupProvider{
			accountName:   "account",
			containerName: "",
		}
		err := provider.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "container name not configured")
	})
}

func TestAzureBackupProvider_List(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Azure integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping Azure integration test - requires Azure credentials")  // SKIP-OK: #integration-mode-only
}

func TestAzureBackupProvider_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Azure integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping Azure integration test - requires Azure credentials")  // SKIP-OK: #integration-mode-only
}

func TestAzureBackupProvider_Exists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Azure integration test in short mode")  // SKIP-OK: #short-mode
	}
	t.Skip("Skipping Azure integration test - requires Azure credentials")  // SKIP-OK: #integration-mode-only
}
