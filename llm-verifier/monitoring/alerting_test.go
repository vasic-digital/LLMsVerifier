package monitoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAlertManager(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	assert.NotNil(t, am)
	assert.NotNil(t, am.alerts)
	assert.NotNil(t, am.rules)
	assert.Equal(t, tracker, am.metricsTracker)
}

func TestAlertManager_DefaultRules(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	rules := am.GetRules()

	// Should have default rules
	assert.GreaterOrEqual(t, len(rules), 3)

	// Check for specific default rules
	ruleIDs := make(map[string]bool)
	for _, rule := range rules {
		ruleIDs[rule.ID] = true
	}

	assert.True(t, ruleIDs["high_error_rate"], "Should have high_error_rate rule")
	assert.True(t, ruleIDs["high_response_time"], "Should have high_response_time rule")
	assert.True(t, ruleIDs["scheduler_failures"], "Should have scheduler_failures rule")
}

func TestAlertManager_CreateAlert(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	alert := am.CreateAlert(
		"Test Alert",
		"test_rule",
		"This is a test alert",
		AlertSeverityWarning,
	)

	require.NotNil(t, alert)
	assert.NotEmpty(t, alert.ID)
	assert.Equal(t, "Test Alert", alert.Name)
	assert.Equal(t, "test_rule", alert.Rule)
	assert.Equal(t, "This is a test alert", alert.Message)
	assert.Equal(t, AlertSeverityWarning, alert.Severity)
	assert.Equal(t, AlertStatusActive, alert.Status)
	assert.True(t, alert.Active)
}

func TestAlertManager_GetActiveAlerts(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	// Initially no alerts
	alerts := am.GetActiveAlerts()
	assert.Empty(t, alerts)

	// Create some alerts
	am.CreateAlert("Alert 1", "rule1", "Message 1", AlertSeverityInfo)
	am.CreateAlert("Alert 2", "rule2", "Message 2", AlertSeverityWarning)
	am.CreateAlert("Alert 3", "rule3", "Message 3", AlertSeverityError)

	// Should have 3 active alerts
	alerts = am.GetActiveAlerts()
	assert.Len(t, alerts, 3)
}

func TestAlertManager_AcknowledgeAlert(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	alert := am.CreateAlert("Test Alert", "test", "Test", AlertSeverityWarning)

	err := am.AcknowledgeAlert(alert.ID)
	require.NoError(t, err)

	// Alert should be acknowledged
	alerts := am.GetAllAlerts()
	for _, a := range alerts {
		if a.ID == alert.ID {
			assert.Equal(t, AlertStatusAcknowledged, a.Status)
			assert.NotNil(t, a.AcknowledgedAt)
			return
		}
	}
	t.Error("Alert not found after acknowledgement")
}

func TestAlertManager_ResolveAlert(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	alert := am.CreateAlert("Test Alert", "test", "Test", AlertSeverityWarning)

	err := am.ResolveAlert(alert.ID)
	require.NoError(t, err)

	// Alert should be resolved and inactive
	alerts := am.GetAllAlerts()
	for _, a := range alerts {
		if a.ID == alert.ID {
			assert.Equal(t, AlertStatusResolved, a.Status)
			assert.False(t, a.Active)
			assert.NotNil(t, a.ResolvedAt)
			return
		}
	}
	t.Error("Alert not found after resolution")
}

func TestAlertManager_GetAlertsByStatus(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	alert1 := am.CreateAlert("Alert 1", "rule", "msg", AlertSeverityInfo)
	alert2 := am.CreateAlert("Alert 2", "rule", "msg", AlertSeverityInfo)
	am.CreateAlert("Alert 3", "rule", "msg", AlertSeverityInfo)

	am.AcknowledgeAlert(alert1.ID)
	am.ResolveAlert(alert2.ID)

	activeAlerts := am.GetAlertsByStatus(AlertStatusActive)
	assert.Len(t, activeAlerts, 1)

	acknowledgedAlerts := am.GetAlertsByStatus(AlertStatusAcknowledged)
	assert.Len(t, acknowledgedAlerts, 1)

	resolvedAlerts := am.GetAlertsByStatus(AlertStatusResolved)
	assert.Len(t, resolvedAlerts, 1)
}

func TestAlertManager_GetAlertsBySeverity(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	am.CreateAlert("Info Alert", "rule", "msg", AlertSeverityInfo)
	am.CreateAlert("Warning 1", "rule", "msg", AlertSeverityWarning)
	am.CreateAlert("Warning 2", "rule", "msg", AlertSeverityWarning)
	am.CreateAlert("Error Alert", "rule", "msg", AlertSeverityError)
	am.CreateAlert("Critical Alert", "rule", "msg", AlertSeverityCritical)

	infoAlerts := am.GetAlertsBySeverity(AlertSeverityInfo)
	assert.Len(t, infoAlerts, 1)

	warningAlerts := am.GetAlertsBySeverity(AlertSeverityWarning)
	assert.Len(t, warningAlerts, 2)

	errorAlerts := am.GetAlertsBySeverity(AlertSeverityError)
	assert.Len(t, errorAlerts, 1)

	criticalAlerts := am.GetAlertsBySeverity(AlertSeverityCritical)
	assert.Len(t, criticalAlerts, 1)
}

func TestAlertManager_GetAlertCount(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	am.CreateAlert("Info Alert", "rule", "msg", AlertSeverityInfo)
	am.CreateAlert("Warning 1", "rule", "msg", AlertSeverityWarning)
	am.CreateAlert("Warning 2", "rule", "msg", AlertSeverityWarning)
	am.CreateAlert("Error Alert", "rule", "msg", AlertSeverityError)

	counts := am.GetAlertCount()

	assert.Equal(t, 1, counts[AlertSeverityInfo])
	assert.Equal(t, 2, counts[AlertSeverityWarning])
	assert.Equal(t, 1, counts[AlertSeverityError])
	assert.Equal(t, 0, counts[AlertSeverityCritical])
}

func TestAlertManager_AddRule(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	initialCount := len(am.GetRules())

	am.AddRule(&AlertRule{
		ID:          "custom_rule",
		Name:        "Custom Rule",
		Description: "A custom alert rule",
		Condition:   "custom > threshold",
		Threshold:   100,
		Severity:    AlertSeverityCritical,
		Enabled:     true,
	})

	rules := am.GetRules()
	assert.Len(t, rules, initialCount+1)

	// Find the custom rule
	found := false
	for _, rule := range rules {
		if rule.ID == "custom_rule" {
			found = true
			assert.Equal(t, "Custom Rule", rule.Name)
			assert.Equal(t, AlertSeverityCritical, rule.Severity)
		}
	}
	assert.True(t, found, "Custom rule should be found")
}

func TestAlertManager_CleanupOldAlerts(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	// Create and resolve an alert
	alert := am.CreateAlert("Old Alert", "rule", "msg", AlertSeverityInfo)
	am.ResolveAlert(alert.ID)

	// Create a current alert
	am.CreateAlert("Current Alert", "rule", "msg", AlertSeverityInfo)

	// Clean up alerts older than 1 hour (none should be cleaned)
	removed := am.CleanupOldAlerts(time.Hour)
	assert.Equal(t, 0, removed)

	// All alerts should still be there
	allAlerts := am.GetAllAlerts()
	assert.Len(t, allAlerts, 2)
}

func TestAlertManager_Listener(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	receivedEvents := make(chan struct {
		alert     *Alert
		eventType string
	}, 10)

	am.AddListener(func(alert *Alert, eventType string) {
		receivedEvents <- struct {
			alert     *Alert
			eventType string
		}{alert, eventType}
	})

	alert := am.CreateAlert("Test Alert", "rule", "msg", AlertSeverityInfo)

	// Wait for listener to be called
	select {
	case event := <-receivedEvents:
		assert.Equal(t, alert.ID, event.alert.ID)
		assert.Equal(t, "created", event.eventType)
	case <-time.After(time.Second):
		t.Error("Listener was not called")
	}

	// Test resolve event
	am.ResolveAlert(alert.ID)

	select {
	case event := <-receivedEvents:
		assert.Equal(t, alert.ID, event.alert.ID)
		assert.Equal(t, "resolved", event.eventType)
	case <-time.After(time.Second):
		t.Error("Listener was not called for resolve")
	}
}

func TestCriticalMetricsTracker_GetPerformanceReport(t *testing.T) {
	cmt := NewCriticalMetricsTracker()

	report := cmt.GetPerformanceReport(nil)

	assert.NotNil(t, report)
	assert.Equal(t, "ok", report["status"])
	assert.NotNil(t, report["timestamp"])
}

func TestAlertSeverityConstants(t *testing.T) {
	assert.Equal(t, AlertSeverity("info"), AlertSeverityInfo)
	assert.Equal(t, AlertSeverity("warning"), AlertSeverityWarning)
	assert.Equal(t, AlertSeverity("error"), AlertSeverityError)
	assert.Equal(t, AlertSeverity("critical"), AlertSeverityCritical)
}

func TestAlertStatusConstants(t *testing.T) {
	assert.Equal(t, AlertStatus("active"), AlertStatusActive)
	assert.Equal(t, AlertStatus("acknowledged"), AlertStatusAcknowledged)
	assert.Equal(t, AlertStatus("resolved"), AlertStatusResolved)
}
