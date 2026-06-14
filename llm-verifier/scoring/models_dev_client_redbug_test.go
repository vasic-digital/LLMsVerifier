package scoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestMakeRequest_HTTP3Disabled_DoesNotPanic is a reproduce-first RED test for
// the nil-response fallback bug in makeRequest. When UseHTTP3 is false, the
// HTTP/3 branch is skipped leaving resp==nil AND err==nil; the fallback guard
// `resp == nil && err != nil` is then false, so the HTTP/2 request is never
// issued and `defer resp.Body.Close()` dereferences a nil *http.Response.
func TestMakeRequest_HTTP3Disabled_DoesNotPanic(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"models":[{"model_id":"m1","model":"M1","provider_id":"p1"}]}`))
	}))
	defer srv.Close()

	cfg := ClientConfig{
		BaseURL:        srv.URL,
		RequestTimeout: 5 * time.Second,
		UseHTTP3:       false, // HTTP/3 disabled -> first branch is skipped
		UseBrotli:      false,
		EnableLogging:  false,
	}

	client, err := NewModelsDevClient(cfg, nil)
	if err != nil {
		t.Fatalf("NewModelsDevClient returned error: %v", err)
	}

	resp, err := client.FetchAllModels(context.Background())
	if err != nil {
		t.Fatalf("FetchAllModels returned error: %v", err)
	}
	if resp == nil || len(resp.Models) != 1 || resp.Models[0].ModelID != "m1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
