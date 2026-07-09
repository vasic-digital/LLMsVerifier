package streaming

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

// FlinkConfig holds Flink connection configuration
type FlinkConfig struct {
	JobManagerHost string        `yaml:"jobmanager_host" json:"jobmanager_host"`
	JobManagerPort int           `yaml:"jobmanager_port" json:"jobmanager_port"`
	WebUIPort      int           `yaml:"web_ui_port" json:"web_ui_port"`
	RESTURL        string        `yaml:"rest_url" json:"rest_url"`
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`
}

// DefaultFlinkConfig returns default Flink configuration
func DefaultFlinkConfig() *FlinkConfig {
	return &FlinkConfig{
		JobManagerHost: "localhost",
		JobManagerPort: 6123,
		WebUIPort:      8082,
		RESTURL:        "http://localhost:8082",
		RequestTimeout: 30 * time.Second,
	}
}

// Validate validates the Flink configuration
func (c *FlinkConfig) Validate() error {
	if c.JobManagerHost == "" {
		return fmt.Errorf("jobmanager_host is required")
	}
	if c.JobManagerPort <= 0 || c.JobManagerPort > 65535 {
		return fmt.Errorf("jobmanager_port must be between 1 and 65535")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive")
	}
	return nil
}

// GetRESTURL returns the REST URL for Flink
func (c *FlinkConfig) GetRESTURL() string {
	if c.RESTURL != "" {
		return c.RESTURL
	}
	return fmt.Sprintf("http://%s:%d", c.JobManagerHost, c.WebUIPort)
}

// FlinkClient wraps the Flink REST API client for LLMsVerifier
type FlinkClient struct {
	config    *FlinkConfig
	client    *http.Client
	mu        sync.RWMutex
	connected bool
}

// NewFlinkClient creates a new Flink client
func NewFlinkClient(config *FlinkConfig) (*FlinkClient, error) {
	if config == nil {
		config = DefaultFlinkConfig()
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &FlinkClient{
		config: config,
		client: &http.Client{
			Timeout: config.RequestTimeout,
		},
		connected: false,
	}, nil
}

// Connect establishes connection to Flink
func (c *FlinkClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Verify connection by getting cluster overview
	url := fmt.Sprintf("%s/overview", c.config.GetRESTURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Flink: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to connect to Flink: status %d", resp.StatusCode)
	}

	c.connected = true
	return nil
}

// Close closes the client connection
func (c *FlinkClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.connected = false
	return nil
}

// IsConnected returns whether the client is connected
func (c *FlinkClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// HealthCheck performs a health check on Flink
func (c *FlinkClient) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/overview", c.config.GetRESTURL())
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

// ClusterOverview represents the Flink cluster overview
type ClusterOverview struct {
	TaskManagers   int    `json:"taskmanagers"`
	SlotsTotal     int    `json:"slots-total"`
	SlotsAvailable int    `json:"slots-available"`
	JobsRunning    int    `json:"jobs-running"`
	JobsFinished   int    `json:"jobs-finished"`
	JobsCancelled  int    `json:"jobs-cancelled"`
	JobsFailed     int    `json:"jobs-failed"`
	FlinkVersion   string `json:"flink-version"`
	FlinkCommit    string `json:"flink-commit"`
}

// GetClusterOverview returns the cluster overview
func (c *FlinkClient) GetClusterOverview(ctx context.Context) (*ClusterOverview, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/overview", c.config.GetRESTURL())
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
		return nil, fmt.Errorf("failed to get cluster overview: status %d", resp.StatusCode)
	}

	var overview ClusterOverview
	if err := json.NewDecoder(resp.Body).Decode(&overview); err != nil {
		return nil, err
	}

	return &overview, nil
}

// JobInfo represents information about a Flink job
type JobInfo struct {
	ID        string `json:"jid"`
	Name      string `json:"name"`
	State     string `json:"state"`
	StartTime int64  `json:"start-time"`
	EndTime   int64  `json:"end-time"`
	Duration  int64  `json:"duration"`
}

// ListJobs returns all jobs
func (c *FlinkClient) ListJobs(ctx context.Context) ([]*JobInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/jobs/overview", c.config.GetRESTURL())
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
		return nil, fmt.Errorf("failed to list jobs: status %d", resp.StatusCode)
	}

	var result struct {
		Jobs []*JobInfo `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Jobs, nil
}

// GetJob returns information about a specific job
func (c *FlinkClient) GetJob(ctx context.Context, jobID string) (*JobInfo, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/jobs/%s", c.config.GetRESTURL(), jobID)
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
		return nil, fmt.Errorf("failed to get job: status %d", resp.StatusCode)
	}

	var job JobInfo
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

// CancelJob cancels a running job
func (c *FlinkClient) CancelJob(ctx context.Context, jobID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/jobs/%s", c.config.GetRESTURL(), jobID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to cancel job: %s", string(body))
	}

	return nil
}

// VerificationStreamConfig represents configuration for verification streaming
type VerificationStreamConfig struct {
	KafkaBootstrapServers string `json:"kafka_bootstrap_servers"`
	InputTopic            string `json:"input_topic"`
	OutputTopic           string `json:"output_topic"`
	GroupID               string `json:"group_id"`
	Parallelism           int    `json:"parallelism"`
}

// DefaultVerificationStreamConfig returns default verification streaming configuration
func DefaultVerificationStreamConfig() *VerificationStreamConfig {
	return &VerificationStreamConfig{
		KafkaBootstrapServers: "localhost:9092",
		InputTopic:            "llmsverifier.verification.requests",
		OutputTopic:           "llmsverifier.verification.results",
		GroupID:               "llmsverifier-stream",
		Parallelism:           4,
	}
}

// SubmitVerificationJob submits a verification streaming job
func (c *FlinkClient) SubmitVerificationJob(ctx context.Context, jarID string, config *VerificationStreamConfig) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return "", fmt.Errorf("not connected to Flink")
	}

	if config == nil {
		config = DefaultVerificationStreamConfig()
	}

	url := fmt.Sprintf("%s/jars/%s/run", c.config.GetRESTURL(), jarID)

	programArgs := fmt.Sprintf("--bootstrap-servers %s --input-topic %s --output-topic %s --group-id %s",
		config.KafkaBootstrapServers,
		config.InputTopic,
		config.OutputTopic,
		config.GroupID,
	)

	payload := map[string]interface{}{
		"parallelism": config.Parallelism,
		"programArgs": programArgs,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to submit job: %s", string(body))
	}

	var result struct {
		JobID string `json:"jobid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.JobID, nil
}

// UploadJar uploads a JAR file to Flink
func (c *FlinkClient) UploadJar(ctx context.Context, jarContent io.Reader) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return "", fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/jars/upload", c.config.GetRESTURL())

	// Create multipart request
	var body bytes.Buffer
	body.WriteString("--boundary\r\n")
	body.WriteString("Content-Disposition: form-data; name=\"jarfile\"; filename=\"job.jar\"\r\n")
	body.WriteString("Content-Type: application/java-archive\r\n\r\n")

	jarBytes, err := io.ReadAll(jarContent)
	if err != nil {
		return "", err
	}
	body.Write(jarBytes)
	body.WriteString("\r\n--boundary--\r\n")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload JAR: %s", string(respBody))
	}

	var result struct {
		Filename string `json:"filename"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Extract JAR ID from filename (e.g., "c6ab0c1a-xxx/job.jar")
	return result.Filename, nil
}

// ListJars returns all uploaded JARs
func (c *FlinkClient) ListJars(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/jars", c.config.GetRESTURL())
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
		return nil, fmt.Errorf("failed to list JARs: status %d", resp.StatusCode)
	}

	var result struct {
		Files []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var jars []string
	for _, f := range result.Files {
		jars = append(jars, f.ID)
	}

	return jars, nil
}

// DeleteJar deletes an uploaded JAR
func (c *FlinkClient) DeleteJar(ctx context.Context, jarID string) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected {
		return fmt.Errorf("not connected to Flink")
	}

	url := fmt.Sprintf("%s/jars/%s", c.config.GetRESTURL(), jarID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete JAR: status %d", resp.StatusCode)
	}

	return nil
}
