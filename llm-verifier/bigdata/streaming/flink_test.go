package streaming

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFlinkConfig(t *testing.T) {
	config := DefaultFlinkConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "localhost", config.JobManagerHost)
	assert.Equal(t, 6123, config.JobManagerPort)
	assert.Equal(t, 8082, config.WebUIPort)
	assert.Equal(t, "http://localhost:8082", config.RESTURL)
	assert.Equal(t, 30*time.Second, config.RequestTimeout)
}

func TestFlinkConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*FlinkConfig)
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			modify:      func(c *FlinkConfig) {},
			expectError: false,
		},
		{
			name: "empty jobmanager host",
			modify: func(c *FlinkConfig) {
				c.JobManagerHost = ""
			},
			expectError: true,
			errorMsg:    "jobmanager_host is required",
		},
		{
			name: "invalid jobmanager port",
			modify: func(c *FlinkConfig) {
				c.JobManagerPort = 0
			},
			expectError: true,
			errorMsg:    "jobmanager_port must be between 1 and 65535",
		},
		{
			name: "invalid request timeout",
			modify: func(c *FlinkConfig) {
				c.RequestTimeout = 0
			},
			expectError: true,
			errorMsg:    "request_timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultFlinkConfig()
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

func TestFlinkConfigGetRESTURL(t *testing.T) {
	t.Run("returns configured URL", func(t *testing.T) {
		config := DefaultFlinkConfig()
		config.RESTURL = "http://custom:9999"

		assert.Equal(t, "http://custom:9999", config.GetRESTURL())
	})

	t.Run("constructs URL from host and port when empty", func(t *testing.T) {
		config := DefaultFlinkConfig()
		config.RESTURL = ""
		config.JobManagerHost = "flink-host"
		config.WebUIPort = 8081

		assert.Equal(t, "http://flink-host:8081", config.GetRESTURL())
	})
}

func TestNewFlinkClient(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		client, err := NewFlinkClient(nil)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.False(t, client.IsConnected())
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &FlinkConfig{
			JobManagerHost: "flink.example.com",
			JobManagerPort: 6123,
			WebUIPort:      8082,
			RequestTimeout: 60 * time.Second,
		}
		client, err := NewFlinkClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("with invalid config", func(t *testing.T) {
		config := &FlinkConfig{
			JobManagerHost: "",
		}
		client, err := NewFlinkClient(config)
		require.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestDefaultVerificationStreamConfig(t *testing.T) {
	config := DefaultVerificationStreamConfig()

	assert.Equal(t, "localhost:9092", config.KafkaBootstrapServers)
	assert.Equal(t, "llmsverifier.verification.requests", config.InputTopic)
	assert.Equal(t, "llmsverifier.verification.results", config.OutputTopic)
	assert.Equal(t, "llmsverifier-stream", config.GroupID)
	assert.Equal(t, 4, config.Parallelism)
}

func TestClusterOverview(t *testing.T) {
	overview := &ClusterOverview{
		TaskManagers:   4,
		SlotsTotal:     16,
		SlotsAvailable: 8,
		JobsRunning:    2,
		JobsFinished:   10,
		JobsCancelled:  1,
		JobsFailed:     0,
		FlinkVersion:   "1.18.0",
		FlinkCommit:    "abc123",
	}

	assert.Equal(t, 4, overview.TaskManagers)
	assert.Equal(t, 16, overview.SlotsTotal)
	assert.Equal(t, 8, overview.SlotsAvailable)
	assert.Equal(t, 2, overview.JobsRunning)
	assert.Equal(t, "1.18.0", overview.FlinkVersion)
}

func TestJobInfo(t *testing.T) {
	job := &JobInfo{
		ID:        "job-123",
		Name:      "VerificationStream",
		State:     "RUNNING",
		StartTime: 1704067200000,
		EndTime:   -1,
		Duration:  3600000,
	}

	assert.Equal(t, "job-123", job.ID)
	assert.Equal(t, "VerificationStream", job.Name)
	assert.Equal(t, "RUNNING", job.State)
}
