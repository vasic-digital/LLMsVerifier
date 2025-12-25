package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ComplianceChecker provides compliance checking capabilities
type ComplianceChecker struct {
	gdprEnabled       bool
	dataRetentionDays int
	piiPatterns       []*regexp.Regexp
	sensitiveFields   map[string]bool
}

// NewComplianceChecker creates a new compliance checker
func NewComplianceChecker(gdprEnabled bool, dataRetentionDays int) *ComplianceChecker {
	cc := &ComplianceChecker{
		gdprEnabled:       gdprEnabled,
		dataRetentionDays: dataRetentionDays,
		piiPatterns:       []*regexp.Regexp{},
		sensitiveFields:   make(map[string]bool),
	}

	cc.initializePatterns()
	return cc
}

// initializePatterns sets up PII detection patterns
func (cc *ComplianceChecker) initializePatterns() {
	// PII patterns
	patterns := []string{
		// Email addresses
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
		// Phone numbers (various formats)
		`\+?\d{1,3}?[-.\s]?\(?(\d{3})\)?[-.\s]?(\d{3})[-.\s]?(\d{4})`,
		// Social Security Numbers (US)
		`\b\d{3}-\d{2}-\d{4}\b`,
		// Credit card numbers (basic pattern)
		`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`,
		// IP addresses
		`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`,
		// API keys (basic patterns)
		`(?i)\b(sk|pk|xoxp|xoxb|ghp)_[a-zA-Z0-9_]{20,}\b`,
	}

	for _, pattern := range patterns {
		if re, err := regexp.Compile(pattern); err == nil {
			cc.piiPatterns = append(cc.piiPatterns, re)
		}
	}

	// Sensitive fields
	sensitiveFields := []string{
		"password", "api_key", "secret", "token", "auth_token",
		"email", "phone", "ssn", "social_security", "credit_card",
		"address", "ip_address", "location", "personal_data",
	}

	for _, field := range sensitiveFields {
		cc.sensitiveFields[strings.ToLower(field)] = true
	}
}

// CheckDataCompliance checks data for compliance violations
func (cc *ComplianceChecker) CheckDataCompliance(data map[string]interface{}) *ComplianceResult {
	result := &ComplianceResult{
		IsCompliant:    true,
		Violations:     []ComplianceViolation{},
		PIIDetected:    false,
		RetentionCheck: true,
	}

	// Check for PII in data
	result.PIIDetected = cc.detectPII(data)

	// Check for sensitive fields
	cc.checkSensitiveFields(data, result)

	// Check data retention requirements
	cc.checkDataRetention(data, result)

	// GDPR compliance check
	if cc.gdprEnabled {
		cc.checkGDPRCompliance(data, result)
	}

	// Overall compliance
	result.IsCompliant = len(result.Violations) == 0

	return result
}

// detectPII detects personally identifiable information
func (cc *ComplianceChecker) detectPII(data map[string]interface{}) bool {
	// Convert data to string for pattern matching
	dataStr := cc.dataToString(data)

	for _, pattern := range cc.piiPatterns {
		if pattern.MatchString(dataStr) {
			return true
		}
	}

	return false
}

// checkSensitiveFields checks for sensitive field usage
func (cc *ComplianceChecker) checkSensitiveFields(data map[string]interface{}, result *ComplianceResult) {
	for key := range data {
		if cc.sensitiveFields[strings.ToLower(key)] {
			result.Violations = append(result.Violations, ComplianceViolation{
				Type:        "sensitive_field",
				Description: fmt.Sprintf("Sensitive field detected: %s", key),
				Severity:    "high",
				Remediation: "Ensure proper encryption and access controls",
			})
		}
	}
}

// checkDataRetention checks data retention compliance
func (cc *ComplianceChecker) checkDataRetention(data map[string]interface{}, result *ComplianceResult) {
	// Check for timestamps that might indicate old data
	if createdAt, ok := data["created_at"]; ok {
		if cc.isDataExpired(createdAt) {
			result.Violations = append(result.Violations, ComplianceViolation{
				Type:        "retention_violation",
				Description: "Data exceeds retention period",
				Severity:    "medium",
				Remediation: "Delete or archive expired data",
			})
			result.RetentionCheck = false
		}
	}
}

// checkGDPRCompliance checks GDPR compliance requirements
func (cc *ComplianceChecker) checkGDPRCompliance(data map[string]interface{}, result *ComplianceResult) {
	// GDPR requires explicit consent for data processing
	if _, hasConsent := data["gdpr_consent"]; !hasConsent {
		if result.PIIDetected {
			result.Violations = append(result.Violations, ComplianceViolation{
				Type:        "gdpr_violation",
				Description: "PII data processed without GDPR consent",
				Severity:    "critical",
				Remediation: "Obtain explicit user consent before processing personal data",
			})
		}
	}

	// Check for data minimization
	fieldCount := len(data)
	if fieldCount > 20 { // Arbitrary threshold
		result.Violations = append(result.Violations, ComplianceViolation{
			Type:        "gdpr_violation",
			Description: "Excessive data collection (data minimization principle)",
			Severity:    "medium",
			Remediation: "Collect only necessary data fields",
		})
	}
}

// isDataExpired checks if data has exceeded retention period
func (cc *ComplianceChecker) isDataExpired(timestamp interface{}) bool {
	var createdTime time.Time

	switch t := timestamp.(type) {
	case time.Time:
		createdTime = t
	case string:
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			createdTime = parsed
		} else {
			return false // Can't determine age
		}
	default:
		return false // Unknown format
	}

	retentionPeriod := time.Duration(cc.dataRetentionDays) * 24 * time.Hour
	return time.Since(createdTime) > retentionPeriod
}

// dataToString converts data map to string for pattern matching
func (cc *ComplianceChecker) dataToString(data map[string]interface{}) string {
	var parts []string
	for key, value := range data {
		parts = append(parts, fmt.Sprintf("%s=%v", key, value))
	}
	return strings.Join(parts, " ")
}

// CheckRequestCompliance checks HTTP request compliance
func (cc *ComplianceChecker) CheckRequestCompliance(r *http.Request) *ComplianceResult {
	result := &ComplianceResult{
		IsCompliant:    true,
		Violations:     []ComplianceViolation{},
		PIIDetected:    false,
		RetentionCheck: true,
	}

	// Check query parameters for PII
	if r.URL.RawQuery != "" {
		queryData := map[string]interface{}{"query": r.URL.RawQuery}
		if cc.detectPII(queryData) {
			result.PIIDetected = true
			result.Violations = append(result.Violations, ComplianceViolation{
				Type:        "pii_in_query",
				Description: "PII detected in URL query parameters",
				Severity:    "high",
				Remediation: "Avoid passing PII in URL parameters",
			})
		}
	}

	// Check headers for sensitive information
	for header, values := range r.Header {
		if cc.sensitiveFields[strings.ToLower(header)] {
			result.Violations = append(result.Violations, ComplianceViolation{
				Type:        "sensitive_header",
				Description: fmt.Sprintf("Sensitive header detected: %s", header),
				Severity:    "medium",
				Remediation: "Avoid sending sensitive data in headers",
			})
		}

		// Check header values for PII
		for _, value := range values {
			headerData := map[string]interface{}{header: value}
			if cc.detectPII(headerData) {
				result.PIIDetected = true
				result.Violations = append(result.Violations, ComplianceViolation{
					Type:        "pii_in_header",
					Description: fmt.Sprintf("PII detected in header: %s", header),
					Severity:    "high",
					Remediation: "Avoid including PII in HTTP headers",
				})
			}
		}
	}

	result.IsCompliant = len(result.Violations) == 0
	return result
}

// ComplianceResult represents the result of compliance checking
type ComplianceResult struct {
	IsCompliant    bool                  `json:"is_compliant"`
	Violations     []ComplianceViolation `json:"violations,omitempty"`
	PIIDetected    bool                  `json:"pii_detected"`
	RetentionCheck bool                  `json:"retention_check"`
}

// ComplianceViolation represents a compliance violation
type ComplianceViolation struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation,omitempty"`
}

// DataRetentionPolicy defines data retention policies
type DataRetentionPolicy struct {
	DataType      string        `json:"data_type"`
	RetentionDays int           `json:"retention_days"`
	AutoDelete    bool          `json:"auto_delete"`
	ArchiveAfter  time.Duration `json:"archive_after,omitempty"`
}

// GetRetentionPolicy returns retention policy for a data type
func (cc *ComplianceChecker) GetRetentionPolicy(dataType string) *DataRetentionPolicy {
	// Default policies (would be configurable in production)
	policies := map[string]*DataRetentionPolicy{
		"user_data": {
			DataType:      "user_data",
			RetentionDays: 2555, // 7 years for GDPR
			AutoDelete:    false,
			ArchiveAfter:  365 * 24 * time.Hour, // Archive after 1 year
		},
		"logs": {
			DataType:      "logs",
			RetentionDays: 90,
			AutoDelete:    true,
		},
		"audit": {
			DataType:      "audit",
			RetentionDays: 2555, // 7 years
			AutoDelete:    false,
		},
		"temp": {
			DataType:      "temp",
			RetentionDays: 7,
			AutoDelete:    true,
		},
	}

	if policy, exists := policies[dataType]; exists {
		return policy
	}

	// Default policy
	return &DataRetentionPolicy{
		DataType:      dataType,
		RetentionDays: cc.dataRetentionDays,
		AutoDelete:    true,
	}
}
