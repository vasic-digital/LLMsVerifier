package scoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAlertManagerFixed(t *testing.T) {
	config := MonitoringConfig{
		Enabled:              true,
		ScoreChangeThreshold: 10.0,
		AlertCooldownPeriod:  5 * time.Minute,
	}

	am := NewAlertManagerFixed(config)

	require.NotNil(t, am)
	assert.Equal(t, config, am.config)
	assert.NotNil(t, am.sentAlerts)
	assert.NotNil(t, am.httpClient)
}

func TestAlertManagerFixed_SendScoreChangeAlert_Disabled(t *testing.T) {
	config := MonitoringConfig{
		Enabled: false, // Alerts disabled
	}

	am := NewAlertManagerFixed(config)

	alert := ScoreChangeAlert{
		ModelID:     "gpt-4",
		OldScore:    85.0,
		NewScore:    75.0,
		ScoreChange: -10.0,
		Severity:    "warning",
	}

	err := am.SendScoreChangeAlert(alert)
	assert.NoError(t, err) // Should return nil when disabled
}

func TestAlertManagerFixed_SendScoreChangeAlert_Cooldown(t *testing.T) {
	config := MonitoringConfig{
		Enabled:             true,
		AlertCooldownPeriod: 1 * time.Hour,
	}

	am := NewAlertManagerFixed(config)

	// Record an alert as sent
	am.recordAlertSent("score_change_gpt-4")

	alert := ScoreChangeAlert{
		ModelID:     "gpt-4",
		OldScore:    85.0,
		NewScore:    75.0,
		ScoreChange: -10.0,
		Severity:    "warning",
	}

	// Should be in cooldown
	err := am.SendScoreChangeAlert(alert)
	assert.NoError(t, err)

	// Check that alert is in cooldown
	assert.True(t, am.isInCooldown("score_change_gpt-4"))
}

func TestAlertManagerFixed_SendScoreChangeAlert_WithWebhook(t *testing.T) {
	// Create a test server to receive webhook
	webhookReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		webhookReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := MonitoringConfig{
		Enabled:             true,
		EnableWebhookAlerts: true,
		WebhookURL:          server.URL,
		AlertCooldownPeriod: 0, // No cooldown for testing
	}

	am := NewAlertManagerFixed(config)

	alert := ScoreChangeAlert{
		ModelID:     "gpt-4",
		OldScore:    85.0,
		NewScore:    75.0,
		ScoreChange: -10.0,
		Severity:    "critical",
		Timestamp:   time.Now(),
	}

	err := am.SendScoreChangeAlert(alert)
	assert.NoError(t, err)
	assert.True(t, webhookReceived)
}

func TestAlertManagerFixed_SendAPIPerformanceAlert_Disabled(t *testing.T) {
	config := MonitoringConfig{
		Enabled: false,
	}

	am := NewAlertManagerFixed(config)

	alert := APIPerformanceAlert{
		APIName:      "openai",
		ResponseTime: 5 * time.Second,
		Success:      false,
		Threshold:    2 * time.Second,
	}

	err := am.SendAPIPerformanceAlert(alert)
	assert.NoError(t, err)
}

func TestAlertManagerFixed_SendAPIPerformanceAlert_Cooldown(t *testing.T) {
	config := MonitoringConfig{
		Enabled:             true,
		AlertCooldownPeriod: 1 * time.Hour,
	}

	am := NewAlertManagerFixed(config)

	// Record an alert as sent
	am.recordAlertSent("api_performance_openai")

	alert := APIPerformanceAlert{
		APIName:      "openai",
		ResponseTime: 5 * time.Second,
		Success:      false,
		Threshold:    2 * time.Second,
	}

	err := am.SendAPIPerformanceAlert(alert)
	assert.NoError(t, err)

	assert.True(t, am.isInCooldown("api_performance_openai"))
}

func TestAlertManagerFixed_SendDatabasePerformanceAlert_Disabled(t *testing.T) {
	config := MonitoringConfig{
		Enabled: false,
	}

	am := NewAlertManagerFixed(config)

	alert := DatabasePerformanceAlert{
		Operation: "SELECT",
		Latency:   500 * time.Millisecond,
		Threshold: 100 * time.Millisecond,
	}

	err := am.SendDatabasePerformanceAlert(alert)
	assert.NoError(t, err)
}

func TestAlertManagerFixed_IsInCooldown(t *testing.T) {
	config := MonitoringConfig{
		Enabled:             true,
		AlertCooldownPeriod: 1 * time.Hour,
	}

	am := NewAlertManagerFixed(config)

	// Not in cooldown initially
	assert.False(t, am.isInCooldown("test_alert"))

	// Record alert
	am.recordAlertSent("test_alert")

	// Now should be in cooldown
	assert.True(t, am.isInCooldown("test_alert"))
}

func TestAlertManagerFixed_IsInCooldown_Expired(t *testing.T) {
	config := MonitoringConfig{
		Enabled:             true,
		AlertCooldownPeriod: 1 * time.Millisecond, // Very short cooldown
	}

	am := NewAlertManagerFixed(config)

	// Record alert
	am.recordAlertSent("test_alert")

	// Wait for cooldown to expire
	time.Sleep(10 * time.Millisecond)

	// Should not be in cooldown anymore
	assert.False(t, am.isInCooldown("test_alert"))
}

func TestAlertManagerFixed_RecordAlertSent(t *testing.T) {
	config := MonitoringConfig{
		Enabled: true,
	}

	am := NewAlertManagerFixed(config)

	// Record multiple alerts
	am.recordAlertSent("alert1")
	am.recordAlertSent("alert2")
	am.recordAlertSent("alert3")

	am.mu.RLock()
	defer am.mu.RUnlock()

	assert.Len(t, am.sentAlerts, 3)
	assert.Contains(t, am.sentAlerts, "alert1")
	assert.Contains(t, am.sentAlerts, "alert2")
	assert.Contains(t, am.sentAlerts, "alert3")
}

func TestAlertManagerFixed_CleanupOldRecords(t *testing.T) {
	config := MonitoringConfig{
		Enabled:             true,
		AlertCooldownPeriod: 1 * time.Millisecond, // Very short for testing
	}

	am := NewAlertManagerFixed(config)

	// Record old alerts
	am.mu.Lock()
	am.sentAlerts["old_alert"] = time.Now().Add(-1 * time.Hour) // Old alert
	am.sentAlerts["new_alert"] = time.Now()                      // Recent alert
	am.mu.Unlock()

	// Cleanup with 30 minute max age
	am.CleanupOldRecords(30 * time.Minute)

	am.mu.RLock()
	defer am.mu.RUnlock()

	// Old alert should be removed
	assert.NotContains(t, am.sentAlerts, "old_alert")
	// New alert should still be there
	assert.Contains(t, am.sentAlerts, "new_alert")
}

func TestAlertManagerFixed_GetAlertStats(t *testing.T) {
	config := MonitoringConfig{
		Enabled: true,
	}

	am := NewAlertManagerFixed(config)

	// Record some alerts
	am.recordAlertSent("alert1")
	am.recordAlertSent("alert2")
	am.recordAlertSent("alert3")

	stats := am.GetAlertStats()

	assert.Equal(t, 3, stats.TotalAlerts)
	assert.NotNil(t, stats.ByType)
	assert.NotNil(t, stats.BySeverity)
}

func TestAlertManagerFixed_SendWebhookAlert_Error(t *testing.T) {
	// Server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := MonitoringConfig{
		Enabled:             true,
		EnableWebhookAlerts: true,
		WebhookURL:          server.URL,
		AlertCooldownPeriod: 0,
	}

	am := NewAlertManagerFixed(config)

	alert := ScoreChangeAlert{
		ModelID:     "test-model",
		OldScore:    90.0,
		NewScore:    80.0,
		ScoreChange: -10.0,
		Severity:    "warning",
	}

	// Should handle error gracefully
	err := am.SendScoreChangeAlert(alert)
	assert.NoError(t, err) // The method doesn't return the webhook error
}

func TestAlertManagerFixed_CreateWebhookPayload(t *testing.T) {
	config := MonitoringConfig{
		Enabled: true,
	}

	am := NewAlertManagerFixed(config)

	t.Run("score change alert", func(t *testing.T) {
		alert := ScoreChangeAlert{
			ModelID:     "gpt-4",
			OldScore:    85.0,
			NewScore:    75.0,
			ScoreChange: -10.0,
			Severity:    "warning",
			Message:     "Score decreased significantly",
			Timestamp:   time.Now(),
		}

		payload, err := am.createWebhookPayload(alert)
		require.NoError(t, err)
		assert.NotNil(t, payload)
		assert.Equal(t, "score_change", payload["type"])
		assert.Equal(t, "gpt-4", payload["model_id"])
	})

	t.Run("api performance alert", func(t *testing.T) {
		alert := APIPerformanceAlert{
			APIName:      "openai",
			ResponseTime: 5 * time.Second,
			Success:      false,
			Threshold:    2 * time.Second,
		}

		payload, err := am.createWebhookPayload(alert)
		require.NoError(t, err)
		assert.NotNil(t, payload)
		assert.Equal(t, "api_performance", payload["type"])
	})

	t.Run("database performance alert", func(t *testing.T) {
		alert := DatabasePerformanceAlert{
			Operation: "SELECT",
			Latency:   500 * time.Millisecond,
			Threshold: 100 * time.Millisecond,
		}

		payload, err := am.createWebhookPayload(alert)
		require.NoError(t, err)
		assert.NotNil(t, payload)
		assert.Equal(t, "database_performance", payload["type"])
	})

	t.Run("unknown alert type", func(t *testing.T) {
		_, err := am.createWebhookPayload("unknown")
		assert.Error(t, err)
	})
}

func TestAlertManagerFixed_FormatScoreChangeEmail(t *testing.T) {
	config := MonitoringConfig{
		Enabled: true,
	}

	am := NewAlertManagerFixed(config)

	alert := ScoreChangeAlert{
		ModelID:     "gpt-4",
		OldScore:    85.0,
		NewScore:    75.0,
		ScoreChange: -10.0,
		Severity:    "critical",
		Message:     "Score dropped significantly",
		Timestamp:   time.Now(),
	}

	body := am.formatScoreChangeEmail(alert)

	assert.Contains(t, body, "Model Score Change Alert")
	assert.Contains(t, body, "gpt-4")
	assert.Contains(t, body, "85")
	assert.Contains(t, body, "75")
	assert.Contains(t, body, "decreased")
}

func TestAlertManagerFixed_FormatAPIPerformanceEmail(t *testing.T) {
	config := MonitoringConfig{
		Enabled: true,
	}

	am := NewAlertManagerFixed(config)

	alert := APIPerformanceAlert{
		APIName:      "openai",
		ResponseTime: 5 * time.Second,
		Success:      false,
		Threshold:    2 * time.Second,
		Message:      "API response time exceeded threshold",
		Timestamp:    time.Now(),
	}

	body := am.formatAPIPerformanceEmail(alert)

	assert.Contains(t, body, "API Performance Alert")
	assert.Contains(t, body, "openai")
	assert.Contains(t, body, "Failed")
}

func TestAlertManagerFixed_FormatDatabasePerformanceEmail(t *testing.T) {
	config := MonitoringConfig{
		Enabled: true,
	}

	am := NewAlertManagerFixed(config)

	alert := DatabasePerformanceAlert{
		Operation: "SELECT",
		Latency:   500 * time.Millisecond,
		Threshold: 100 * time.Millisecond,
		Message:   "Database latency exceeded threshold",
		Timestamp: time.Now(),
	}

	body := am.formatDatabasePerformanceEmail(alert)

	assert.Contains(t, body, "Database Performance Alert")
	assert.Contains(t, body, "SELECT")
}
