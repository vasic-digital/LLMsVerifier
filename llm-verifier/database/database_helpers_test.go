package database

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanNullableTime(t *testing.T) {
	t.Run("valid time", func(t *testing.T) {
		now := time.Now()
		nullTime := sql.NullTime{Time: now, Valid: true}
		result := scanNullableTime(nullTime)
		require.NotNil(t, result)
		assert.Equal(t, now, *result)
	})

	t.Run("null time", func(t *testing.T) {
		nullTime := sql.NullTime{Valid: false}
		result := scanNullableTime(nullTime)
		assert.Nil(t, result)
	})
}

func TestScanNullableString(t *testing.T) {
	t.Run("valid string", func(t *testing.T) {
		nullString := sql.NullString{String: "test", Valid: true}
		result := scanNullableString(nullString)
		require.NotNil(t, result)
		assert.Equal(t, "test", *result)
	})

	t.Run("null string", func(t *testing.T) {
		nullString := sql.NullString{Valid: false}
		result := scanNullableString(nullString)
		assert.Nil(t, result)
	})
}

func TestScanNullableInt64(t *testing.T) {
	t.Run("valid int64", func(t *testing.T) {
		nullInt64 := sql.NullInt64{Int64: 42, Valid: true}
		result := scanNullableInt64(nullInt64)
		require.NotNil(t, result)
		assert.Equal(t, int64(42), *result)
	})

	t.Run("null int64", func(t *testing.T) {
		nullInt64 := sql.NullInt64{Valid: false}
		result := scanNullableInt64(nullInt64)
		assert.Nil(t, result)
	})
}

func TestScanNullableBoolFromString(t *testing.T) {
	t.Run("true string", func(t *testing.T) {
		nullString := sql.NullString{String: "true", Valid: true}
		result := scanNullableBoolFromString(nullString)
		require.NotNil(t, result)
		assert.True(t, *result)
	})

	t.Run("1 string", func(t *testing.T) {
		nullString := sql.NullString{String: "1", Valid: true}
		result := scanNullableBoolFromString(nullString)
		require.NotNil(t, result)
		assert.True(t, *result)
	})

	t.Run("false string", func(t *testing.T) {
		nullString := sql.NullString{String: "false", Valid: true}
		result := scanNullableBoolFromString(nullString)
		require.NotNil(t, result)
		assert.False(t, *result)
	})

	t.Run("0 string", func(t *testing.T) {
		nullString := sql.NullString{String: "0", Valid: true}
		result := scanNullableBoolFromString(nullString)
		require.NotNil(t, result)
		assert.False(t, *result)
	})

	t.Run("empty string", func(t *testing.T) {
		nullString := sql.NullString{String: "", Valid: true}
		result := scanNullableBoolFromString(nullString)
		assert.Nil(t, result)
	})

	t.Run("other string", func(t *testing.T) {
		nullString := sql.NullString{String: "maybe", Valid: true}
		result := scanNullableBoolFromString(nullString)
		assert.Nil(t, result)
	})
}

func TestScanNullableTimeFromString(t *testing.T) {
	t.Run("RFC3339 timestamp", func(t *testing.T) {
		testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		nullString := sql.NullString{String: testTime.Format(time.RFC3339), Valid: true}
		result := scanNullableTimeFromString(nullString)
		require.NotNil(t, result)
		// Compare truncated to seconds
		assert.Equal(t, testTime.Unix(), result.Unix())
	})

	t.Run("Unix timestamp", func(t *testing.T) {
		nullString := sql.NullString{String: "1609459200", Valid: true} // 2021-01-01 00:00:00 UTC
		result := scanNullableTimeFromString(nullString)
		require.NotNil(t, result)
		assert.Equal(t, int64(1609459200), result.Unix())
	})

	t.Run("empty string", func(t *testing.T) {
		nullString := sql.NullString{String: "", Valid: true}
		result := scanNullableTimeFromString(nullString)
		assert.Nil(t, result)
	})

	t.Run("invalid string", func(t *testing.T) {
		nullString := sql.NullString{String: "not-a-timestamp", Valid: true}
		result := scanNullableTimeFromString(nullString)
		assert.Nil(t, result)
	})

	t.Run("null string", func(t *testing.T) {
		nullString := sql.NullString{Valid: false}
		result := scanNullableTimeFromString(nullString)
		assert.Nil(t, result)
	})
}

func TestScanNullableIntFromInt64(t *testing.T) {
	t.Run("valid int", func(t *testing.T) {
		nullInt64 := sql.NullInt64{Int64: 100, Valid: true}
		result := scanNullableIntFromInt64(nullInt64)
		require.NotNil(t, result)
		assert.Equal(t, 100, *result)
	})

	t.Run("null int", func(t *testing.T) {
		nullInt64 := sql.NullInt64{Valid: false}
		result := scanNullableIntFromInt64(nullInt64)
		assert.Nil(t, result)
	})
}

func TestScanJSONString(t *testing.T) {
	t.Run("valid JSON array", func(t *testing.T) {
		nullString := sql.NullString{String: `["go", "python", "javascript"]`, Valid: true}
		result := scanJSONString(nullString)
		assert.Equal(t, []string{"go", "python", "javascript"}, result)
	})

	t.Run("empty string", func(t *testing.T) {
		nullString := sql.NullString{String: "", Valid: true}
		result := scanJSONString(nullString)
		assert.Equal(t, []string{}, result)
	})

	t.Run("null string", func(t *testing.T) {
		nullString := sql.NullString{Valid: false}
		result := scanJSONString(nullString)
		assert.Equal(t, []string{}, result)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		nullString := sql.NullString{String: "not-valid-json", Valid: true}
		result := scanJSONString(nullString)
		assert.Equal(t, []string{}, result)
	})
}

func TestToNullTime(t *testing.T) {
	t.Run("non-nil time", func(t *testing.T) {
		now := time.Now()
		result := toNullTime(&now)
		assert.True(t, result.Valid)
		assert.Equal(t, now, result.Time)
	})

	t.Run("nil time", func(t *testing.T) {
		result := toNullTime(nil)
		assert.False(t, result.Valid)
	})
}

func TestToNullString(t *testing.T) {
	t.Run("non-nil string", func(t *testing.T) {
		s := "test"
		result := toNullString(&s)
		assert.True(t, result.Valid)
		assert.Equal(t, "test", result.String)
	})

	t.Run("nil string", func(t *testing.T) {
		result := toNullString(nil)
		assert.False(t, result.Valid)
	})
}

func TestToNullInt(t *testing.T) {
	t.Run("non-nil int", func(t *testing.T) {
		i := 42
		result := toNullInt(&i)
		assert.True(t, result.Valid)
		assert.Equal(t, int32(42), result.Int32)
	})

	t.Run("nil int", func(t *testing.T) {
		result := toNullInt(nil)
		assert.False(t, result.Valid)
	})
}

func TestToNullBool(t *testing.T) {
	t.Run("non-nil true", func(t *testing.T) {
		b := true
		result := toNullBool(&b)
		assert.True(t, result.Valid)
		assert.True(t, result.Bool)
	})

	t.Run("non-nil false", func(t *testing.T) {
		b := false
		result := toNullBool(&b)
		assert.True(t, result.Valid)
		assert.False(t, result.Bool)
	})

	t.Run("nil bool", func(t *testing.T) {
		result := toNullBool(nil)
		assert.False(t, result.Valid)
	})
}

func TestDefaultConnectionPoolConfig(t *testing.T) {
	config := DefaultConnectionPoolConfig()

	assert.Equal(t, 25, config.MaxOpenConns)
	assert.Equal(t, 5, config.MaxIdleConns)
	assert.Equal(t, 5*time.Minute, config.ConnMaxLifetime)
	assert.Equal(t, 2*time.Minute, config.ConnMaxIdleTime)
}

func TestDatabase_New(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Verify database is accessible
	err = db.Ping()
	assert.NoError(t, err)
}

func TestDatabase_Close(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)

	err = db.Close()
	assert.NoError(t, err)
}

func TestDatabase_Ping(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	assert.NoError(t, err)
}

func TestDatabase_Transaction(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	t.Run("successful transaction", func(t *testing.T) {
		err := db.Transaction(func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO providers (name, endpoint, api_key_encrypted, description, website, support_email, documentation_url) VALUES (?, ?, ?, ?, ?, ?, ?)",
				"Test Provider", "https://test.com", "", "", "", "", "")
			return err
		})
		assert.NoError(t, err)

		// Verify provider was inserted by counting
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM providers WHERE name = ?", "Test Provider").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("failed transaction rolls back", func(t *testing.T) {
		err := db.Transaction(func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO providers (name, endpoint, api_key_encrypted, description, website, support_email, documentation_url) VALUES (?, ?, ?, ?, ?, ?, ?)",
				"Rollback Test", "https://rollback.com", "", "", "", "", "")
			if err != nil {
				return err
			}
			// Force error to trigger rollback
			_, err = tx.Exec("INVALID SQL QUERY")
			return err
		})
		assert.Error(t, err)

		// Verify provider was NOT inserted
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM providers WHERE name = ?", "Rollback Test").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestDatabase_WithTransaction(t *testing.T) {
	db, err := New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	t.Run("successful transaction", func(t *testing.T) {
		err := db.WithTransaction(func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO providers (name, endpoint, api_key_encrypted, description, website, support_email, documentation_url) VALUES (?, ?, ?, ?, ?, ?, ?)",
				"WithTx Provider", "https://withtx.com", "", "", "", "", "")
			return err
		})
		assert.NoError(t, err)

		// Verify provider was inserted
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM providers WHERE name = ?", "WithTx Provider").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("failed transaction rolls back", func(t *testing.T) {
		err := db.WithTransaction(func(tx *sql.Tx) error {
			_, err := tx.Exec("INSERT INTO providers (name, endpoint, api_key_encrypted, description, website, support_email, documentation_url) VALUES (?, ?, ?, ?, ?, ?, ?)",
				"WithTx Rollback", "https://rollback.com", "", "", "", "", "")
			if err != nil {
				return err
			}
			// Force error
			_, err = tx.Exec("INVALID SQL")
			return err
		})
		assert.Error(t, err)

		// Verify it was rolled back
		var count int
		err = db.conn.QueryRow("SELECT COUNT(*) FROM providers WHERE name = ?", "WithTx Rollback").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestProvider_JSON(t *testing.T) {
	provider := &Provider{
		ID:                    1,
		Name:                  "Test Provider",
		Endpoint:              "https://api.test.com",
		Description:           "A test provider",
		Website:               "https://test.com",
		IsActive:              true,
		ReliabilityScore:      0.95,
		AverageResponseTimeMs: 150,
	}

	data, err := json.Marshal(provider)
	require.NoError(t, err)

	var decoded Provider
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, provider.Name, decoded.Name)
	assert.Equal(t, provider.Endpoint, decoded.Endpoint)
	assert.Equal(t, provider.ReliabilityScore, decoded.ReliabilityScore)
}

func TestModel_JSON(t *testing.T) {
	model := &Model{
		ID:                 1,
		ProviderID:         1,
		ModelID:            "gpt-4",
		Name:               "GPT-4",
		Description:        "Advanced language model",
		IsMultimodal:       true,
		SupportsVision:     true,
		VerificationStatus: "verified",
		OverallScore:       92.5,
		Tags:               []string{"language", "multimodal"},
		LanguageSupport:    []string{"en", "es", "fr"},
	}

	data, err := json.Marshal(model)
	require.NoError(t, err)

	var decoded Model
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, model.Name, decoded.Name)
	assert.Equal(t, model.VerificationStatus, decoded.VerificationStatus)
	assert.Equal(t, model.Tags, decoded.Tags)
}

func TestVerificationResult_JSON(t *testing.T) {
	result := &VerificationResult{
		ID:                      1,
		ModelID:                 1,
		VerificationType:        "full",
		Status:                  "completed",
		SupportsCodeGeneration:  true,
		SupportsStreaming:       true,
		OverallScore:            88.0,
		CodeLanguageSupport:     []string{"go", "python"},
		AvgLatencyMs:            250,
		P95LatencyMs:            500,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded VerificationResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.VerificationType, decoded.VerificationType)
	assert.Equal(t, result.Status, decoded.Status)
	assert.Equal(t, result.OverallScore, decoded.OverallScore)
}

func TestEvent_JSON(t *testing.T) {
	modelID := int64(1)
	event := &Event{
		ID:        1,
		EventType: "verification_completed",
		Severity:  "info",
		Title:     "Verification Complete",
		Message:   "Model verification finished successfully",
		ModelID:   &modelID,
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	var decoded Event
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, event.EventType, decoded.EventType)
	assert.Equal(t, event.Title, decoded.Title)
}

func TestSchedule_JSON(t *testing.T) {
	cron := "0 */6 * * *"
	schedule := &Schedule{
		ID:             1,
		Name:           "Daily Verification",
		ScheduleType:   "cron",
		CronExpression: &cron,
		TargetType:     "all_models",
		IsActive:       true,
		RunCount:       10,
	}

	data, err := json.Marshal(schedule)
	require.NoError(t, err)

	var decoded Schedule
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, schedule.Name, decoded.Name)
	assert.Equal(t, schedule.ScheduleType, decoded.ScheduleType)
}

func TestConfigExport_JSON(t *testing.T) {
	export := &ConfigExport{
		ID:            1,
		ExportType:    "json",
		Name:          "Full Export",
		Description:   "Complete system export",
		ConfigData:    `{"key": "value"}`,
		IsVerified:    true,
		DownloadCount: 25,
	}

	data, err := json.Marshal(export)
	require.NoError(t, err)

	var decoded ConfigExport
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, export.Name, decoded.Name)
	assert.Equal(t, export.DownloadCount, decoded.DownloadCount)
}

func TestPricing_JSON(t *testing.T) {
	pricing := &Pricing{
		ID:              1,
		ModelID:         1,
		InputTokenCost:  0.001,
		OutputTokenCost: 0.002,
		Currency:        "USD",
		PricingModel:    "per_token",
	}

	data, err := json.Marshal(pricing)
	require.NoError(t, err)

	var decoded Pricing
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, pricing.InputTokenCost, decoded.InputTokenCost)
	assert.Equal(t, pricing.Currency, decoded.Currency)
}

func TestLimit_JSON(t *testing.T) {
	limit := &Limit{
		ID:           1,
		ModelID:      1,
		LimitType:    "requests_per_minute",
		LimitValue:   60,
		CurrentUsage: 10,
		ResetPeriod:  "minute",
		IsHardLimit:  true,
	}

	data, err := json.Marshal(limit)
	require.NoError(t, err)

	var decoded Limit
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, limit.LimitType, decoded.LimitType)
	assert.Equal(t, limit.LimitValue, decoded.LimitValue)
}

func TestIssue_JSON(t *testing.T) {
	issue := &Issue{
		ID:               1,
		ModelID:          1,
		IssueType:        "performance",
		Severity:         "high",
		Title:            "Slow Response Time",
		Description:      "Model responses are slower than expected",
		AffectedFeatures: []string{"code_generation", "completions"},
	}

	data, err := json.Marshal(issue)
	require.NoError(t, err)

	var decoded Issue
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, issue.IssueType, decoded.IssueType)
	assert.Equal(t, issue.Severity, decoded.Severity)
	assert.Equal(t, issue.AffectedFeatures, decoded.AffectedFeatures)
}

func TestNotification_JSON(t *testing.T) {
	notification := &Notification{
		ID:         1,
		Type:       "alert",
		Channel:    "email",
		Priority:   "high",
		Title:      "Critical Issue",
		Message:    "A critical issue has been detected",
		Recipient:  "admin@example.com",
		Sent:       true,
		RetryCount: 0,
		Data:       map[string]any{"issue_id": 1},
	}

	data, err := json.Marshal(notification)
	require.NoError(t, err)

	var decoded Notification
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, notification.Type, decoded.Type)
	assert.Equal(t, notification.Channel, decoded.Channel)
}

func TestUser_JSON(t *testing.T) {
	user := &User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		FullName: "Test User",
		Role:     "admin",
		IsActive: true,
		Preferences: map[string]any{
			"theme": "dark",
		},
	}

	data, err := json.Marshal(user)
	require.NoError(t, err)

	var decoded User
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, user.Username, decoded.Username)
	assert.Equal(t, user.Role, decoded.Role)
	// Password hash should not be in JSON
	assert.Empty(t, decoded.PasswordHash)
}

func TestAPIKey_JSON(t *testing.T) {
	apiKey := &APIKey{
		ID:       1,
		UserID:   1,
		Name:     "Production Key",
		Scopes:   []string{"read", "write"},
		IsActive: true,
	}

	data, err := json.Marshal(apiKey)
	require.NoError(t, err)

	var decoded APIKey
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, apiKey.Name, decoded.Name)
	assert.Equal(t, apiKey.Scopes, decoded.Scopes)
	// KeyHash should not be in JSON
	assert.Empty(t, decoded.KeyHash)
}
