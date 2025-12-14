-- LLM Verifier Database Schema
-- SQLite with SQL Cipher encryption

-- Enable foreign keys
PRAGMA foreign_keys = ON;

-- Providers table (companies/organizations providing LLMs)
CREATE TABLE providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    endpoint TEXT NOT NULL,
    api_key_encrypted TEXT,
    description TEXT,
    website TEXT,
    support_email TEXT,
    documentation_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_checked TIMESTAMP,
    is_active BOOLEAN DEFAULT 1,
    reliability_score REAL DEFAULT 0.0,
    average_response_time_ms INTEGER DEFAULT 0
);

-- Models table (individual LLM models)
CREATE TABLE models (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL,
    model_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    version TEXT,
    architecture TEXT,
    parameter_count INTEGER,
    context_window_tokens INTEGER,
    max_output_tokens INTEGER,
    training_data_cutoff DATE,
    release_date DATE,
    is_multimodal BOOLEAN DEFAULT 0,
    supports_vision BOOLEAN DEFAULT 0,
    supports_audio BOOLEAN DEFAULT 0,
    supports_video BOOLEAN DEFAULT 0,
    supports_reasoning BOOLEAN DEFAULT 0,
    open_source BOOLEAN DEFAULT 0,
    deprecated BOOLEAN DEFAULT 0,
    tags TEXT, -- JSON array of tags
    language_support TEXT, -- JSON array of supported languages
    use_case TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_verified TIMESTAMP,
    verification_status TEXT DEFAULT 'pending', -- pending, verified, failed, deprecated
    overall_score REAL DEFAULT 0.0,
    code_capability_score REAL DEFAULT 0.0,
    responsiveness_score REAL DEFAULT 0.0,
    reliability_score REAL DEFAULT 0.0,
    feature_richness_score REAL DEFAULT 0.0,
    value_proposition_score REAL DEFAULT 0.0,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

-- Pricing table (model pricing information)
CREATE TABLE pricing (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    input_token_cost REAL DEFAULT 0.0, -- Cost per 1M input tokens
    output_token_cost REAL DEFAULT 0.0, -- Cost per 1M output tokens
    cached_input_token_cost REAL DEFAULT 0.0,
    storage_cost REAL DEFAULT 0.0,
    request_cost REAL DEFAULT 0.0,
    currency TEXT DEFAULT 'USD',
    pricing_model TEXT DEFAULT 'per_token', -- per_token, per_request, per_hour, etc.
    effective_from DATE,
    effective_to DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);

-- Limits table (rate limits and quotas)
CREATE TABLE limits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    limit_type TEXT NOT NULL, -- requests_per_minute, requests_per_hour, requests_per_day, tokens_per_minute, etc.
    limit_value INTEGER NOT NULL,
    current_usage INTEGER DEFAULT 0,
    reset_period TEXT, -- minute, hour, day, week, month
    reset_time TIMESTAMP,
    is_hard_limit BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);

-- Verification results table (individual verification runs)
CREATE TABLE verification_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    verification_type TEXT NOT NULL, -- full, quick, scheduled, manual
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    status TEXT DEFAULT 'running', -- running, completed, failed, cancelled
    error_message TEXT,
    
    -- Availability metrics
    exists BOOLEAN,
    responsive BOOLEAN,
    overloaded BOOLEAN,
    latency_ms INTEGER,
    
    -- Feature detection results
    supports_tool_use BOOLEAN DEFAULT 0,
    supports_function_calling BOOLEAN DEFAULT 0,
    supports_code_generation BOOLEAN DEFAULT 0,
    supports_code_completion BOOLEAN DEFAULT 0,
    supports_code_review BOOLEAN DEFAULT 0,
    supports_code_explanation BOOLEAN DEFAULT 0,
    supports_embeddings BOOLEAN DEFAULT 0,
    supports_reranking BOOLEAN DEFAULT 0,
    supports_image_generation BOOLEAN DEFAULT 0,
    supports_audio_generation BOOLEAN DEFAULT 0,
    supports_video_generation BOOLEAN DEFAULT 0,
    supports_mcps BOOLEAN DEFAULT 0,
    supports_lsps BOOLEAN DEFAULT 0,
    supports_multimodal BOOLEAN DEFAULT 0,
    supports_streaming BOOLEAN DEFAULT 0,
    supports_json_mode BOOLEAN DEFAULT 0,
    supports_structured_output BOOLEAN DEFAULT 0,
    supports_reasoning BOOLEAN DEFAULT 0,
    supports_parallel_tool_use BOOLEAN DEFAULT 0,
    max_parallel_calls INTEGER DEFAULT 0,
    supports_batch_processing BOOLEAN DEFAULT 0,
    
    -- Code capability results
    code_language_support TEXT, -- JSON array
    code_debugging BOOLEAN DEFAULT 0,
    code_optimization BOOLEAN DEFAULT 0,
    test_generation BOOLEAN DEFAULT 0,
    documentation_generation BOOLEAN DEFAULT 0,
    refactoring BOOLEAN DEFAULT 0,
    error_resolution BOOLEAN DEFAULT 0,
    architecture_design BOOLEAN DEFAULT 0,
    security_assessment BOOLEAN DEFAULT 0,
    pattern_recognition BOOLEAN DEFAULT 0,
    debugging_accuracy REAL DEFAULT 0.0,
    max_handled_depth INTEGER DEFAULT 0,
    code_quality_score REAL DEFAULT 0.0,
    logic_correctness_score REAL DEFAULT 0.0,
    runtime_efficiency_score REAL DEFAULT 0.0,
    
    -- Performance scores
    overall_score REAL DEFAULT 0.0,
    code_capability_score REAL DEFAULT 0.0,
    responsiveness_score REAL DEFAULT 0.0,
    reliability_score REAL DEFAULT 0.0,
    feature_richness_score REAL DEFAULT 0.0,
    value_proposition_score REAL DEFAULT 0.0,
    
    -- Detailed scoring breakdown (JSON)
    score_details TEXT,
    
    -- Response time metrics
    avg_latency_ms INTEGER DEFAULT 0,
    p95_latency_ms INTEGER DEFAULT 0,
    min_latency_ms INTEGER DEFAULT 0,
    max_latency_ms INTEGER DEFAULT 0,
    throughput_rps REAL DEFAULT 0.0,
    
    -- Raw response data (for debugging)
    raw_request TEXT,
    raw_response TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);

-- Issues table (documented problems with models)
CREATE TABLE issues (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    issue_type TEXT NOT NULL, -- availability, performance, accuracy, security, etc.
    severity TEXT NOT NULL, -- critical, high, medium, low
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    symptoms TEXT,
    workarounds TEXT,
    affected_features TEXT, -- JSON array
    first_detected TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_occurred TIMESTAMP,
    resolved_at TIMESTAMP,
    resolution_notes TEXT,
    verification_result_id INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
    FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE SET NULL
);

-- Events table (system events and notifications)
CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL, -- verification_started, verification_completed, score_changed, issue_detected, etc.
    severity TEXT DEFAULT 'info', -- debug, info, warning, error, critical
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT, -- JSON with additional data
    model_id INTEGER,
    provider_id INTEGER,
    verification_result_id INTEGER,
    issue_id INTEGER,
    user_id INTEGER, -- For future multi-user support
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE CASCADE,
    FOREIGN KEY (issue_id) REFERENCES issues(id) ON DELETE CASCADE
);

-- Schedules table (for periodic re-tests)
CREATE TABLE schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    schedule_type TEXT NOT NULL, -- cron, interval, manual
    cron_expression TEXT,
    interval_seconds INTEGER,
    target_type TEXT NOT NULL, -- all_models, provider, specific_model
    target_id INTEGER, -- provider_id or model_id depending on target_type
    is_active BOOLEAN DEFAULT 1,
    last_run TIMESTAMP,
    next_run TIMESTAMP,
    run_count INTEGER DEFAULT 0,
    max_runs INTEGER, -- NULL for unlimited
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT, -- For future multi-user support
    FOREIGN KEY (target_id) REFERENCES models(id) ON DELETE CASCADE
);

-- Schedule runs table (execution history)
CREATE TABLE schedule_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id INTEGER NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    status TEXT DEFAULT 'running', -- running, completed, failed, cancelled
    results_count INTEGER DEFAULT 0,
    errors_count INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE
);

-- Configuration exports table (exported configs for CLI agents)
CREATE TABLE config_exports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    export_type TEXT NOT NULL, -- opencode, crush, claude_code, custom
    name TEXT NOT NULL,
    description TEXT,
    config_data TEXT NOT NULL, -- JSON configuration data
    target_models TEXT, -- JSON array of model_ids, NULL for all
    target_providers TEXT, -- JSON array of provider_ids, NULL for all
    filters TEXT, -- JSON with filtering criteria
    is_verified BOOLEAN DEFAULT 0,
    verification_notes TEXT,
    created_by TEXT, -- For future multi-user support
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    download_count INTEGER DEFAULT 0
);

-- Logs table (structured application logs)
CREATE TABLE logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    level TEXT NOT NULL, -- DEBUG, INFO, WARNING, ERROR, CRITICAL
    logger TEXT NOT NULL,
    message TEXT NOT NULL,
    details TEXT, -- JSON with additional context
    request_id TEXT, -- For request tracing
    user_id INTEGER, -- For future multi-user support
    model_id INTEGER,
    provider_id INTEGER,
    verification_result_id INTEGER,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE,
    FOREIGN KEY (provider_id) REFERENCES providers(id) ON DELETE CASCADE,
    FOREIGN KEY (verification_result_id) REFERENCES verification_results(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_providers_endpoint ON providers(endpoint);
CREATE INDEX idx_providers_active ON providers(is_active);
CREATE INDEX idx_models_provider ON models(provider_id);
CREATE INDEX idx_models_model_id ON models(model_id);
CREATE INDEX idx_models_verification_status ON models(verification_status);
CREATE INDEX idx_models_overall_score ON models(overall_score);
CREATE INDEX idx_pricing_model ON pricing(model_id);
CREATE INDEX idx_pricing_effective ON pricing(effective_from, effective_to);
CREATE INDEX idx_limits_model ON limits(model_id);
CREATE INDEX idx_limits_type ON limits(limit_type);
CREATE INDEX idx_verification_results_model ON verification_results(model_id);
CREATE INDEX idx_verification_results_status ON verification_results(status);
CREATE INDEX idx_verification_results_timestamp ON verification_results(created_at);
CREATE INDEX idx_issues_model ON issues(model_id);
CREATE INDEX idx_issues_severity ON issues(severity);
CREATE INDEX idx_issues_resolved ON issues(resolved_at);
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_timestamp ON events(created_at);
CREATE INDEX idx_events_model ON events(model_id);
CREATE INDEX idx_schedules_active ON schedules(is_active);
CREATE INDEX idx_schedules_next_run ON schedules(next_run);
CREATE INDEX idx_schedule_runs_schedule ON schedule_runs(schedule_id);
CREATE INDEX idx_config_exports_type ON config_exports(export_type);
CREATE INDEX idx_logs_timestamp ON logs(timestamp);
CREATE INDEX idx_logs_level ON logs(level);
CREATE INDEX idx_logs_logger ON logs(logger);
CREATE INDEX idx_logs_request_id ON logs(request_id);

-- Views for common queries
CREATE VIEW model_summary AS
SELECT 
    m.id,
    m.model_id,
    m.name,
    m.description,
    p.name as provider_name,
    m.overall_score,
    m.verification_status,
    m.last_verified,
    m.is_multimodal,
    m.supports_vision,
    m.supports_audio,
    m.supports_reasoning,
    m.deprecated,
    vr.overall_score as latest_score,
    vr.avg_latency_ms as latest_latency,
    COUNT(i.id) as open_issues
FROM models m
JOIN providers p ON m.provider_id = p.id
LEFT JOIN verification_results vr ON vr.model_id = m.id AND vr.status = 'completed'
LEFT JOIN issues i ON i.model_id = m.id AND i.resolved_at IS NULL
WHERE vr.id = (SELECT MAX(id) FROM verification_results WHERE model_id = m.id)
   OR vr.id IS NULL
GROUP BY m.id;

CREATE VIEW provider_summary AS
SELECT 
    p.id,
    p.name,
    p.endpoint,
    p.is_active,
    p.reliability_score,
    p.average_response_time_ms,
    COUNT(m.id) as total_models,
    COUNT(CASE WHEN m.verification_status = 'verified' THEN 1 END) as verified_models,
    AVG(m.overall_score) as average_model_score,
    MAX(p.last_checked) as last_checked
FROM providers p
LEFT JOIN models m ON m.provider_id = p.id
GROUP BY p.id;

CREATE VIEW recent_verifications AS
SELECT 
    vr.id,
    m.name as model_name,
    p.name as provider_name,
    vr.started_at,
    vr.completed_at,
    vr.status,
    vr.overall_score,
    vr.avg_latency_ms,
    vr.error_message
FROM verification_results vr
JOIN models m ON vr.model_id = m.id
JOIN providers p ON m.provider_id = p.id
WHERE vr.created_at >= datetime('now', '-7 days')
ORDER BY vr.created_at DESC;

-- Full-text search setup (if needed)
-- Note: SQLite FTS would require additional setup

-- Triggers for updated_at timestamps
CREATE TRIGGER update_providers_timestamp 
AFTER UPDATE ON providers
BEGIN
    UPDATE providers SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_models_timestamp 
AFTER UPDATE ON models
BEGIN
    UPDATE models SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_pricing_timestamp 
AFTER UPDATE ON pricing
BEGIN
    UPDATE pricing SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_limits_timestamp 
AFTER UPDATE ON limits
BEGIN
    UPDATE limits SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_issues_timestamp 
AFTER UPDATE ON issues
BEGIN
    UPDATE issues SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_schedules_timestamp 
AFTER UPDATE ON schedules
BEGIN
    UPDATE schedules SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_config_exports_timestamp 
AFTER UPDATE ON config_exports
BEGIN
    UPDATE config_exports SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;