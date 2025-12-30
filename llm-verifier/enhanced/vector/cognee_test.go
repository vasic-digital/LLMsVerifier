package vector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCogneeDB(t *testing.T) {
	db := NewCogneeDB("http://localhost:8080", "test-api-key")
	assert.NotNil(t, db)
	assert.Equal(t, "http://localhost:8080", db.baseURL)
	assert.Equal(t, "test-api-key", db.apiKey)
	assert.NotNil(t, db.httpClient)
}

func TestCogneeDB_Store(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/documents", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer")

		// Decode body
		var body map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&body)
		require.NoError(t, err)
		assert.Equal(t, "doc-1", body["id"])
		assert.Equal(t, "Test content", body["content"])

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	doc := &Document{
		ID:        "doc-1",
		Content:   "Test content",
		Vector:    Vector{0.1, 0.2, 0.3},
		Timestamp: time.Now(),
	}

	err := db.Store(ctx, doc)
	assert.NoError(t, err)
}

func TestCogneeDB_Store_Success200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Some APIs return 200 instead of 201
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	doc := &Document{ID: "doc-1", Content: "Test", Vector: Vector{0.1}}
	err := db.Store(ctx, doc)
	assert.NoError(t, err)
}

func TestCogneeDB_Store_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	doc := &Document{ID: "doc-1", Content: "Test", Vector: Vector{0.1}}
	err := db.Store(ctx, doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCogneeDB_Store_ConnectionError(t *testing.T) {
	// Use a URL that will fail to connect
	db := NewCogneeDB("http://localhost:99999", "test-key")
	ctx := context.Background()

	doc := &Document{ID: "doc-1", Content: "Test", Vector: Vector{0.1}}
	err := db.Store(ctx, doc)
	assert.Error(t, err)
}

func TestCogneeDB_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/search", r.URL.Path)

		response := map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"document": map[string]interface{}{
						"id":      "doc-1",
						"content": "Similar content",
					},
					"score": 0.95,
				},
				{
					"document": map[string]interface{}{
						"id":      "doc-2",
						"content": "Another match",
					},
					"score": 0.85,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	results, err := db.Search(ctx, Vector{0.1, 0.2, 0.3}, 10, 0.5)
	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "doc-1", results[0].Document.ID)
	assert.Equal(t, 0.95, results[0].Score)
}

func TestCogneeDB_Search_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	_, err := db.Search(ctx, Vector{0.1, 0.2}, 10, 0.5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCogneeDB_Search_ConnectionError(t *testing.T) {
	db := NewCogneeDB("http://localhost:99999", "test-key")
	ctx := context.Background()

	_, err := db.Search(ctx, Vector{0.1}, 10, 0.5)
	assert.Error(t, err)
}

func TestCogneeDB_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v1/documents/doc-1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	err := db.Delete(ctx, "doc-1")
	assert.NoError(t, err)
}

func TestCogneeDB_Delete_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	err := db.Delete(ctx, "doc-1")
	assert.NoError(t, err)
}

func TestCogneeDB_Delete_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	err := db.Delete(ctx, "doc-1")
	assert.Error(t, err)
}

func TestCogneeDB_Delete_ConnectionError(t *testing.T) {
	db := NewCogneeDB("http://localhost:99999", "test-key")
	ctx := context.Background()

	err := db.Delete(ctx, "doc-1")
	assert.Error(t, err)
}

func TestCogneeDB_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/documents/doc-1", r.URL.Path)

		doc := Document{
			ID:      "doc-1",
			Content: "Retrieved content",
			Vector:  Vector{0.1, 0.2},
			Source:  "test",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(doc)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	doc, err := db.Get(ctx, "doc-1")
	require.NoError(t, err)
	assert.Equal(t, "doc-1", doc.ID)
	assert.Equal(t, "Retrieved content", doc.Content)
}

func TestCogneeDB_Get_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	_, err := db.Get(ctx, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestCogneeDB_Get_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	_, err := db.Get(ctx, "doc-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestCogneeDB_Get_ConnectionError(t *testing.T) {
	db := NewCogneeDB("http://localhost:99999", "test-key")
	ctx := context.Background()

	_, err := db.Get(ctx, "doc-1")
	assert.Error(t, err)
}

func TestCogneeDB_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/documents", r.URL.Path)

		response := map[string]interface{}{
			"documents": []string{"doc-1", "doc-2", "doc-3"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	ids, err := db.List(ctx)
	require.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "doc-1")
}

func TestCogneeDB_List_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	db := NewCogneeDB(server.URL, "test-key")
	ctx := context.Background()

	_, err := db.List(ctx)
	assert.Error(t, err)
}

func TestCogneeDB_List_ConnectionError(t *testing.T) {
	db := NewCogneeDB("http://localhost:99999", "test-key")
	ctx := context.Background()

	_, err := db.List(ctx)
	assert.Error(t, err)
}

func TestCogneeDB_Close(t *testing.T) {
	db := NewCogneeDB("http://localhost:8080", "test-key")
	err := db.Close()
	assert.NoError(t, err)
}

// Test interface compliance
func TestCogneeDB_VectorDatabaseInterface(t *testing.T) {
	var _ VectorDatabase = (*CogneeDB)(nil)
}
