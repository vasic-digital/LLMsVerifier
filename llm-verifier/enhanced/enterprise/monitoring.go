package enterprise

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"llm-verifier/monitoring"
)

// EnterpriseMonitor integrates with enterprise monitoring systems
type EnterpriseMonitor struct {
	config         EnterpriseMonitorConfig
	metricsTracker *monitoring.CriticalMetricsTracker
	alertManager   *monitoring.AlertManager
	httpClient     *http.Client
}

// EnterpriseMonitorConfig holds enterprise monitoring configuration
type EnterpriseMonitorConfig struct {
	Enabled        bool            `json:"enabled"`
	Splunk         SplunkConfig    `json:"splunk,omitempty"`
	DataDog        DataDogConfig   `json:"datadog,omitempty"`
	NewRelic       NewRelicConfig  `json:"newrelic,omitempty"`
	ELK            ELKConfig       `json:"elk,omitempty"`
	CustomWebhooks []WebhookConfig `json:"custom_webhooks,omitempty"`
	BatchInterval  time.Duration   `json:"batch_interval"`
	RetryAttempts  int             `json:"retry_attempts"`
	RetryDelay     time.Duration   `json:"retry_delay"`
}

// SplunkConfig holds Splunk integration configuration
type SplunkConfig struct {
	Host       string            `json:"host"`
	Port       int               `json:"port"`
	Token      string            `json:"token"`
	Index      string            `json:"index"`
	Source     string            `json:"source"`
	Sourcetype string            `json:"sourcetype"`
	Fields     map[string]string `json:"fields"`
}

// DataDogConfig holds DataDog integration configuration
type DataDogConfig struct {
	APIKey      string            `json:"api_key"`
	AppKey      string            `json:"app_key"`
	Endpoint    string            `json:"endpoint"`
	Tags        map[string]string `json:"tags"`
	ServiceName string            `json:"service_name"`
	Environment string            `json:"environment"`
}

// NewRelicConfig holds New Relic integration configuration
type NewRelicConfig struct {
	LicenseKey string            `json:"license_key"`
	AppName    string            `json:"app_name"`
	Labels     map[string]string `json:"labels"`
	Endpoint   string            `json:"endpoint"`
}

// ELKConfig holds ELK stack integration configuration
type ELKConfig struct {
	ElasticsearchURL string                 `json:"elasticsearch_url"`
	IndexName        string                 `json:"index_name"`
	Username         string                 `json:"username"`
	Password         string                 `json:"password"`
	Mapping          map[string]interface{} `json:"mapping"`
}

// WebhookConfig holds custom webhook configuration
type WebhookConfig struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers"`
	Template    string            `json:"template"`
	EventTypes  []string          `json:"event_types"`
	RetryConfig RetryConfig       `json:"retry_config"`
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     float64       `json:"backoff"`
}

// NewEnterpriseMonitor creates a new enterprise monitor
func NewEnterpriseMonitor(config EnterpriseMonitorConfig, metricsTracker *monitoring.CriticalMetricsTracker, alertManager *monitoring.AlertManager) *EnterpriseMonitor {
	return &EnterpriseMonitor{
		config:         config,
		metricsTracker: metricsTracker,
		alertManager:   alertManager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start begins enterprise monitoring
func (em *EnterpriseMonitor) Start() error {
	if !em.config.Enabled {
		log.Println("Enterprise monitoring is disabled")
		return nil
	}

	log.Println("Starting enterprise monitoring integration")

	// Start batch processing
	go em.batchProcessor()

	// Start alert forwarding
	go em.alertForwarder()

	return nil
}

// Stop stops enterprise monitoring
func (em *EnterpriseMonitor) Stop() {
	log.Println("Stopping enterprise monitoring integration")
}

// batchProcessor processes metrics in batches
func (em *EnterpriseMonitor) batchProcessor() {
	ticker := time.NewTicker(em.config.BatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			em.processBatch()
		}
	}
}

// processBatch processes a batch of metrics and events
func (em *EnterpriseMonitor) processBatch() {
	// Get current metrics
	report := em.metricsTracker.GetPerformanceReport(5 * time.Minute)

	// Convert to enterprise monitoring format
	em.sendToSplunk(report)
	em.sendToDataDog(report)
	em.sendToNewRelic(report)
	em.sendToELK(report)
	em.sendToWebhooks(report)
}

// alertForwarder forwards alerts to enterprise systems
func (em *EnterpriseMonitor) alertForwarder() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			em.forwardAlerts()
		}
	}
}

// forwardAlerts forwards active alerts to enterprise systems
func (em *EnterpriseMonitor) forwardAlerts() {
	alerts := em.alertManager.GetActiveAlerts()

	for _, activeAlert := range alerts {
		// Convert ActiveAlert to Alert for the methods
		alert := &monitoring.Alert{
			ID:       activeAlert.ID,
			Name:     activeAlert.Rule, // Use rule as name
			Severity: activeAlert.Severity,
			Active:   true,
		}
		em.sendAlertToSplunk(alert)
		em.sendAlertToDataDog(alert)
		em.sendAlertToNewRelic(alert)
		em.sendAlertToELK(alert)
		em.sendAlertToWebhooks(alert, "alert")
	}
}

// sendToSplunk sends data to Splunk
func (em *EnterpriseMonitor) sendToSplunk(data interface{}) {
	if em.config.Splunk.Host == "" {
		return
	}

	payload := em.formatForSplunk(data)
	em.sendWithRetry("Splunk", fmt.Sprintf("http://%s:%d/services/collector",
		em.config.Splunk.Host, em.config.Splunk.Port), payload, em.config.Splunk.Token)
}

// sendToDataDog sends data to DataDog
func (em *EnterpriseMonitor) sendToDataDog(data interface{}) {
	if em.config.DataDog.APIKey == "" {
		return
	}

	payload := em.formatForDataDog(data)
	em.sendWithRetry("DataDog", em.config.DataDog.Endpoint+"/api/v1/series", payload, "")
}

// sendToNewRelic sends data to New Relic
func (em *EnterpriseMonitor) sendToNewRelic(data interface{}) {
	if em.config.NewRelic.LicenseKey == "" {
		return
	}

	payload := em.formatForNewRelic(data)
	em.sendWithRetry("New Relic", em.config.NewRelic.Endpoint+"/v1/data", payload, "")
}

// sendToELK sends data to ELK stack
func (em *EnterpriseMonitor) sendToELK(data interface{}) {
	if em.config.ELK.ElasticsearchURL == "" {
		return
	}

	payload := em.formatForELK(data)
	url := fmt.Sprintf("%s/%s/_doc", em.config.ELK.ElasticsearchURL, em.config.ELK.IndexName)
	em.sendWithRetry("ELK", url, payload, "")
}

// sendToWebhooks sends data to custom webhooks
func (em *EnterpriseMonitor) sendToWebhooks(data interface{}, eventType ...string) {
	for _, webhook := range em.config.CustomWebhooks {
		if len(eventType) > 0 && !em.webhookSupportsEventType(webhook, eventType[0]) {
			continue
		}

		payload := em.formatForWebhook(data, webhook.Template)
		em.sendWebhookWithRetry(webhook, payload)
	}
}

// sendAlertToSplunk sends an alert to Splunk
func (em *EnterpriseMonitor) sendAlertToSplunk(alert *monitoring.Alert) {
	if em.config.Splunk.Host == "" {
		return
	}

	payload := em.formatAlertForSplunk(alert)
	em.sendWithRetry("Splunk Alert", fmt.Sprintf("http://%s:%d/services/collector",
		em.config.Splunk.Host, em.config.Splunk.Port), payload, em.config.Splunk.Token)
}

// sendAlertToDataDog sends an alert to DataDog
func (em *EnterpriseMonitor) sendAlertToDataDog(alert *monitoring.Alert) {
	if em.config.DataDog.APIKey == "" {
		return
	}

	payload := em.formatAlertForDataDog(alert)
	em.sendWithRetry("DataDog Alert", em.config.DataDog.Endpoint+"/api/v1/events", payload, "")
}

// sendAlertToNewRelic sends an alert to New Relic
func (em *EnterpriseMonitor) sendAlertToNewRelic(alert *monitoring.Alert) {
	if em.config.NewRelic.LicenseKey == "" {
		return
	}

	payload := em.formatAlertForNewRelic(alert)
	em.sendWithRetry("New Relic Alert", em.config.NewRelic.Endpoint+"/v1/data", payload, "")
}

// sendAlertToELK sends an alert to ELK
func (em *EnterpriseMonitor) sendAlertToELK(alert *monitoring.Alert) {
	if em.config.ELK.ElasticsearchURL == "" {
		return
	}

	payload := em.formatAlertForELK(alert)
	url := fmt.Sprintf("%s/%s/_doc", em.config.ELK.ElasticsearchURL, em.config.ELK.IndexName)
	em.sendWithRetry("ELK Alert", url, payload, "")
}

// sendAlertToWebhooks sends an alert to webhooks
func (em *EnterpriseMonitor) sendAlertToWebhooks(alert *monitoring.Alert, eventType string) {
	em.sendToWebhooks(alert, eventType)
}

// sendWithRetry sends data with retry logic
func (em *EnterpriseMonitor) sendWithRetry(system, url, payload, authToken string) {
	delay := em.config.RetryDelay

	for attempt := 1; attempt <= em.config.RetryAttempts; attempt++ {
		if err := em.sendHTTPRequest(system, url, payload, authToken); err == nil {
			return // Success
		}

		if attempt < em.config.RetryAttempts {
			log.Printf("%s attempt %d failed, retrying in %v", system, attempt, delay)
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * (1.0 + em.config.RetryDelay.Seconds()*0.1)) // Simple backoff
		}
	}

	log.Printf("%s failed after %d attempts", system, em.config.RetryAttempts)
}

// sendHTTPRequest sends an HTTP request
func (em *EnterpriseMonitor) sendHTTPRequest(system, url, payload, authToken string) error {
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Set("Authorization", "Splunk "+authToken)
	}

	resp, err := em.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// sendWebhookWithRetry sends webhook with retry logic
func (em *EnterpriseMonitor) sendWebhookWithRetry(webhook WebhookConfig, payload string) {
	delay := webhook.RetryConfig.Delay

	for attempt := 1; attempt <= webhook.RetryConfig.MaxAttempts; attempt++ {
		if err := em.sendWebhookRequest(webhook, payload); err == nil {
			return // Success
		}

		if attempt < webhook.RetryConfig.MaxAttempts {
			log.Printf("Webhook %s attempt %d failed, retrying in %v", webhook.URL, attempt, delay)
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * webhook.RetryConfig.Backoff)
		}
	}

	log.Printf("Webhook %s failed after %d attempts", webhook.URL, webhook.RetryConfig.MaxAttempts)
}

// sendWebhookRequest sends a webhook request
func (em *EnterpriseMonitor) sendWebhookRequest(webhook WebhookConfig, payload string) error {
	req, err := http.NewRequest(webhook.Method, webhook.URL, strings.NewReader(payload))
	if err != nil {
		return err
	}

	// Set headers
	for key, value := range webhook.Headers {
		req.Header.Set(key, value)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := em.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// Format methods for different systems
func (em *EnterpriseMonitor) formatForSplunk(data interface{}) string {
	event := map[string]interface{}{
		"time":       time.Now().Unix(),
		"host":       "llm-verifier",
		"source":     em.config.Splunk.Source,
		"sourcetype": em.config.Splunk.Sourcetype,
		"event":      data,
	}

	for k, v := range em.config.Splunk.Fields {
		event[k] = v
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func (em *EnterpriseMonitor) formatForDataDog(data interface{}) string {
	// Simplified DataDog format
	series := []map[string]interface{}{
		{
			"metric": "llm.health_score",
			"points": [][]interface{}{{time.Now().Unix(), data.(map[string]interface{})["health_score"]}},
			"type":   "gauge",
			"tags":   em.formatDataDogTags(),
		},
	}

	jsonData, _ := json.Marshal(map[string]interface{}{"series": series})
	return string(jsonData)
}

func (em *EnterpriseMonitor) formatForNewRelic(data interface{}) string {
	metrics := []map[string]interface{}{
		{
			"name":       "llm.health_score",
			"type":       "gauge",
			"value":      data.(map[string]interface{})["health_score"],
			"timestamp":  time.Now().Unix() * 1000,
			"attributes": em.config.NewRelic.Labels,
		},
	}

	jsonData, err := json.Marshal(metrics)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func (em *EnterpriseMonitor) formatForELK(data interface{}) string {
	doc := map[string]interface{}{
		"@timestamp": time.Now().Format(time.RFC3339),
		"service":    "llm-verifier",
		"data":       data,
	}

	jsonData, _ := json.Marshal(doc)
	return string(jsonData)
}

func (em *EnterpriseMonitor) formatForWebhook(data interface{}, template string) string {
	// Simple template replacement (in real implementation, use proper templating)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	payload := fmt.Sprintf(`{"data": %s, "timestamp": "%s"}`,
		string(jsonData), time.Now().Format(time.RFC3339))
	return payload
}

func (em *EnterpriseMonitor) formatAlertForSplunk(alert *monitoring.Alert) string {
	return em.formatForSplunk(alert)
}

func (em *EnterpriseMonitor) formatAlertForDataDog(alert *monitoring.Alert) string {
	event := map[string]interface{}{
		"title":            alert.Name,
		"text":             alert.Message,
		"alert_type":       string(alert.Severity),
		"source_type_name": "llm-verifier",
		"tags":             em.formatDataDogTags(),
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func (em *EnterpriseMonitor) formatAlertForNewRelic(alert *monitoring.Alert) string {
	event := map[string]interface{}{
		"eventType": "LLMAlert",
		"severity":  string(alert.Severity),
		"message":   alert.Message,
		"timestamp": time.Now().Unix() * 1000,
	}

	for k, v := range em.config.NewRelic.Labels {
		event[k] = v
	}

	jsonData, err := json.Marshal([]map[string]interface{}{event})
	if err != nil {
		return ""
	}
	return string(jsonData)
}

func (em *EnterpriseMonitor) formatAlertForELK(alert *monitoring.Alert) string {
	doc := map[string]interface{}{
		"@timestamp": time.Now().Format(time.RFC3339),
		"service":    "llm-verifier",
		"type":       "alert",
		"alert":      alert,
	}

	jsonData, _ := json.Marshal(doc)
	return string(jsonData)
}

// Helper methods
func (em *EnterpriseMonitor) formatDataDogTags() []string {
	tags := []string{
		fmt.Sprintf("service:%s", em.config.DataDog.ServiceName),
		fmt.Sprintf("env:%s", em.config.DataDog.Environment),
	}

	for k, v := range em.config.DataDog.Tags {
		tags = append(tags, fmt.Sprintf("%s:%s", k, v))
	}

	return tags
}

func (em *EnterpriseMonitor) webhookSupportsEventType(webhook WebhookConfig, eventType string) bool {
	if len(webhook.EventTypes) == 0 {
		return true // No filter means all events
	}

	for _, et := range webhook.EventTypes {
		if et == eventType {
			return true
		}
	}

	return false
}

// GetStatus returns the status of enterprise monitoring integrations
func (em *EnterpriseMonitor) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"enabled": em.config.Enabled,
		"integrations": map[string]bool{
			"splunk":   em.config.Splunk.Host != "",
			"datadog":  em.config.DataDog.APIKey != "",
			"newrelic": em.config.NewRelic.LicenseKey != "",
			"elk":      em.config.ELK.ElasticsearchURL != "",
			"webhooks": len(em.config.CustomWebhooks) > 0,
		},
		"batch_interval": em.config.BatchInterval.String(),
		"retry_attempts": em.config.RetryAttempts,
	}

	return status
}
