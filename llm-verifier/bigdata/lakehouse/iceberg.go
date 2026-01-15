package lakehouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// IcebergConfig holds Iceberg REST catalog configuration
type IcebergConfig struct {
	CatalogURI       string        `yaml:"catalog_uri" json:"catalog_uri"`
	Warehouse        string        `yaml:"warehouse" json:"warehouse"`
	S3Endpoint       string        `yaml:"s3_endpoint" json:"s3_endpoint"`
	S3AccessKey      string        `yaml:"s3_access_key" json:"s3_access_key"`
	S3SecretKey      string        `yaml:"s3_secret_key" json:"s3_secret_key"`
	S3Region         string        `yaml:"s3_region" json:"s3_region"`
	Timeout          time.Duration `yaml:"timeout" json:"timeout"`
	Namespace        string        `yaml:"namespace" json:"namespace"`
}

// DefaultIcebergConfig returns default Iceberg configuration
func DefaultIcebergConfig() *IcebergConfig {
	return &IcebergConfig{
		CatalogURI:  "http://localhost:8181",
		Warehouse:   "s3://llmsverifier-iceberg/warehouse",
		S3Endpoint:  "http://localhost:9000",
		S3Region:    "us-east-1",
		Timeout:     30 * time.Second,
		Namespace:   "llmsverifier",
	}
}

// Validate validates the Iceberg configuration
func (c *IcebergConfig) Validate() error {
	if c.CatalogURI == "" {
		return fmt.Errorf("catalog_uri is required")
	}
	if c.Warehouse == "" {
		return fmt.Errorf("warehouse is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// IcebergClient wraps the Iceberg REST catalog client for LLMsVerifier
type IcebergClient struct {
	config    *IcebergConfig
	client    *http.Client
	mu        sync.RWMutex
	connected bool
}

// NewIcebergClient creates a new Iceberg client
func NewIcebergClient(config *IcebergConfig) (*IcebergClient, error) {
	if config == nil {
		config = DefaultIcebergConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &IcebergClient{
		config: config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		connected: false,
	}, nil
}

// Connect establishes connection to Iceberg catalog
func (c *IcebergClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify connection by listing namespaces
	url := fmt.Sprintf("%s/v1/namespaces", c.config.CatalogURI)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Iceberg catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to connect to Iceberg catalog: status %d", resp.StatusCode)
	}

	// Ensure namespace exists
	if err := c.ensureNamespace(ctx); err != nil {
		return fmt.Errorf("failed to ensure namespace: %w", err)
	}

	c.connected = true
	return nil
}

func (c *IcebergClient) ensureNamespace(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1/namespaces/%s", c.config.CatalogURI, c.config.Namespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil // Namespace exists
	}

	// Create namespace
	url = fmt.Sprintf("%s/v1/namespaces", c.config.CatalogURI)
	payload := map[string]interface{}{
		"namespace": []string{c.config.Namespace},
		"properties": map[string]string{
			"owner": "llmsverifier",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err = http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create namespace: %s", string(body))
	}

	return nil
}

// Close closes the client connection
func (c *IcebergClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return nil
}

// IsConnected returns whether the client is connected
func (c *IcebergClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// HealthCheck performs a health check on Iceberg catalog
func (c *IcebergClient) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Iceberg catalog")
	}

	url := fmt.Sprintf("%s/v1/namespaces", c.config.CatalogURI)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// TableSchema represents an Iceberg table schema
type TableSchema struct {
	SchemaID int          `json:"schema-id"`
	Fields   []TableField `json:"fields"`
}

// TableField represents a field in a table schema
type TableField struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
	Doc      string `json:"doc,omitempty"`
}

// VerificationResultsSchema returns the schema for verification results table
func VerificationResultsSchema() *TableSchema {
	return &TableSchema{
		SchemaID: 0,
		Fields: []TableField{
			{ID: 1, Name: "id", Type: "string", Required: true, Doc: "Unique verification ID"},
			{ID: 2, Name: "provider_id", Type: "string", Required: true, Doc: "Provider identifier"},
			{ID: 3, Name: "model_id", Type: "string", Required: true, Doc: "Model identifier"},
			{ID: 4, Name: "score", Type: "double", Required: true, Doc: "Verification score"},
			{ID: 5, Name: "passed", Type: "boolean", Required: true, Doc: "Whether verification passed"},
			{ID: 6, Name: "duration_ms", Type: "long", Required: true, Doc: "Duration in milliseconds"},
			{ID: 7, Name: "error_message", Type: "string", Required: false, Doc: "Error message if failed"},
			{ID: 8, Name: "timestamp", Type: "timestamp", Required: true, Doc: "Verification timestamp"},
			{ID: 9, Name: "details_json", Type: "string", Required: false, Doc: "JSON details"},
		},
	}
}

// ModelMetricsSchema returns the schema for model metrics table
func ModelMetricsSchema() *TableSchema {
	return &TableSchema{
		SchemaID: 0,
		Fields: []TableField{
			{ID: 1, Name: "id", Type: "string", Required: true, Doc: "Unique metric ID"},
			{ID: 2, Name: "provider_id", Type: "string", Required: true, Doc: "Provider identifier"},
			{ID: 3, Name: "model_id", Type: "string", Required: true, Doc: "Model identifier"},
			{ID: 4, Name: "metric_name", Type: "string", Required: true, Doc: "Metric name"},
			{ID: 5, Name: "metric_value", Type: "double", Required: true, Doc: "Metric value"},
			{ID: 6, Name: "timestamp", Type: "timestamp", Required: true, Doc: "Collection timestamp"},
			{ID: 7, Name: "tags_json", Type: "string", Required: false, Doc: "JSON tags"},
		},
	}
}

// TableInfo represents information about an Iceberg table
type TableInfo struct {
	Namespace  string
	Name       string
	Schema     *TableSchema
	Location   string
	Properties map[string]string
}

// CreateTable creates a new Iceberg table
func (c *IcebergClient) CreateTable(ctx context.Context, tableName string, schema *TableSchema) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Iceberg catalog")
	}

	url := fmt.Sprintf("%s/v1/namespaces/%s/tables", c.config.CatalogURI, c.config.Namespace)

	payload := map[string]interface{}{
		"name":   tableName,
		"schema": schema,
		"properties": map[string]string{
			"write.format.default": "parquet",
			"write.parquet.compression-codec": "zstd",
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create table: %s", string(body))
	}

	return nil
}

// ListTables returns all tables in the namespace
func (c *IcebergClient) ListTables(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Iceberg catalog")
	}

	url := fmt.Sprintf("%s/v1/namespaces/%s/tables", c.config.CatalogURI, c.config.Namespace)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list tables: status %d", resp.StatusCode)
	}

	var result struct {
		Identifiers []struct {
			Namespace []string `json:"namespace"`
			Name      string   `json:"name"`
		} `json:"identifiers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var tables []string
	for _, id := range result.Identifiers {
		tables = append(tables, id.Name)
	}

	return tables, nil
}

// GetTable returns information about a table
func (c *IcebergClient) GetTable(ctx context.Context, tableName string) (*TableInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Iceberg catalog")
	}

	url := fmt.Sprintf("%s/v1/namespaces/%s/tables/%s", c.config.CatalogURI, c.config.Namespace, tableName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get table: status %d", resp.StatusCode)
	}

	var result struct {
		MetadataLocation string            `json:"metadata-location"`
		Metadata         struct {
			Schema     *TableSchema      `json:"schema"`
			Properties map[string]string `json:"properties"`
			Location   string            `json:"location"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &TableInfo{
		Namespace:  c.config.Namespace,
		Name:       tableName,
		Schema:     result.Metadata.Schema,
		Location:   result.Metadata.Location,
		Properties: result.Metadata.Properties,
	}, nil
}

// DropTable drops an Iceberg table
func (c *IcebergClient) DropTable(ctx context.Context, tableName string, purge bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Iceberg catalog")
	}

	url := fmt.Sprintf("%s/v1/namespaces/%s/tables/%s?purgeRequested=%t",
		c.config.CatalogURI, c.config.Namespace, tableName, purge)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to drop table: %s", string(body))
	}

	return nil
}

// TableExists checks if a table exists
func (c *IcebergClient) TableExists(ctx context.Context, tableName string) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return false, fmt.Errorf("not connected to Iceberg catalog")
	}

	url := fmt.Sprintf("%s/v1/namespaces/%s/tables/%s", c.config.CatalogURI, c.config.Namespace, tableName)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, err
	}
	resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// EnsureVerificationTables ensures all verification tables exist
func (c *IcebergClient) EnsureVerificationTables(ctx context.Context) error {
	tables := []struct {
		name   string
		schema *TableSchema
	}{
		{"verification_results", VerificationResultsSchema()},
		{"model_metrics", ModelMetricsSchema()},
	}

	for _, t := range tables {
		exists, err := c.TableExists(ctx, t.name)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", t.name, err)
		}
		if !exists {
			if err := c.CreateTable(ctx, t.name, t.schema); err != nil {
				return fmt.Errorf("failed to create table %s: %w", t.name, err)
			}
		}
	}

	return nil
}

// SnapshotInfo represents information about a table snapshot
type SnapshotInfo struct {
	SnapshotID    int64     `json:"snapshot-id"`
	TimestampMS   int64     `json:"timestamp-ms"`
	ManifestList  string    `json:"manifest-list"`
	Summary       map[string]string `json:"summary"`
}

// ListSnapshots lists snapshots for a table
func (c *IcebergClient) ListSnapshots(ctx context.Context, tableName string) ([]*SnapshotInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Iceberg catalog")
	}

	// Get table metadata to access snapshots
	tableInfo, err := c.GetTable(ctx, tableName)
	if err != nil {
		return nil, err
	}

	// Note: The REST catalog doesn't directly expose snapshots in the standard API
	// This would require fetching from the metadata file location
	_ = tableInfo

	return nil, nil
}
