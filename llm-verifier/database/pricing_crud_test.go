package database

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPricingTestDB(t *testing.T) *Database {
	dbFile := "/tmp/test_pricing_" + time.Now().Format("20060102150405") + ".db"
	db, err := New(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbFile)
	})
	return db
}

func createTestPricingModel(t *testing.T, db *Database) *Model {
	provider := &Provider{
		Name:     "test-provider",
		Endpoint: "http://test.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)
	return model
}

func createTestPricing(modelID int64) *Pricing {
	return &Pricing{
		ModelID:              modelID,
		InputTokenCost:       0.01,
		OutputTokenCost:      0.03,
		CachedInputTokenCost: 0.005,
		StorageCost:          0.0001,
		RequestCost:          0.001,
		Currency:             "USD",
		PricingModel:         "per_token",
	}
}

// ==================== Pricing CRUD Tests ====================

func TestCreatePricing(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	pricing := createTestPricing(model.ID)
	err := db.CreatePricing(pricing)
	require.NoError(t, err)
	assert.NotZero(t, pricing.ID)
}

func TestCreatePricing_WithEffectiveDates(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	now := time.Now()
	effectiveTo := now.Add(30 * 24 * time.Hour)
	pricing := &Pricing{
		ModelID:         model.ID,
		InputTokenCost:  0.02,
		OutputTokenCost: 0.04,
		Currency:        "USD",
		PricingModel:    "per_token",
		EffectiveFrom:   &now,
		EffectiveTo:     &effectiveTo,
	}

	err := db.CreatePricing(pricing)
	require.NoError(t, err)
	assert.NotZero(t, pricing.ID)
}

func TestGetPricing(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	pricing := createTestPricing(model.ID)
	err := db.CreatePricing(pricing)
	require.NoError(t, err)

	retrieved, err := db.GetPricing(pricing.ID)
	require.NoError(t, err)
	assert.Equal(t, pricing.ID, retrieved.ID)
	assert.Equal(t, pricing.ModelID, retrieved.ModelID)
	assert.Equal(t, pricing.InputTokenCost, retrieved.InputTokenCost)
	assert.Equal(t, pricing.OutputTokenCost, retrieved.OutputTokenCost)
	assert.Equal(t, pricing.Currency, retrieved.Currency)
}

func TestGetPricing_NotFound(t *testing.T) {
	db := setupPricingTestDB(t)

	_, err := db.GetPricing(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetLatestPricing(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	// Create pricing without effective_to (current)
	pricing := createTestPricing(model.ID)
	err := db.CreatePricing(pricing)
	require.NoError(t, err)

	retrieved, err := db.GetLatestPricing(model.ID)
	require.NoError(t, err)
	assert.Equal(t, pricing.ID, retrieved.ID)
}

func TestGetLatestPricing_NotFound(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	_, err := db.GetLatestPricing(model.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no valid pricing found")
}

func TestListPricing_NoFilters(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	// Create multiple pricing records
	for i := 0; i < 3; i++ {
		pricing := createTestPricing(model.ID)
		pricing.InputTokenCost = float64(i+1) * 0.01
		err := db.CreatePricing(pricing)
		require.NoError(t, err)
	}

	pricings, err := db.ListPricing(nil)
	require.NoError(t, err)
	assert.Len(t, pricings, 3)
}

func TestListPricing_FilterByModelID(t *testing.T) {
	db := setupPricingTestDB(t)
	model1 := createTestPricingModel(t, db)

	// Create second model
	provider := &Provider{
		Name:     "test-provider-2",
		Endpoint: "http://test2.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model2 := &Model{
		ProviderID: provider.ID,
		ModelID:    "test-model-2",
		Name:       "Test Model 2",
	}
	err = db.CreateModel(model2)
	require.NoError(t, err)

	// Create pricing for model1
	for i := 0; i < 3; i++ {
		pricing := createTestPricing(model1.ID)
		err = db.CreatePricing(pricing)
		require.NoError(t, err)
	}

	// Create pricing for model2
	pricing := createTestPricing(model2.ID)
	err = db.CreatePricing(pricing)
	require.NoError(t, err)

	// Filter by model1
	pricings, err := db.ListPricing(map[string]interface{}{"model_id": model1.ID})
	require.NoError(t, err)
	assert.Len(t, pricings, 3)
}

func TestListPricing_FilterByCurrency(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	currencies := []string{"USD", "EUR", "GBP"}
	for _, currency := range currencies {
		pricing := createTestPricing(model.ID)
		pricing.Currency = currency
		err := db.CreatePricing(pricing)
		require.NoError(t, err)
	}

	pricings, err := db.ListPricing(map[string]interface{}{"currency": "USD"})
	require.NoError(t, err)
	assert.Len(t, pricings, 1)
	assert.Equal(t, "USD", pricings[0].Currency)
}

func TestUpdatePricing(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	pricing := createTestPricing(model.ID)
	err := db.CreatePricing(pricing)
	require.NoError(t, err)

	// Update
	pricing.InputTokenCost = 0.05
	pricing.OutputTokenCost = 0.10
	err = db.UpdatePricing(pricing)
	require.NoError(t, err)

	// Verify
	retrieved, err := db.GetPricing(pricing.ID)
	require.NoError(t, err)
	assert.Equal(t, 0.05, retrieved.InputTokenCost)
	assert.Equal(t, 0.10, retrieved.OutputTokenCost)
}

func TestDeletePricing(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	pricing := createTestPricing(model.ID)
	err := db.CreatePricing(pricing)
	require.NoError(t, err)

	err = db.DeletePricing(pricing.ID)
	require.NoError(t, err)

	_, err = db.GetPricing(pricing.ID)
	assert.Error(t, err)
}

// ==================== Limit CRUD Tests ====================

func createTestLimit(modelID int64) *Limit {
	return &Limit{
		ModelID:     modelID,
		LimitType:   "requests_per_minute",
		LimitValue:  100,
		ResetPeriod: "minute",
		IsHardLimit: true,
	}
}

func TestCreateLimit(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	limit := createTestLimit(model.ID)
	err := db.CreateLimit(limit)
	require.NoError(t, err)
	assert.NotZero(t, limit.ID)
}

func TestGetLimit(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	limit := createTestLimit(model.ID)
	err := db.CreateLimit(limit)
	require.NoError(t, err)

	retrieved, err := db.GetLimit(limit.ID)
	require.NoError(t, err)
	assert.Equal(t, limit.ID, retrieved.ID)
	assert.Equal(t, limit.ModelID, retrieved.ModelID)
	assert.Equal(t, limit.LimitType, retrieved.LimitType)
	assert.Equal(t, limit.LimitValue, retrieved.LimitValue)
}

func TestGetLimit_NotFound(t *testing.T) {
	db := setupPricingTestDB(t)

	_, err := db.GetLimit(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetLimitsForModel(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	// Create multiple limits
	limitTypes := []string{"requests_per_minute", "tokens_per_day", "requests_per_hour"}
	for _, limitType := range limitTypes {
		limit := createTestLimit(model.ID)
		limit.LimitType = limitType
		err := db.CreateLimit(limit)
		require.NoError(t, err)
	}

	limits, err := db.GetLimitsForModel(model.ID)
	require.NoError(t, err)
	assert.Len(t, limits, 3)
}

func TestListLimits_NoFilters(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	for i := 0; i < 3; i++ {
		limit := createTestLimit(model.ID)
		limit.LimitType = "type_" + string(rune('A'+i))
		err := db.CreateLimit(limit)
		require.NoError(t, err)
	}

	limits, err := db.ListLimits(nil)
	require.NoError(t, err)
	assert.Len(t, limits, 3)
}

func TestListLimits_FilterByModelID(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	for i := 0; i < 2; i++ {
		limit := createTestLimit(model.ID)
		err := db.CreateLimit(limit)
		require.NoError(t, err)
	}

	limits, err := db.ListLimits(map[string]interface{}{"model_id": model.ID})
	require.NoError(t, err)
	assert.Len(t, limits, 2)
}

func TestUpdateLimitCurrentUsage(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	limit := createTestLimit(model.ID)
	err := db.CreateLimit(limit)
	require.NoError(t, err)

	err = db.UpdateLimitCurrentUsage(limit.ID, 50)
	require.NoError(t, err)

	retrieved, err := db.GetLimit(limit.ID)
	require.NoError(t, err)
	assert.Equal(t, 50, retrieved.CurrentUsage)
}

func TestResetLimitUsage(t *testing.T) {
	db := setupPricingTestDB(t)

	// Just verify the function doesn't error
	err := db.ResetLimitUsage()
	require.NoError(t, err)
}

func TestUpdateLimit(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	limit := createTestLimit(model.ID)
	err := db.CreateLimit(limit)
	require.NoError(t, err)

	// Update
	limit.LimitValue = 200
	limit.IsHardLimit = false
	err = db.UpdateLimit(limit)
	require.NoError(t, err)

	retrieved, err := db.GetLimit(limit.ID)
	require.NoError(t, err)
	assert.Equal(t, 200, retrieved.LimitValue)
	assert.False(t, retrieved.IsHardLimit)
}

func TestDeleteLimit(t *testing.T) {
	db := setupPricingTestDB(t)
	model := createTestPricingModel(t, db)

	limit := createTestLimit(model.ID)
	err := db.CreateLimit(limit)
	require.NoError(t, err)

	err = db.DeleteLimit(limit.ID)
	require.NoError(t, err)

	_, err = db.GetLimit(limit.ID)
	assert.Error(t, err)
}

// ==================== Struct Tests ====================

func TestPricing_Struct(t *testing.T) {
	now := time.Now()
	later := now.Add(30 * 24 * time.Hour)

	pricing := Pricing{
		ID:                   1,
		ModelID:              100,
		InputTokenCost:       0.01,
		OutputTokenCost:      0.03,
		CachedInputTokenCost: 0.005,
		StorageCost:          0.0001,
		RequestCost:          0.001,
		Currency:             "USD",
		PricingModel:         "per_token",
		EffectiveFrom:        &now,
		EffectiveTo:          &later,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	assert.Equal(t, int64(1), pricing.ID)
	assert.Equal(t, int64(100), pricing.ModelID)
	assert.Equal(t, 0.01, pricing.InputTokenCost)
	assert.Equal(t, "USD", pricing.Currency)
}

func TestLimit_Struct(t *testing.T) {
	now := time.Now()

	limit := Limit{
		ID:           1,
		ModelID:      100,
		LimitType:    "requests_per_minute",
		LimitValue:   100,
		CurrentUsage: 50,
		ResetPeriod:  "minute",
		ResetTime:    &now,
		IsHardLimit:  true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, int64(1), limit.ID)
	assert.Equal(t, "requests_per_minute", limit.LimitType)
	assert.Equal(t, 100, limit.LimitValue)
	assert.True(t, limit.IsHardLimit)
}
