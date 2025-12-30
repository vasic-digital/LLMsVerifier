package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Event CRUD Tests ====================

func TestCreateEvent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	details := `{"key": "value"}`
	event := &Event{
		EventType: "verification",
		Severity:  "info",
		Title:     "Test Event",
		Message:   "This is a test event",
		Details:   &details,
	}

	err := db.CreateEvent(event)
	require.NoError(t, err)
	assert.NotZero(t, event.ID)

	// Verify creation
	retrieved, err := db.GetEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Event", retrieved.Title)
	assert.Equal(t, "verification", retrieved.EventType)
	assert.Equal(t, "info", retrieved.Severity)
}

func TestCreateEvent_WithRelations(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create a provider first
	provider := &Provider{
		Name:     "Test Provider",
		Endpoint: "https://api.test.com/v1",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create a model
	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
		Deprecated: false,
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	// Create event with relations
	event := &Event{
		EventType:  "model_update",
		Severity:   "warning",
		Title:      "Model Updated",
		Message:    "Model configuration changed",
		ModelID:    &model.ID,
		ProviderID: &provider.ID,
	}

	err = db.CreateEvent(event)
	require.NoError(t, err)
	assert.NotZero(t, event.ID)
}

func TestCreateEvent_NilDetails(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	event := &Event{
		EventType: "system",
		Severity:  "info",
		Title:     "System Event",
		Message:   "System startup",
		Details:   nil,
	}

	err := db.CreateEvent(event)
	require.NoError(t, err)
	assert.NotZero(t, event.ID)
}

func TestGetEvent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create first
	event := &Event{
		EventType: "test",
		Severity:  "error",
		Title:     "Get Test Event",
		Message:   "For retrieval test",
	}
	err := db.CreateEvent(event)
	require.NoError(t, err)

	// Get by ID
	retrieved, err := db.GetEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, event.Title, retrieved.Title)
	assert.Equal(t, event.Message, retrieved.Message)
	assert.Equal(t, "error", retrieved.Severity)

	// Get non-existent
	_, err = db.GetEvent(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListEvents(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create multiple events
	for i := 0; i < 5; i++ {
		event := &Event{
			EventType: "list_test",
			Severity:  "info",
			Title:     "List Test Event",
			Message:   "For listing",
		}
		err := db.CreateEvent(event)
		require.NoError(t, err)
	}

	// List all
	events, err := db.ListEvents(nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 5)
}

func TestListEvents_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create events with different types and severities
	types := []string{"verification", "system", "verification"}
	severities := []string{"info", "warning", "error"}

	for i, eventType := range types {
		event := &Event{
			EventType: eventType,
			Severity:  severities[i],
			Title:     "Filter Test",
			Message:   "For filter test",
		}
		err := db.CreateEvent(event)
		require.NoError(t, err)
	}

	// Filter by type
	filters := map[string]any{"event_type": "verification"}
	events, err := db.ListEvents(filters)
	require.NoError(t, err)
	for _, e := range events {
		assert.Equal(t, "verification", e.EventType)
	}
}

func TestUpdateEvent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create
	event := &Event{
		EventType: "test",
		Severity:  "info",
		Title:     "Original Title",
		Message:   "Original Message",
	}
	err := db.CreateEvent(event)
	require.NoError(t, err)

	// Update
	newDetails := `{"updated": true}`
	event.Title = "Updated Title"
	event.Message = "Updated Message"
	event.Severity = "warning"
	event.Details = &newDetails

	err = db.UpdateEvent(event)
	require.NoError(t, err)

	// Verify update
	retrieved, err := db.GetEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", retrieved.Title)
	assert.Equal(t, "Updated Message", retrieved.Message)
	assert.Equal(t, "warning", retrieved.Severity)
}

func TestDeleteEvent(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create
	event := &Event{
		EventType: "delete_test",
		Severity:  "info",
		Title:     "Delete Test",
		Message:   "Will be deleted",
	}
	err := db.CreateEvent(event)
	require.NoError(t, err)

	// Delete
	err = db.DeleteEvent(event.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = db.GetEvent(event.ID)
	assert.Error(t, err)
}

func TestGetEventsByType(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create events with different types
	for _, eventType := range []string{"verification", "system", "verification", "alert"} {
		event := &Event{
			EventType: eventType,
			Severity:  "info",
			Title:     "Type Test",
			Message:   "For type test",
		}
		err := db.CreateEvent(event)
		require.NoError(t, err)
	}

	// Get by type
	events, err := db.GetEventsByType("verification", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(events), 2)
	for _, e := range events {
		assert.Equal(t, "verification", e.EventType)
	}
}

func TestGetRecentEvents(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create events
	for i := 0; i < 10; i++ {
		event := &Event{
			EventType: "recent_test",
			Severity:  "info",
			Title:     "Recent Test",
			Message:   "For recent test",
		}
		err := db.CreateEvent(event)
		require.NoError(t, err)
	}

	// Get recent with limit
	events, err := db.GetRecentEvents(5)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(events), 5)
}
