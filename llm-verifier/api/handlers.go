// Package api contains HTTP API handlers
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"digital.vasic.llmsverifier/database"
)

// HealthHandler handles health check requests
func (s *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	health := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	}

	if s.database != nil {
		health["database"] = "connected"
		if err := s.database.Ping(); err == nil {
			health["database_status"] = "ok"
		} else {
			health["database_status"] = "error"
			health["database_error"] = err.Error()
		}
	} else {
		health["database"] = "not_configured"
	}

	json.NewEncoder(w).Encode(health)
}

// ListModelsHandler handles listing all models
func (s *Server) ListModelsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters for filtering
	filters := make(map[string]interface{})
	if providerID := r.URL.Query().Get("provider_id"); providerID != "" {
		if id, err := strconv.ParseInt(providerID, 10, 64); err == nil {
			filters["provider_id"] = id
		}
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filters["verification_status"] = status
	}
	if minScore := r.URL.Query().Get("min_score"); minScore != "" {
		if score, err := strconv.ParseFloat(minScore, 64); err == nil {
			filters["min_score"] = score
		}
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}
	if limit := r.URL.Query().Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filters["limit"] = l
		}
	}

	// Get models from database
	models, err := s.database.ListModels(filters)
	if err != nil {
		http.Error(w, "Failed to retrieve models: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// HXC-135: batch-resolve the latest completed VerificationResult per model
	// so the CONST-040 capability flags (MCP/LSP/ACP/RAG/Skills/Plugins) can be
	// sourced from the verifier's own computed data — never hardcoded — with a
	// single query instead of N+1 lookups.
	modelIDs := make([]int64, 0, len(models))
	for _, model := range models {
		modelIDs = append(modelIDs, model.ID)
	}
	latestByModel := latestVerificationResultsByModelID(s.database, modelIDs)

	// Transform models to API response format
	modelResponses := make([]map[string]any, 0, len(models))
	for _, model := range models {
		// Get provider name if available
		providerName := ""
		if provider, err := s.database.GetProvider(model.ProviderID); err == nil {
			providerName = provider.Name
		}

		// Build capabilities list from model features
		capabilities := buildCapabilitiesList(model)

		resp := map[string]any{
			// HXC-134: the model id MUST be emitted as text end-to-end —
			// database.Model.ID is an internal int64 primary key, but every
			// consumer's wire contract (helix_code's
			// internal/verifier.VerifiedModel.ID, this package's own
			// pkg/api.Model.ID) declares it a string. Convert at the JSON
			// boundary so the internal storage type stays int64 while the
			// emitted type is consistently string.
			"id":           strconv.FormatInt(model.ID, 10),
			"model_id":     model.ModelID,
			"name":         model.Name,
			"provider":     providerName,
			"provider_id":  model.ProviderID,
			"status":       model.VerificationStatus,
			"score":        model.OverallScore,
			"capabilities": capabilities,
			"description":  model.Description,
			"version":      model.Version,
			"deprecated":   model.Deprecated,
			"created_at":   model.CreatedAt,
			"updated_at":   model.UpdatedAt,
		}
		// HXC-135 / CONST-040: publish the six advanced-capability indicators
		// sourced from the model's latest computed VerificationResult.
		addCapabilityFields(resp, latestByModel[model.ID])
		modelResponses = append(modelResponses, resp)
	}

	json.NewEncoder(w).Encode(map[string]any{
		"models": modelResponses,
		"count":  len(modelResponses),
	})
}

// addCapabilityFields publishes the six CONST-040 advanced-capability
// indicators (MCP, LSP, ACP, RAG/embedding-retrieval, Skills, Plugins) into
// resp, sourced from vr — the model's latest COMPLETED database.VerificationResult
// (the verifier's own computed source of truth; see verification/verification.go's
// Verify, which sets these six fields from the real C4 capability probes:
// SupportsMCPs/SupportsLSPs/SupportsACPs/SupportsRAG/SupportsSkills/
// SupportsPlugins). Field names follow the verifier's OWN canonical DB/JSON
// tag convention (database.go: "supports_mcps"/"supports_lsps"/"supports_acps"
// for MCP/LSP/ACP — plural, matching the historical DB column names — and
// "supports_rag"/"supports_skills"/"supports_plugins" for the rest), NOT a
// consumer-specific schema (CONST-069 decoupling).
//
// vr is nil when the model has no completed verification result yet — in
// that case every flag is published as its honest zero value (false, "not
// yet verified as supporting"), never a fabricated true (CONST-036/037/040:
// LLMsVerifier is the sole source of truth; no hardcoded capability flags).
func addCapabilityFields(resp map[string]any, vr *database.VerificationResult) {
	if vr == nil {
		resp["supports_mcps"] = false
		resp["supports_lsps"] = false
		resp["supports_acps"] = false
		resp["supports_rag"] = false
		resp["supports_skills"] = false
		resp["supports_plugins"] = false
		return
	}
	resp["supports_mcps"] = vr.SupportsMCPs
	resp["supports_lsps"] = vr.SupportsLSPs
	resp["supports_acps"] = vr.SupportsACPs
	resp["supports_rag"] = vr.SupportsRAG
	resp["supports_skills"] = vr.SupportsSkills
	resp["supports_plugins"] = vr.SupportsPlugins
}

// latestVerificationResultsByModelID batch-resolves the latest completed
// database.VerificationResult per model id via a single query, returning a
// map for O(1) per-model lookup. Any error is treated as "no completed
// verification result available yet" (honest empty map) rather than failing
// the whole listing response — capability flags degrade to their honest false
// default per addCapabilityFields, they never block model listing/retrieval.
func latestVerificationResultsByModelID(db *database.Database, modelIDs []int64) map[int64]*database.VerificationResult {
	out := make(map[int64]*database.VerificationResult, len(modelIDs))
	if db == nil || len(modelIDs) == 0 {
		return out
	}
	results, err := db.GetLatestVerificationResults(modelIDs)
	if err != nil {
		return out
	}
	for _, r := range results {
		out[r.ModelID] = r
	}
	return out
}

// buildCapabilitiesList builds a list of capabilities from model features
func buildCapabilitiesList(model *database.Model) []string {
	capabilities := []string{}
	if model.IsMultimodal {
		capabilities = append(capabilities, "multimodal")
	}
	if model.SupportsVision {
		capabilities = append(capabilities, "vision")
	}
	if model.SupportsAudio {
		capabilities = append(capabilities, "audio")
	}
	if model.SupportsVideo {
		capabilities = append(capabilities, "video")
	}
	if model.SupportsReasoning {
		capabilities = append(capabilities, "reasoning")
	}
	// Add default text capability
	capabilities = append(capabilities, "text")
	return capabilities
}

// GetModelHandler handles getting a single model
func (s *Server) GetModelHandler(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from path: /api/models/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/models/")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	// Check for verify suffix
	if strings.HasSuffix(path, "/verify") {
		s.VerifyModelHandler(w, r)
		return
	}

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse model ID
	modelID, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get model from database
	model, err := s.database.GetModel(modelID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to retrieve model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get provider name
	providerName := ""
	if provider, err := s.database.GetProvider(model.ProviderID); err == nil {
		providerName = provider.Name
	}

	// Build capabilities list
	capabilities := buildCapabilitiesList(model)

	// Format context window
	contextWindow := ""
	if model.ContextWindowTokens != nil {
		contextWindow = formatTokenCount(*model.ContextWindowTokens)
	}

	// Format parameter count
	parameters := ""
	if model.ParameterCount != nil {
		parameters = formatParameterCount(*model.ParameterCount)
	}

	response := map[string]any{
		// HXC-134: emit the model id as text end-to-end (see the identical
		// note in ListModelsHandler above).
		"id":                     strconv.FormatInt(model.ID, 10),
		"model_id":               model.ModelID,
		"name":                   model.Name,
		"provider":               providerName,
		"provider_id":            model.ProviderID,
		"status":                 model.VerificationStatus,
		"score":                  model.OverallScore,
		"capabilities":           capabilities,
		"description":            model.Description,
		"version":                model.Version,
		"architecture":           model.Architecture,
		"parameters":             parameters,
		"context":                contextWindow,
		"context_window_tokens":  model.ContextWindowTokens,
		"max_output_tokens":      model.MaxOutputTokens,
		"is_multimodal":          model.IsMultimodal,
		"supports_vision":        model.SupportsVision,
		"supports_audio":         model.SupportsAudio,
		"supports_video":         model.SupportsVideo,
		"supports_reasoning":     model.SupportsReasoning,
		"open_source":            model.OpenSource,
		"deprecated":             model.Deprecated,
		"tags":                   model.Tags,
		"use_case":               model.UseCase,
		"code_capability_score":  model.CodeCapabilityScore,
		"responsiveness_score":   model.ResponsivenessScore,
		"reliability_score":      model.ReliabilityScore,
		"feature_richness_score": model.FeatureRichnessScore,
		"last_verified":          model.LastVerified,
		"created_at":             model.CreatedAt,
		"updated_at":             model.UpdatedAt,
	}
	// HXC-135 / CONST-040: publish the six advanced-capability indicators
	// sourced from the model's latest computed VerificationResult.
	latest := latestVerificationResultsByModelID(s.database, []int64{model.ID})
	addCapabilityFields(response, latest[model.ID])

	json.NewEncoder(w).Encode(response)
}

// formatTokenCount formats token count to human readable string
func formatTokenCount(tokens int) string {
	if tokens >= 1000000 {
		return strconv.FormatFloat(float64(tokens)/1000000, 'f', 1, 64) + "M tokens"
	}
	if tokens >= 1000 {
		return strconv.Itoa(tokens/1000) + "K tokens"
	}
	return strconv.Itoa(tokens) + " tokens"
}

// formatParameterCount formats parameter count to human readable string
func formatParameterCount(params int64) string {
	if params >= 1000000000000 {
		return strconv.FormatFloat(float64(params)/1000000000000, 'f', 2, 64) + " trillion"
	}
	if params >= 1000000000 {
		return strconv.FormatFloat(float64(params)/1000000000, 'f', 1, 64) + " billion"
	}
	if params >= 1000000 {
		return strconv.FormatFloat(float64(params)/1000000, 'f', 1, 64) + " million"
	}
	return strconv.FormatInt(params, 10)
}

// VerifyModelHandler handles model verification
func (s *Server) VerifyModelHandler(w http.ResponseWriter, r *http.Request) {
	// Extract model ID from path: /api/models/{id}/verify
	path := strings.TrimPrefix(r.URL.Path, "/api/models/")
	parts := strings.Split(path, "/")
	if len(parts) < 1 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	modelIDStr := parts[0]

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse model ID
	modelID, err := strconv.ParseInt(modelIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid model ID", http.StatusBadRequest)
		return
	}

	// Verify model exists
	model, err := s.database.GetModel(modelID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "Failed to retrieve model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Resolve the provider so the verifier can reach the model's live API.
	provider, err := s.database.GetProvider(model.ProviderID)
	if err != nil {
		http.Error(w, "Failed to retrieve provider: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Record the verification run for audit/history.
	verificationResult := &database.VerificationResult{
		ModelID:          modelID,
		VerificationType: "comprehensive",
		StartedAt:        time.Now(),
		Status:           "running",
	}
	if err := s.database.CreateVerificationResult(verificationResult); err != nil {
		http.Error(w, "Failed to create verification job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Run the REAL verification synchronously, then persist the honest outcome
	// onto the model so /api/models reflects the verified status + score.
	// A model that runs but does not pass is persisted as "failed"/"error" —
	// the handler NEVER marks a model "verified" unless the verifier says so.
	status, score, verr := s.verifier.Verify(r.Context(), model, provider)

	completedAt := time.Now()
	verificationResult.CompletedAt = &completedAt
	if verr != nil {
		status = "error"
		msg := verr.Error()
		verificationResult.ErrorMessage = &msg
	}
	// Persist the run record outcome (best-effort — the model-side persistence
	// below is the authoritative source of truth surfaced to /api/models).
	verificationResult.Status = status
	_ = s.database.UpdateVerificationResult(verificationResult)

	// Persist the verification status + score onto the model itself.
	model.VerificationStatus = status
	model.OverallScore = score
	model.LastVerified = &completedAt
	if err := s.database.UpdateModel(model); err != nil {
		http.Error(w, "Failed to persist verification result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	resp := map[string]any{
		"status": status,
		// HXC-134: emit the model id as text end-to-end (see the identical
		// note in ListModelsHandler above); modelID is the int64 parsed from
		// the URL path.
		"model_id":            strconv.FormatInt(modelID, 10),
		"model_name":          model.Name,
		"verification_status": status,
		"score":               score,
		"message":             tr("api.handler.verification_process_initiated"),
		"job_id":              verificationResult.ID,
		"verification_id":     verificationResult.ID,
		"started_at":          verificationResult.StartedAt,
		"completed_at":        completedAt,
	}
	// HXC-135 / CONST-040: publish the six advanced-capability indicators
	// sourced from the model's latest computed VerificationResult (this run's
	// own result if it was the capability-probing verification.Verifier path,
	// or the most recent prior completed one otherwise — never fabricated).
	latest := latestVerificationResultsByModelID(s.database, []int64{modelID})
	addCapabilityFields(resp, latest[modelID])

	json.NewEncoder(w).Encode(resp)
}

// ListProvidersHandler handles listing all providers
func (s *Server) ListProvidersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	// Parse query parameters for filtering
	filters := make(map[string]interface{})
	if isActive := r.URL.Query().Get("is_active"); isActive != "" {
		filters["is_active"] = isActive == "true"
	}
	if search := r.URL.Query().Get("search"); search != "" {
		filters["search"] = search
	}

	// Get providers from database
	providers, err := s.database.ListProviders(filters)
	if err != nil {
		http.Error(w, "Failed to retrieve providers: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform providers to API response format
	providerResponses := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		// Count models for this provider
		modelFilters := map[string]interface{}{"provider_id": provider.ID}
		models, _ := s.database.ListModels(modelFilters)
		modelCount := len(models)

		status := "inactive"
		if provider.IsActive {
			status = "active"
		}

		providerResponses = append(providerResponses, map[string]any{
			"id":                       provider.ID,
			"name":                     provider.Name,
			"status":                   status,
			"is_active":                provider.IsActive,
			"models":                   modelCount,
			"api_url":                  provider.Endpoint,
			"endpoint":                 provider.Endpoint,
			"description":              provider.Description,
			"website":                  provider.Website,
			"support_email":            provider.SupportEmail,
			"documentation_url":        provider.DocumentationURL,
			"reliability_score":        provider.ReliabilityScore,
			"average_response_time_ms": provider.AverageResponseTimeMs,
			"last_checked":             provider.LastChecked,
			"created_at":               provider.CreatedAt,
			"updated_at":               provider.UpdatedAt,
		})
	}

	json.NewEncoder(w).Encode(map[string]any{
		"providers": providerResponses,
		"count":     len(providerResponses),
	})
}

// ProvidersHandler handles both GET (list) and POST (add) for providers
func (s *Server) ProvidersHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.ListProvidersHandler(w, r)
	case http.MethodPost:
		s.AddProviderHandler(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// AddProviderHandler handles adding a new provider
func (s *Server) AddProviderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check if database is available
	if s.database == nil {
		http.Error(w, "Database not available", http.StatusServiceUnavailable)
		return
	}

	var providerData map[string]any
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&providerData); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate required fields
	name, nameOK := providerData["name"].(string)
	endpoint, endpointOK := providerData["endpoint"].(string)
	if !nameOK || name == "" {
		http.Error(w, "Missing required field: name", http.StatusBadRequest)
		return
	}
	if !endpointOK || endpoint == "" {
		// Use api_url as fallback
		if apiURL, ok := providerData["api_url"].(string); ok && apiURL != "" {
			endpoint = apiURL
		} else {
			http.Error(w, "Missing required field: endpoint or api_url", http.StatusBadRequest)
			return
		}
	}

	// Create provider object
	provider := &database.Provider{
		Name:     name,
		Endpoint: endpoint,
		IsActive: true,
	}

	// Optional fields
	if desc, ok := providerData["description"].(string); ok {
		provider.Description = desc
	}
	if website, ok := providerData["website"].(string); ok {
		provider.Website = website
	}
	if email, ok := providerData["support_email"].(string); ok {
		provider.SupportEmail = email
	}
	if docURL, ok := providerData["documentation_url"].(string); ok {
		provider.DocumentationURL = docURL
	}
	if isActive, ok := providerData["is_active"].(bool); ok {
		provider.IsActive = isActive
	}

	// Create in database
	err := s.database.CreateProvider(provider)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			http.Error(w, "Provider with this name already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create provider: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]any{
		"status":     "provider_added",
		"id":         provider.ID,
		"name":       provider.Name,
		"endpoint":   provider.Endpoint,
		"is_active":  provider.IsActive,
		"created_at": provider.CreatedAt,
	})
}
