# Configuration Export Comprehensive Challenge

## Overview
This challenge validates system's ability to export configurations for OpenCode, Crush, Claude Code, and other AI coding agents.

## Challenge Type
Integration Test + Validation Test + Format Test

## Test Scenarios

### 1. OpenCode Configuration Export Challenge
**Objective**: Verify configuration export for OpenCode

**Steps**:
1. Select models for export
2. Generate OpenCode config
3. Validate JSON format
4. Verify API key handling
5. Verify model priority

**Expected Results**:
- Config is valid JSON
- Format matches OpenCode specification
- API keys included or redacted
- Models prioritized by score

**Test Code**:
```go
func TestOpenCodeConfigExport(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "gpt-4", Provider: "openai", Score: 95, APIKey: "sk-xxx"},
        {ID: "claude-3-opus", Provider: "anthropic", Score: 92, APIKey: "sk-yyy"},
    }

    config, err := exporter.ExportOpenCode(models, ExportOptions{RedactKeys: false})
    assert.NoError(t, err)

    var ocConfig OpenCodeConfig
    err = json.Unmarshal(config, &ocConfig)
    assert.NoError(t, err)

    assert.Equal(t, 2, len(ocConfig.Providers))
    assert.Equal(t, "gpt-4", ocConfig.Providers[0].Models[0].ID)
    assert.Equal(t, "sk-xxx", ocConfig.Providers[0].APIKey)
}
```

---

### 2. Crush Configuration Export Challenge
**Objective**: Verify configuration export for Crush

**Steps**:
1. Select models for export
2. Generate Crush config
3. Validate YAML format
4. Verify API key handling
5. Verify model ordering

**Expected Results**:
- Config is valid YAML
- Format matches Crush specification
- API keys handled correctly
- Models sorted by score

**Test Code**:
```go
func TestCrushConfigExport(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "gpt-4", Provider: "openai", Score: 95, APIKey: "sk-xxx"},
        {ID: "claude-3-opus", Provider: "anthropic", Score: 92, APIKey: "sk-yyy"},
    }

    config, err := exporter.ExportCrush(models, ExportOptions{RedactKeys: false})
    assert.NoError(t, err)

    var crushConfig CrushConfig
    err = yaml.Unmarshal(config, &crushConfig)
    assert.NoError(t, err)

    assert.Equal(t, 2, len(crushConfig.Providers))
    assert.Equal(t, "gpt-4", crushConfig.Providers[0].Models[0].ID)
}
```

---

### 3. Claude Code Configuration Export Challenge
**Objective**: Verify configuration export for Claude Code

**Steps**:
1. Select models for export
2. Generate Claude Code config
3. Validate JSON format
4. Verify Anthropic-specific settings
5. Verify model capabilities

**Expected Results**:
- Config is valid JSON
- Format matches Claude Code spec
- Anthropic settings included
- Capabilities mapped correctly

**Test Code**:
```go
func TestClaudeCodeConfigExport(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "claude-3-opus", Provider: "anthropic", Score: 92,
         Capabilities: []string{"vision", "tools", "200k_context"}},
    }

    config, err := exporter.ExportClaudeCode(models, ExportOptions{RedactKeys: false})
    assert.NoError(t, err)

    var ccConfig ClaudeCodeConfig
    err = json.Unmarshal(config, &ccConfig)
    assert.NoError(t, err)

    assert.Equal(t, "claude-3-opus", ccConfig.Models[0].ID)
    assert.Contains(t, ccConfig.Models[0].Capabilities, "vision")
}
```

---

### 4. Multiple Platforms Export Challenge
**Objective**: Verify simultaneous export for multiple platforms

**Steps**:
1. Select models for all platforms
2. Export for OpenCode
3. Export for Crush
4. Export for Claude Code
5. Verify all exports

**Expected Results**:
- All exports generated
- Each export in correct format
- All exports valid

**Test Code**:
```go
func TestMultiplePlatformsExport(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "gpt-4", Provider: "openai", Score: 95},
        {ID: "claude-3-opus", Provider: "anthropic", Score: 92},
    }

    exports, err := exporter.ExportAll(models, ExportOptions{RedactKeys: false})
    assert.NoError(t, err)
    assert.Contains(t, exports, "opencode")
    assert.Contains(t, exports, "crush")
    assert.Contains(t, exports, "claude_code")
}
```

---

### 5. API Key Redaction Challenge
**Objective**: Verify API keys can be redacted

**Steps**:
1. Export config with keys
2. Export config with redaction
3. Verify keys are masked
4. Verify format remains valid

**Expected Results**:
- Redacted keys show placeholders
- Format remains valid
- Unredacted exports have full keys

**Test Code**:
```go
func TestAPIKeyRedaction(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "gpt-4", Provider: "openai", Score: 95, APIKey: "sk-1234567890"},
    }

    // Without redaction
    config1, _ := exporter.ExportOpenCode(models, ExportOptions{RedactKeys: false})
    assert.Contains(t, string(config1), "sk-1234567890")

    // With redaction
    config2, _ := exporter.ExportOpenCode(models, ExportOptions{RedactKeys: true})
    assert.NotContains(t, string(config2), "sk-1234567890")
    assert.Contains(t, string(config2), "YOUR_API_KEY")
}
```

---

### 6. Score-Based Prioritization Challenge
**Objective**: Verify models are prioritized by score

**Steps**:
1. Export with score-based ordering
2. Verify highest score first
3. Verify threshold filtering
4. Verify top N selection

**Expected Results**:
- Models sorted by score
- Threshold filters applied
- Top N selected correctly

**Test Code**:
```go
func TestScorePrioritization(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "gpt-3.5", Score: 80},
        {ID: "gpt-4", Score: 95},
        {ID: "claude-3-opus", Score: 92},
    }

    config, _ := exporter.ExportOpenCode(models, ExportOptions{
        MinScore: 90,
        TopN:     2,
    })

    var ocConfig OpenCodeConfig
    json.Unmarshal(config, &ocConfig)

    // Should have top 2 models with score >= 90
    assert.Equal(t, 2, len(ocConfig.Providers[0].Models))
    assert.Equal(t, "gpt-4", ocConfig.Providers[0].Models[0].ID)
    assert.Equal(t, "claude-3-opus", ocConfig.Providers[0].Models[1].ID)
}
```

---

### 7. Provider-Specific Export Challenge
**Objective**: Verify export for specific providers

**Steps**:
1. Export only OpenAI models
2. Export only Anthropic models
3. Export multiple providers
4. Verify correct provider mappings

**Expected Results**:
- Only selected providers exported
- Provider mappings correct
- Provider-specific settings included

**Test Code**:
```go
func TestProviderSpecificExport(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "gpt-4", Provider: "openai", Score: 95},
        {ID: "claude-3-opus", Provider: "anthropic", Score: 92},
        {ID: "gemini-pro", Provider: "google", Score: 88},
    }

    // Export only OpenAI
    config, _ := exporter.ExportOpenCode(models, ExportOptions{
        Providers: []string{"openai"},
    })

    var ocConfig OpenCodeConfig
    json.Unmarshal(config, &ocConfig)

    assert.Equal(t, 1, len(ocConfig.Providers))
    assert.Equal(t, "openai", ocConfig.Providers[0].Name)
    assert.Equal(t, 1, len(ocConfig.Providers[0].Models))
}
```

---

### 8. Feature-Based Export Challenge
**Objective**: Verify export based on features

**Steps**:
1. Export models with specific features
2. Verify feature filtering
3. Test feature combinations
4. Verify capability mappings

**Expected Results**:
- Models with selected features exported
- Feature filtering works
- Capability mappings correct

**Test Code**:
```go
func TestFeatureBasedExport(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{
        {ID: "gpt-4", Features: []string{"streaming", "vision", "tools"}},
        {ID: "gpt-3.5", Features: []string{"streaming"}},
        {ID: "claude-3-opus", Features: []string{"streaming", "vision", "tools"}},
    }

    config, _ := exporter.ExportOpenCode(models, ExportOptions{
        RequiredFeatures: []string{"vision", "tools"},
    })

    var ocConfig OpenCodeConfig
    json.Unmarshal(config, &ocConfig)

    // Should only export models with vision AND tools
    assert.Equal(t, 2, len(ocConfig.Providers[0].Models))
}
```

---

### 9. Configuration Validation Challenge
**Objective**: Verify exported configurations are valid

**Steps**:
1. Export configuration
2. Validate against schema
3. Test loading in target platform
4. Verify all required fields

**Expected Results**:
- Schema validation passes
- Configuration loads in platform
- All required fields present

**Test Code**:
```go
func TestConfigurationValidation(t *testing.T) {
    exporter := NewConfigExporter()

    models := []Model{{ID: "gpt-4", Provider: "openai", Score: 95, APIKey: "sk-xxx"}}

    config, _ := exporter.ExportOpenCode(models, ExportOptions{})

    // Validate against schema
    validator := NewSchemaValidator()
    err := validator.Validate("opencode_schema.json", config)
    assert.NoError(t, err)
}
```

---

### 10. Export History Challenge
**Objective**: Verify export history is tracked

**Steps**:
1. Export configuration
2. Record in history
3. List exports
4. Compare versions
5. Revert to previous export

**Expected Results**:
- Exports recorded in history
- History accessible
- Versions comparable
- Revert works

**Test Code**:
```go
func TestExportHistory(t *testing.T) {
    exporter := NewConfigExporter()
    history := NewExportHistory()

    models := []Model{{ID: "gpt-4", Provider: "openai", Score: 95}}

    config1, _ := exporter.ExportOpenCode(models, ExportOptions{})
    history.Record("opencode", config1, time.Now())

    models[0].Score = 90
    config2, _ := exporter.ExportOpenCode(models, ExportOptions{})
    history.Record("opencode", config2, time.Now())

    exports := history.List("opencode")
    assert.Equal(t, 2, len(exports))

    previous := history.GetPrevious("opencode")
    assert.Equal(t, config1, previous)
}
```

---

## Success Criteria

### Functional Requirements
- [ ] OpenCode export works
- [ ] Crush export works
- [ ] Claude Code export works
- [ ] Multiple platforms export works
- [ ] API key redaction works
- [ ] Score prioritization works
- [ ] Provider filtering works
- [ ] Feature filtering works
- [ ] Configuration validation works
- [ ] Export history works

### Format Requirements
- [ ] JSON format valid
- [ ] YAML format valid
- [ ] Schema validation passes
- [ ] Platform specs matched

### Security Requirements
- [ ] API keys protected
- [ ] Redaction works
- [ ] No sensitive data in unencrypted exports

## Dependencies
- Valid model data
- Platform schemas

## Cleanup
- Remove exported configs
