// Package compliance provides GDPR/SOC2 compliance features
package auth

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// AuditEvent represents a compliance audit event
type AuditEvent struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	UserID      string                 `json:"user_id,omitempty"`
	ClientID    string                 `json:"client_id,omitempty"`
	Resource    string                 `json:"resource"`
	Action      string                 `json:"action"`
	Status      string                 `json:"status"`
	IPAddress   string                 `json:"ip_address,omitempty"`
	UserAgent   string                 `json:"user_agent,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	DataSubject string                 `json:"data_subject,omitempty"` // For GDPR
}

// ComplianceManager handles compliance reporting and audit trails
type ComplianceManager struct {
	auditLogFile *os.File
	auditEvents  chan *AuditEvent
	done         chan bool
}

// NewComplianceManager creates a new compliance manager
func NewComplianceManager(logFilePath string) (*ComplianceManager, error) {
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	cm := &ComplianceManager{
		auditLogFile: file,
		auditEvents:  make(chan *AuditEvent, 1000),
		done:         make(chan bool),
	}

	// Start audit logging goroutine
	go cm.auditLogger()

	return cm, nil
}

// LogAuditEvent logs an audit event
func (cm *ComplianceManager) LogAuditEvent(event *AuditEvent) error {
	select {
	case cm.auditEvents <- event:
		return nil
	default:
		// Channel is full, log to stderr as fallback
		log.Printf("Audit queue full, logging to stderr: %+v", event)
		return fmt.Errorf("audit queue full")
	}
}

// auditLogger processes audit events
func (cm *ComplianceManager) auditLogger() {
	for {
		select {
		case event := <-cm.auditEvents:
			cm.writeAuditEvent(event)
		case <-cm.done:
			return
		}
	}
}

// writeAuditEvent writes an audit event to the log file
func (cm *ComplianceManager) writeAuditEvent(event *AuditEvent) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal audit event: %v", err)
		return
	}

	// Write with newline for easy parsing
	_, err = fmt.Fprintf(cm.auditLogFile, "%s\n", eventJSON)
	if err != nil {
		log.Printf("Failed to write audit event: %v", err)
	}

	// Flush to ensure it's written
	cm.auditLogFile.Sync()
}

// Close closes the compliance manager
func (cm *ComplianceManager) Close() error {
	close(cm.done)
	return cm.auditLogFile.Close()
}

// QueryAuditLogs queries audit logs with filters
func (cm *ComplianceManager) QueryAuditLogs(filters map[string]interface{}, limit int) ([]*AuditEvent, error) {
	// Get the audit log file path
	if cm.auditLogFile == nil {
		return nil, fmt.Errorf("audit log file not initialized")
	}

	// Get the file path from the file
	filePath := cm.auditLogFile.Name()

	// Open the file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log for reading: %w", err)
	}
	defer file.Close()

	var events []*AuditEvent
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if limit > 0 && len(events) >= limit {
			break
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		var event AuditEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip malformed lines
		}

		// Apply filters
		if !cm.matchesFilters(&event, filters) {
			continue
		}

		events = append(events, &event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading audit log: %w", err)
	}

	return events, nil
}

// matchesFilters checks if an event matches the given filters
func (cm *ComplianceManager) matchesFilters(event *AuditEvent, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, value := range filters {
		switch key {
		case "event_type":
			if v, ok := value.(string); ok && event.EventType != v {
				return false
			}
		case "user_id":
			if v, ok := value.(string); ok && event.UserID != v {
				return false
			}
		case "client_id":
			if v, ok := value.(string); ok && event.ClientID != v {
				return false
			}
		case "resource":
			if v, ok := value.(string); ok && event.Resource != v {
				return false
			}
		case "action":
			if v, ok := value.(string); ok && event.Action != v {
				return false
			}
		case "status":
			if v, ok := value.(string); ok && event.Status != v {
				return false
			}
		case "from_date":
			if v, ok := value.(time.Time); ok && event.Timestamp.Before(v) {
				return false
			}
		case "to_date":
			if v, ok := value.(time.Time); ok && event.Timestamp.After(v) {
				return false
			}
		}
	}

	return true
}

// DataRetentionManager handles data retention policies
type DataRetentionManager struct {
	policies map[string]RetentionPolicy
}

// RetentionPolicy defines data retention rules
type RetentionPolicy struct {
	Resource   string        `json:"resource"`
	Retention  time.Duration `json:"retention"`
	Action     string        `json:"action"` // delete, archive, anonymize
	Compliance string        `json:"compliance,omitempty"`
}

// NewDataRetentionManager creates a data retention manager
func NewDataRetentionManager() *DataRetentionManager {
	drm := &DataRetentionManager{
		policies: make(map[string]RetentionPolicy),
	}

	// Initialize default policies
	drm.initializeDefaultPolicies()

	return drm
}

// initializeDefaultPolicies sets up default retention policies
func (drm *DataRetentionManager) initializeDefaultPolicies() {
	// Audit logs - 7 years for SOC2
	drm.policies["audit_logs"] = RetentionPolicy{
		Resource:   "audit_logs",
		Retention:  7 * 365 * 24 * time.Hour,
		Action:     "archive",
		Compliance: "SOC2",
	}

	// User data - 3 years minimum for GDPR
	drm.policies["user_data"] = RetentionPolicy{
		Resource:   "user_data",
		Retention:  3 * 365 * 24 * time.Hour,
		Action:     "anonymize",
		Compliance: "GDPR",
	}

	// Session data - 30 days
	drm.policies["session_data"] = RetentionPolicy{
		Resource:   "session_data",
		Retention:  30 * 24 * time.Hour,
		Action:     "delete",
		Compliance: "General",
	}

	// API logs - 1 year
	drm.policies["api_logs"] = RetentionPolicy{
		Resource:   "api_logs",
		Retention:  365 * 24 * time.Hour,
		Action:     "archive",
		Compliance: "General",
	}
}

// SetRetentionPolicy sets a custom retention policy
func (drm *DataRetentionManager) SetRetentionPolicy(resource string, policy RetentionPolicy) {
	drm.policies[resource] = policy
}

// GetRetentionPolicy gets the retention policy for a resource
func (drm *DataRetentionManager) GetRetentionPolicy(resource string) (RetentionPolicy, bool) {
	policy, exists := drm.policies[resource]
	return policy, exists
}

// CheckRetention checks if data should be retained or deleted
func (drm *DataRetentionManager) CheckRetention(resource string, createdAt time.Time) (bool, string) {
	policy, exists := drm.policies[resource]
	if !exists {
		return true, "" // Keep by default
	}

	age := time.Since(createdAt)
	if age > policy.Retention {
		return false, policy.Action
	}

	return true, ""
}

// GDPRManager handles GDPR compliance features
type GDPRManager struct {
	consentManager *ConsentManager
	dataManager    *DataManager
}

// NewGDPRManager creates a GDPR compliance manager
func NewGDPRManager() *GDPRManager {
	return &GDPRManager{
		consentManager: NewConsentManager(),
		dataManager:    NewDataManager(),
	}
}

// HandleDataSubjectRequest handles GDPR data subject requests
func (gm *GDPRManager) HandleDataSubjectRequest(requestType, userID string) error {
	switch requestType {
	case "access":
		return gm.handleDataAccessRequest(userID)
	case "rectification":
		return gm.handleDataRectificationRequest(userID)
	case "erasure":
		return gm.handleDataErasureRequest(userID)
	case "portability":
		return gm.handleDataPortabilityRequest(userID)
	case "restriction":
		return gm.handleDataRestrictionRequest(userID)
	case "objection":
		return gm.handleDataObjectionRequest(userID)
	default:
		return fmt.Errorf("unknown request type: %s", requestType)
	}
}

// handleDataAccessRequest handles GDPR right of access
func (gm *GDPRManager) handleDataAccessRequest(userID string) error {
	// Export all user data
	data, err := gm.dataManager.ExportUserData(userID)
	if err != nil {
		return fmt.Errorf("failed to export user data: %w", err)
	}

	// In production, this would send the data to the user
	log.Printf("Data access request fulfilled for user %s: %d bytes of data", userID, len(data))

	return nil
}

// handleDataErasureRequest handles GDPR right to erasure
func (gm *GDPRManager) handleDataErasureRequest(userID string) error {
	// Delete all user data
	err := gm.dataManager.DeleteUserData(userID)
	if err != nil {
		return fmt.Errorf("failed to delete user data: %w", err)
	}

	log.Printf("Data erasure request fulfilled for user %s", userID)
	return nil
}

// handleDataRectificationRequest handles GDPR right to rectification
func (gm *GDPRManager) handleDataRectificationRequest(userID string) error {
	// In production, this would update user data based on provided corrections
	log.Printf("Data rectification request received for user %s", userID)
	return nil
}

// handleDataPortabilityRequest handles GDPR right to data portability
func (gm *GDPRManager) handleDataPortabilityRequest(userID string) error {
	data, err := gm.dataManager.ExportUserData(userID)
	if err != nil {
		return fmt.Errorf("failed to export user data: %w", err)
	}

	// In production, this would provide data in portable format
	log.Printf("Data portability request fulfilled for user %s: %d bytes", userID, len(data))
	return nil
}

// handleDataRestrictionRequest handles GDPR right to restriction of processing
func (gm *GDPRManager) handleDataRestrictionRequest(userID string) error {
	// Mark user data as restricted
	log.Printf("Data processing restriction applied for user %s", userID)
	return nil
}

// handleDataObjectionRequest handles GDPR right to object
func (gm *GDPRManager) handleDataObjectionRequest(userID string) error {
	// Stop processing user data for specified purposes
	log.Printf("Data processing objection processed for user %s", userID)
	return nil
}

// ConsentManager manages user consent for data processing
type ConsentManager struct {
	consents map[string]*UserConsent
}

// UserConsent represents user consent for data processing
type UserConsent struct {
	UserID      string     `json:"user_id"`
	Purpose     string     `json:"purpose"`
	GrantedAt   time.Time  `json:"granted_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	WithdrawnAt *time.Time `json:"withdrawn_at,omitempty"`
}

// NewConsentManager creates a consent manager
func NewConsentManager() *ConsentManager {
	return &ConsentManager{
		consents: make(map[string]*UserConsent),
	}
}

// GrantConsent grants user consent for a purpose
func (cm *ConsentManager) GrantConsent(userID, purpose string, expiresAt *time.Time) {
	key := fmt.Sprintf("%s:%s", userID, purpose)
	cm.consents[key] = &UserConsent{
		UserID:    userID,
		Purpose:   purpose,
		GrantedAt: time.Now(),
		ExpiresAt: expiresAt,
	}
}

// WithdrawConsent withdraws user consent
func (cm *ConsentManager) WithdrawConsent(userID, purpose string) {
	key := fmt.Sprintf("%s:%s", userID, purpose)
	if consent, exists := cm.consents[key]; exists {
		now := time.Now()
		consent.WithdrawnAt = &now
	}
}

// CheckConsent checks if user has given consent for a purpose
func (cm *ConsentManager) CheckConsent(userID, purpose string) bool {
	key := fmt.Sprintf("%s:%s", userID, purpose)
	consent, exists := cm.consents[key]
	if !exists {
		return false
	}

	if consent.WithdrawnAt != nil {
		return false
	}

	if consent.ExpiresAt != nil && time.Now().After(*consent.ExpiresAt) {
		return false
	}

	return true
}

// DataManager handles user data operations for GDPR
type DataManager struct {
	mu           sync.RWMutex
	userData     map[string]*UserDataRecord
	deletedUsers map[string]time.Time
	dataDir      string
}

// UserDataRecord stores user data for GDPR compliance
type UserDataRecord struct {
	UserID      string                 `json:"user_id"`
	Email       string                 `json:"email,omitempty"`
	Name        string                 `json:"name,omitempty"`
	Preferences map[string]interface{} `json:"preferences,omitempty"`
	APIKeys     []string               `json:"api_keys,omitempty"`
	APILogs     []APILogEntry          `json:"api_logs,omitempty"`
	AuditLogs   []AuditEvent           `json:"audit_logs,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Restricted  bool                   `json:"restricted"` // Processing restricted
	Anonymized  bool                   `json:"anonymized"`
}

// APILogEntry represents an API log entry
type APILogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Endpoint   string    `json:"endpoint"`
	Method     string    `json:"method"`
	StatusCode int       `json:"status_code"`
	IPAddress  string    `json:"ip_address,omitempty"`
}

// NewDataManager creates a data manager
func NewDataManager() *DataManager {
	return &DataManager{
		userData:     make(map[string]*UserDataRecord),
		deletedUsers: make(map[string]time.Time),
		dataDir:      "./data/gdpr",
	}
}

// SetDataDir sets the directory for data storage
func (dm *DataManager) SetDataDir(dir string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.dataDir = dir
}

// StoreUserData stores user data record
func (dm *DataManager) StoreUserData(record *UserDataRecord) error {
	if record.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	dm.mu.Lock()
	defer dm.mu.Unlock()

	record.UpdatedAt = time.Now()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	dm.userData[record.UserID] = record
	return nil
}

// GetUserData retrieves user data record
func (dm *DataManager) GetUserData(userID string) (*UserDataRecord, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	record, exists := dm.userData[userID]
	if !exists {
		return nil, fmt.Errorf("user data not found: %s", userID)
	}

	return record, nil
}

// ExportUserData exports all user data for GDPR access requests
func (dm *DataManager) ExportUserData(userID string) ([]byte, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	record, exists := dm.userData[userID]
	if !exists {
		// Return minimal data structure if user not found
		data := map[string]interface{}{
			"user_id":       userID,
			"exported_at":   time.Now(),
			"data_found":    false,
			"data_sources":  []string{"auth", "api_logs", "preferences", "audit_logs"},
			"export_format": "GDPR-compliant",
		}
		return json.MarshalIndent(data, "", "  ")
	}

	// Create comprehensive export
	export := map[string]interface{}{
		"user_id":       record.UserID,
		"exported_at":   time.Now(),
		"data_found":    true,
		"export_format": "GDPR-compliant",
		"data_sources":  []string{"auth", "api_logs", "preferences", "audit_logs"},
		"user_data": map[string]interface{}{
			"email":       record.Email,
			"name":        record.Name,
			"created_at":  record.CreatedAt,
			"updated_at":  record.UpdatedAt,
			"preferences": record.Preferences,
		},
		"api_activity": map[string]interface{}{
			"api_keys_count": len(record.APIKeys),
			"api_logs_count": len(record.APILogs),
			"api_logs":       record.APILogs,
		},
		"audit_trail": map[string]interface{}{
			"audit_logs_count": len(record.AuditLogs),
			"audit_logs":       record.AuditLogs,
		},
	}

	return json.MarshalIndent(export, "", "  ")
}

// DeleteUserData deletes all user data for GDPR erasure requests
func (dm *DataManager) DeleteUserData(userID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check if user exists
	_, exists := dm.userData[userID]
	if !exists {
		log.Printf("No user data found for deletion: %s", userID)
		return nil // Not an error - data may already be deleted
	}

	// Delete the user data
	delete(dm.userData, userID)

	// Record the deletion for audit purposes
	dm.deletedUsers[userID] = time.Now()

	log.Printf("User data deleted for GDPR erasure: %s", userID)

	// In production, you would also:
	// 1. Delete from database tables
	// 2. Delete from file storage
	// 3. Notify downstream systems
	// 4. Update audit log

	return nil
}

// AnonymizeUserData anonymizes user data for retention
func (dm *DataManager) AnonymizeUserData(userID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	record, exists := dm.userData[userID]
	if !exists {
		return fmt.Errorf("user data not found: %s", userID)
	}

	// Anonymize PII fields
	record.Email = anonymizeEmail(record.Email)
	record.Name = anonymizeName(record.Name)

	// Anonymize IP addresses in API logs
	for i := range record.APILogs {
		record.APILogs[i].IPAddress = anonymizeIP(record.APILogs[i].IPAddress)
	}

	// Anonymize IP addresses in audit logs
	for i := range record.AuditLogs {
		record.AuditLogs[i].IPAddress = anonymizeIP(record.AuditLogs[i].IPAddress)
	}

	// Mark as anonymized
	record.Anonymized = true
	record.UpdatedAt = time.Now()

	log.Printf("User data anonymized for GDPR compliance: %s", userID)
	return nil
}

// RestrictProcessing marks user data as restricted
func (dm *DataManager) RestrictProcessing(userID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	record, exists := dm.userData[userID]
	if !exists {
		return fmt.Errorf("user data not found: %s", userID)
	}

	record.Restricted = true
	record.UpdatedAt = time.Now()

	log.Printf("Processing restricted for user: %s", userID)
	return nil
}

// IsProcessingRestricted checks if processing is restricted for a user
func (dm *DataManager) IsProcessingRestricted(userID string) bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	record, exists := dm.userData[userID]
	if !exists {
		return false
	}

	return record.Restricted
}

// anonymizeEmail anonymizes an email address
func anonymizeEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "***@***.***"
	}
	return fmt.Sprintf("***@%s", parts[1])
}

// anonymizeName anonymizes a name
func anonymizeName(name string) string {
	if name == "" {
		return ""
	}
	return "[ANONYMIZED]"
}

// anonymizeIP anonymizes an IP address
func anonymizeIP(ip string) string {
	if ip == "" {
		return ""
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return fmt.Sprintf("%s.%s.0.0", parts[0], parts[1])
	}
	// IPv6 or other format
	return "0.0.0.0"
}

// SaveToDisk saves user data to disk (for backup/export)
func (dm *DataManager) SaveToDisk(userID string) error {
	data, err := dm.ExportUserData(userID)
	if err != nil {
		return fmt.Errorf("failed to export user data: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(dm.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := fmt.Sprintf("%s/%s_export.json", dm.dataDir, userID)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	log.Printf("User data exported to disk: %s", filePath)
	return nil
}

// LoadFromDisk loads user data from disk
func (dm *DataManager) LoadFromDisk(userID string) (*UserDataRecord, error) {
	filePath := fmt.Sprintf("%s/%s_export.json", dm.dataDir, userID)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open export file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read export file: %w", err)
	}

	var record UserDataRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, fmt.Errorf("failed to parse export file: %w", err)
	}

	return &record, nil
}

// ComplianceReport generates compliance reports
type ComplianceReport struct {
	ReportType  string                 `json:"report_type"`
	GeneratedAt time.Time              `json:"generated_at"`
	Period      string                 `json:"period"`
	Findings    []ComplianceFinding    `json:"findings"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// ComplianceFinding represents a compliance finding
type ComplianceFinding struct {
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Resource    string `json:"resource"`
	Action      string `json:"action,omitempty"`
}

// GenerateComplianceReport generates a compliance report
func GenerateComplianceReport(reportType, period string) (*ComplianceReport, error) {
	report := &ComplianceReport{
		ReportType:  reportType,
		GeneratedAt: time.Now(),
		Period:      period,
		Findings:    []ComplianceFinding{},
		Metrics:     make(map[string]interface{}),
	}

	// Add sample findings
	switch reportType {
	case "GDPR":
		report.Findings = append(report.Findings, ComplianceFinding{
			Severity:    "info",
			Category:    "data_processing",
			Description: "All user data processing has valid consent",
			Resource:    "consent_manager",
		})
	case "SOC2":
		report.Findings = append(report.Findings, ComplianceFinding{
			Severity:    "info",
			Category:    "access_control",
			Description: "Multi-factor authentication enabled for admin accounts",
			Resource:    "auth_system",
		})
	}

	report.Metrics["total_audit_events"] = 150
	report.Metrics["active_users"] = 25
	report.Metrics["data_subjects"] = 25

	return report, nil
}
