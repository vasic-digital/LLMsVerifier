package llmops

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// InMemoryPromptRegistry implements PromptRegistry with in-memory storage
type InMemoryPromptRegistry struct {
	mu       sync.RWMutex
	prompts  map[string]map[string]*PromptVersion // name -> version -> prompt
	active   map[string]string                    // name -> active version
	logger   *log.Logger
}

// NewInMemoryPromptRegistry creates a new in-memory prompt registry
func NewInMemoryPromptRegistry(logger *log.Logger) *InMemoryPromptRegistry {
	if logger == nil {
		logger = log.Default()
	}
	return &InMemoryPromptRegistry{
		prompts: make(map[string]map[string]*PromptVersion),
		active:  make(map[string]string),
		logger:  logger,
	}
}

// Create creates a new prompt version
func (r *InMemoryPromptRegistry) Create(ctx context.Context, prompt *PromptVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if prompt.ID == "" {
		prompt.ID = uuid.New().String()
	}
	if prompt.CreatedAt.IsZero() {
		prompt.CreatedAt = time.Now()
	}
	prompt.UpdatedAt = time.Now()

	if _, ok := r.prompts[prompt.Name]; !ok {
		r.prompts[prompt.Name] = make(map[string]*PromptVersion)
	}

	// First version becomes active
	if len(r.prompts[prompt.Name]) == 0 {
		prompt.IsActive = true
		r.active[prompt.Name] = prompt.Version
	}

	r.prompts[prompt.Name][prompt.Version] = prompt
	r.logger.Printf("Created prompt %s version %s", prompt.Name, prompt.Version)
	return nil
}

// Get retrieves a specific prompt version
func (r *InMemoryPromptRegistry) Get(ctx context.Context, name, version string) (*PromptVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, ok := r.prompts[name]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", name)
	}

	prompt, ok := versions[version]
	if !ok {
		return nil, fmt.Errorf("version not found: %s@%s", name, version)
	}

	return prompt, nil
}

// GetLatest retrieves the active/latest prompt version
func (r *InMemoryPromptRegistry) GetLatest(ctx context.Context, name string) (*PromptVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	activeVersion, ok := r.active[name]
	if !ok {
		return nil, fmt.Errorf("no active version for prompt: %s", name)
	}

	return r.prompts[name][activeVersion], nil
}

// List lists all versions of a prompt
func (r *InMemoryPromptRegistry) List(ctx context.Context, name string) ([]*PromptVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	versions, ok := r.prompts[name]
	if !ok {
		return nil, nil
	}

	result := make([]*PromptVersion, 0, len(versions))
	for _, p := range versions {
		result = append(result, p)
	}
	return result, nil
}

// Activate activates a specific version
func (r *InMemoryPromptRegistry) Activate(ctx context.Context, name, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions, ok := r.prompts[name]
	if !ok {
		return fmt.Errorf("prompt not found: %s", name)
	}

	if _, ok := versions[version]; !ok {
		return fmt.Errorf("version not found: %s@%s", name, version)
	}

	// Deactivate old version
	if oldVersion, ok := r.active[name]; ok {
		if p, ok := versions[oldVersion]; ok {
			p.IsActive = false
		}
	}

	// Activate new version
	versions[version].IsActive = true
	r.active[name] = version

	r.logger.Printf("Activated prompt %s version %s", name, version)
	return nil
}

// Delete deletes a prompt version
func (r *InMemoryPromptRegistry) Delete(ctx context.Context, name, version string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	versions, ok := r.prompts[name]
	if !ok {
		return fmt.Errorf("prompt not found: %s", name)
	}

	prompt, ok := versions[version]
	if !ok {
		return fmt.Errorf("version not found: %s@%s", name, version)
	}

	if prompt.IsActive {
		return fmt.Errorf("cannot delete active version: %s@%s", name, version)
	}

	delete(versions, version)
	r.logger.Printf("Deleted prompt %s version %s", name, version)
	return nil
}

// Render renders a prompt with variables
func (r *InMemoryPromptRegistry) Render(ctx context.Context, name, version string, vars map[string]interface{}) (string, error) {
	prompt, err := r.Get(ctx, name, version)
	if err != nil {
		return "", err
	}

	content := prompt.Content

	// Apply defaults for missing variables
	for _, v := range prompt.Variables {
		if _, ok := vars[v.Name]; !ok && v.Default != nil {
			vars[v.Name] = v.Default
		}
	}

	// Validate required variables
	for _, v := range prompt.Variables {
		if v.Required {
			if _, ok := vars[v.Name]; !ok {
				return "", fmt.Errorf("missing required variable: %s", v.Name)
			}
		}
	}

	// Simple template replacement
	for k, v := range vars {
		placeholder := fmt.Sprintf("{{%s}}", k)
		content = strings.ReplaceAll(content, placeholder, fmt.Sprintf("%v", v))
	}

	return content, nil
}

// PromptVersionComparator compares prompt versions
type PromptVersionComparator struct {
	registry PromptRegistry
	logger   *log.Logger
}

// NewPromptVersionComparator creates a new comparator
func NewPromptVersionComparator(registry PromptRegistry, logger *log.Logger) *PromptVersionComparator {
	if logger == nil {
		logger = log.Default()
	}
	return &PromptVersionComparator{
		registry: registry,
		logger:   logger,
	}
}

// PromptDiff represents differences between versions
type PromptDiff struct {
	OldVersion  string   `json:"old_version"`
	NewVersion  string   `json:"new_version"`
	ContentDiff string   `json:"content_diff"`
	AddedVars   []string `json:"added_vars"`
	RemovedVars []string `json:"removed_vars"`
}

// Compare compares two prompt versions
func (c *PromptVersionComparator) Compare(ctx context.Context, name, version1, version2 string) (*PromptDiff, error) {
	p1, err := c.registry.Get(ctx, name, version1)
	if err != nil {
		return nil, err
	}

	p2, err := c.registry.Get(ctx, name, version2)
	if err != nil {
		return nil, err
	}

	diff := &PromptDiff{
		OldVersion: version1,
		NewVersion: version2,
	}

	// Simple diff - in production use proper diff library
	if p1.Content != p2.Content {
		diff.ContentDiff = fmt.Sprintf("Content changed from %d to %d chars", len(p1.Content), len(p2.Content))
	}

	// Compare variables
	v1Vars := make(map[string]bool)
	for _, v := range p1.Variables {
		v1Vars[v.Name] = true
	}

	v2Vars := make(map[string]bool)
	for _, v := range p2.Variables {
		v2Vars[v.Name] = true
		if !v1Vars[v.Name] {
			diff.AddedVars = append(diff.AddedVars, v.Name)
		}
	}

	for _, v := range p1.Variables {
		if !v2Vars[v.Name] {
			diff.RemovedVars = append(diff.RemovedVars, v.Name)
		}
	}

	return diff, nil
}
