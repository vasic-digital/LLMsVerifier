package lakehouse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultIcebergConfig(t *testing.T) {
	config := DefaultIcebergConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "http://localhost:8181", config.CatalogURI)
	assert.Equal(t, "s3://llmsverifier-iceberg/warehouse", config.Warehouse)
	assert.Equal(t, "http://localhost:9000", config.S3Endpoint)
	assert.Equal(t, "us-east-1", config.S3Region)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, "llmsverifier", config.Namespace)
}

func TestIcebergConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*IcebergConfig)
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			modify:      func(c *IcebergConfig) {},
			expectError: false,
		},
		{
			name: "empty catalog URI",
			modify: func(c *IcebergConfig) {
				c.CatalogURI = ""
			},
			expectError: true,
			errorMsg:    "catalog_uri is required",
		},
		{
			name: "empty warehouse",
			modify: func(c *IcebergConfig) {
				c.Warehouse = ""
			},
			expectError: true,
			errorMsg:    "warehouse is required",
		},
		{
			name: "invalid timeout",
			modify: func(c *IcebergConfig) {
				c.Timeout = 0
			},
			expectError: true,
			errorMsg:    "timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultIcebergConfig()
			tt.modify(config)

			err := config.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewIcebergClient(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		client, err := NewIcebergClient(nil)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.False(t, client.IsConnected())
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &IcebergConfig{
			CatalogURI:  "http://iceberg.example.com:8181",
			Warehouse:   "s3://custom-warehouse",
			Timeout:     60 * time.Second,
		}
		client, err := NewIcebergClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("with invalid config", func(t *testing.T) {
		config := &IcebergConfig{
			CatalogURI: "",
		}
		client, err := NewIcebergClient(config)
		require.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestVerificationResultsSchema(t *testing.T) {
	schema := VerificationResultsSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, 0, schema.SchemaID)
	assert.Len(t, schema.Fields, 9)

	// Verify field names
	fieldNames := make([]string, len(schema.Fields))
	for i, f := range schema.Fields {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "id")
	assert.Contains(t, fieldNames, "provider_id")
	assert.Contains(t, fieldNames, "model_id")
	assert.Contains(t, fieldNames, "score")
	assert.Contains(t, fieldNames, "passed")
	assert.Contains(t, fieldNames, "duration_ms")
	assert.Contains(t, fieldNames, "error_message")
	assert.Contains(t, fieldNames, "timestamp")
	assert.Contains(t, fieldNames, "details_json")
}

func TestModelMetricsSchema(t *testing.T) {
	schema := ModelMetricsSchema()

	assert.NotNil(t, schema)
	assert.Equal(t, 0, schema.SchemaID)
	assert.Len(t, schema.Fields, 7)

	// Verify field names
	fieldNames := make([]string, len(schema.Fields))
	for i, f := range schema.Fields {
		fieldNames[i] = f.Name
	}

	assert.Contains(t, fieldNames, "id")
	assert.Contains(t, fieldNames, "provider_id")
	assert.Contains(t, fieldNames, "model_id")
	assert.Contains(t, fieldNames, "metric_name")
	assert.Contains(t, fieldNames, "metric_value")
	assert.Contains(t, fieldNames, "timestamp")
	assert.Contains(t, fieldNames, "tags_json")
}

func TestTableSchema(t *testing.T) {
	schema := &TableSchema{
		SchemaID: 1,
		Fields: []TableField{
			{ID: 1, Name: "id", Type: "string", Required: true, Doc: "Primary key"},
			{ID: 2, Name: "value", Type: "double", Required: false},
		},
	}

	assert.Equal(t, 1, schema.SchemaID)
	assert.Len(t, schema.Fields, 2)
	assert.Equal(t, "id", schema.Fields[0].Name)
	assert.Equal(t, "string", schema.Fields[0].Type)
	assert.True(t, schema.Fields[0].Required)
	assert.Equal(t, "Primary key", schema.Fields[0].Doc)
}

func TestTableInfo(t *testing.T) {
	schema := VerificationResultsSchema()
	info := &TableInfo{
		Namespace:  "llmsverifier",
		Name:       "verification_results",
		Schema:     schema,
		Location:   "s3://warehouse/llmsverifier/verification_results",
		Properties: map[string]string{
			"write.format.default": "parquet",
		},
	}

	assert.Equal(t, "llmsverifier", info.Namespace)
	assert.Equal(t, "verification_results", info.Name)
	assert.NotNil(t, info.Schema)
	assert.Equal(t, "parquet", info.Properties["write.format.default"])
}

func TestSnapshotInfo(t *testing.T) {
	snapshot := &SnapshotInfo{
		SnapshotID:   12345678901234,
		TimestampMS:  1704067200000,
		ManifestList: "s3://warehouse/metadata/snap-12345.avro",
		Summary: map[string]string{
			"added-records": "1000",
			"added-files":   "5",
		},
	}

	assert.Equal(t, int64(12345678901234), snapshot.SnapshotID)
	assert.Equal(t, int64(1704067200000), snapshot.TimestampMS)
	assert.Equal(t, "1000", snapshot.Summary["added-records"])
}
