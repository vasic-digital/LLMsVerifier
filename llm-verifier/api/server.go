// Package api contains HTTP API handlers
// Temporarily commented out due to dependency issues
package api

import (
	"fmt"
	"net/http"
	"runtime"

	"digital.vasic.llmsverifier/config"
	"digital.vasic.llmsverifier/database"
)

// Server represents the REST API server
type Server struct {
	config   *config.Config
	database *database.Database
	server   *http.Server
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *database.Database) *Server {
	return &Server{
		config:   cfg,
		database: db,
	}
}

// Router returns the HTTP router for testing purposes
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/api/health", s.HealthHandler)
	mux.HandleFunc("/api/models", s.ListModelsHandler)
	mux.HandleFunc("/api/models/", s.GetModelHandler)
	mux.HandleFunc("/api/models/{id}/verify", s.VerifyModelHandler)
	mux.HandleFunc("/api/providers", s.ProvidersHandler)
	mux.HandleFunc("/api/metrics", s.MetricsHandler)
	mux.HandleFunc("/metrics", s.MetricsHandler)

	return mux
}

// Start starts the HTTP server on the specified port
func (s *Server) Start(port string) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", s.HealthHandler)
	mux.HandleFunc("/api/models", s.ListModelsHandler)
	mux.HandleFunc("/api/models/", s.GetModelHandler)
	mux.HandleFunc("/api/models/{id}/verify", s.VerifyModelHandler)
	mux.HandleFunc("/api/providers", s.ProvidersHandler)
	mux.HandleFunc("/api/metrics", s.MetricsHandler)
	mux.HandleFunc("/metrics", s.MetricsHandler)

	if port == "" {
		port = "8080"
	}

	s.server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	return s.server.ListenAndServe()
}

// MetricsHandler exposes a minimal but REAL Prometheus text-format exposition
// (stdlib only — no external dependency, no collector cascade). Prior to this
// the app registered no metrics route, so prometheus could not scrape it
// (target down, 404). These are genuine runtime gauges, not a stub.
func (s *Server) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	dbUp := 0
	if s.database != nil {
		dbUp = 1
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprint(w, "# HELP llmverifier_up 1 if the API server is serving.\n# TYPE llmverifier_up gauge\nllmverifier_up 1\n")
	fmt.Fprintf(w, "# HELP llmverifier_database_up 1 if the database handle is initialised.\n# TYPE llmverifier_database_up gauge\nllmverifier_database_up %d\n", dbUp)
	fmt.Fprintf(w, "# HELP llmverifier_goroutines Current number of goroutines.\n# TYPE llmverifier_goroutines gauge\nllmverifier_goroutines %d\n", runtime.NumGoroutine())
	fmt.Fprintf(w, "# HELP llmverifier_memory_alloc_bytes Allocated heap bytes.\n# TYPE llmverifier_memory_alloc_bytes gauge\nllmverifier_memory_alloc_bytes %d\n", m.Alloc)
	fmt.Fprintf(w, "# HELP llmverifier_memory_sys_bytes Bytes obtained from the OS.\n# TYPE llmverifier_memory_sys_bytes gauge\nllmverifier_memory_sys_bytes %d\n", m.Sys)
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}
