package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CogneeDB implements VectorDatabase interface for Cognee
type CogneeDB struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewCogneeDB creates a new Cognee vector database client
func NewCogneeDB(baseURL, apiKey string) *CogneeDB {
	return &CogneeDB{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Store stores a document with its vector in Cognee
func (c *CogneeDB) Store(ctx context.Context, doc *Document) error {
	payload := map[string]interface{}{
		"id":        doc.ID,
		"content":   doc.Content,
		"metadata":  doc.Metadata,
		"vector":    doc.Vector,
		"source":    doc.Source,
		"timestamp": doc.Timestamp.Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/documents", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Cognee API returned status %d", resp.StatusCode)
	}

	return nil
}

// Search searches for similar documents in Cognee
func (c *CogneeDB) Search(ctx context.Context, queryVector Vector, limit int, threshold float64) ([]SearchResult, error) {
	payload := map[string]interface{}{
		"vector":    queryVector,
		"limit":     limit,
		"threshold": threshold,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search payload: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/search", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Cognee search API returned status %d", resp.StatusCode)
	}

	var response struct {
		Results []struct {
			Document Document `json:"document"`
			Score    float64  `json:"score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	results := make([]SearchResult, len(response.Results))
	for i, result := range response.Results {
		results[i] = SearchResult{
			Document: result.Document,
			Score:    result.Score,
		}
	}

	return results, nil
}

// Delete removes a document by ID from Cognee
func (c *CogneeDB) Delete(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/api/v1/documents/%s", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("Cognee delete API returned status %d", resp.StatusCode)
	}

	return nil
}

// Get retrieves a document by ID from Cognee
func (c *CogneeDB) Get(ctx context.Context, id string) (*Document, error) {
	url := fmt.Sprintf("%s/api/v1/documents/%s", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create get request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("Cognee get API returned status %d", resp.StatusCode)
	}

	var doc Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, fmt.Errorf("failed to decode document: %w", err)
	}

	return &doc, nil
}

// List returns all document IDs from Cognee
func (c *CogneeDB) List(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/documents", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Cognee list API returned status %d", resp.StatusCode)
	}

	var response struct {
		Documents []string `json:"documents"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode list response: %w", err)
	}

	return response.Documents, nil
}

// Close closes the Cognee database connection
func (c *CogneeDB) Close() error {
	// Cognee uses HTTP, so no persistent connection to close
	return nil
}
