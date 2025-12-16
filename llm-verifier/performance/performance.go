package performance

import (
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// CacheManager provides multi-level caching
type CacheManager struct {
	redisClient CacheBackend // Interface for Redis or other backends
	localCache  map[string]CacheItem
	mu          sync.RWMutex
	ttl         time.Duration
}

// CacheBackend interface for cache backends
type CacheBackend interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	FlushAll(ctx context.Context) error
	HIncrBy(ctx context.Context, key string, field string, incr int64) error
}

// CacheItem represents a cached item
type CacheItem struct {
	Value      interface{}
	ExpiresAt  time.Time
	HitCount   int64
	LastAccess time.Time
}

// NewCacheManager creates a new cache manager
func NewCacheManager(backend CacheBackend, ttl time.Duration) *CacheManager {
	return &CacheManager{
		redisClient: backend,
		localCache:  make(map[string]CacheItem),
		ttl:         ttl,
	}
}

// NewMemoryCacheBackend creates an in-memory cache backend
func NewMemoryCacheBackend() CacheBackend {
	return &MemoryCacheBackend{
		data: make(map[string]MemoryItem),
	}
}

// MemoryCacheBackend implements in-memory caching
type MemoryCacheBackend struct {
	data map[string]MemoryItem
	mu   sync.RWMutex
}

type MemoryItem struct {
	Value     string
	ExpiresAt time.Time
}

func (mcb *MemoryCacheBackend) Get(ctx context.Context, key string) (string, error) {
	mcb.mu.RLock()
	defer mcb.mu.RUnlock()

	if item, exists := mcb.data[key]; exists && time.Now().Before(item.ExpiresAt) {
		return item.Value, nil
	}
	return "", fmt.Errorf("key not found")
}

func (mcb *MemoryCacheBackend) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	mcb.data[key] = MemoryItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (mcb *MemoryCacheBackend) Del(ctx context.Context, key string) error {
	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	delete(mcb.data, key)
	return nil
}

func (mcb *MemoryCacheBackend) FlushAll(ctx context.Context) error {
	mcb.mu.Lock()
	defer mcb.mu.Unlock()

	mcb.data = make(map[string]MemoryItem)
	return nil
}

func (mcb *MemoryCacheBackend) HIncrBy(ctx context.Context, key string, field string, incr int64) error {
	// Simple implementation for memory backend
	return nil
}

// Get retrieves a value from cache
func (cm *CacheManager) Get(key string) (interface{}, bool) {
	// Try local cache first
	cm.mu.RLock()
	if item, exists := cm.localCache[key]; exists && time.Now().Before(item.ExpiresAt) {
		item.HitCount++
		item.LastAccess = time.Now()
		cm.mu.RUnlock()

		// Update hit count in Redis asynchronously
		go cm.incrementHitCount(key)
		return item.Value, true
	}
	cm.mu.RUnlock()

	// Try backend cache
	ctx := context.Background()
	val, err := cm.redisClient.Get(ctx, key)
	if err == nil {
		// Cache hit in backend, store locally
		var value interface{}
		// In real implementation, you'd deserialize based on type
		value = val

		cm.mu.Lock()
		cm.localCache[key] = CacheItem{
			Value:      value,
			ExpiresAt:  time.Now().Add(cm.ttl),
			HitCount:   1,
			LastAccess: time.Now(),
		}
		cm.mu.Unlock()

		return value, true
	}

	return nil, false
}

// Set stores a value in cache
func (cm *CacheManager) Set(key string, value interface{}, customTTL ...time.Duration) error {
	ttl := cm.ttl
	if len(customTTL) > 0 {
		ttl = customTTL[0]
	}

	// Store in local cache
	cm.mu.Lock()
	cm.localCache[key] = CacheItem{
		Value:      value,
		ExpiresAt:  time.Now().Add(ttl),
		HitCount:   0,
		LastAccess: time.Now(),
	}
	cm.mu.Unlock()

	// Store in backend
	ctx := context.Background()
	// In real implementation, you'd serialize the value properly
	val := fmt.Sprintf("%v", value)
	return cm.redisClient.Set(ctx, key, val, ttl)
}

// Delete removes a value from cache
func (cm *CacheManager) Delete(key string) error {
	// Remove from local cache
	cm.mu.Lock()
	delete(cm.localCache, key)
	cm.mu.Unlock()

	// Remove from backend
	ctx := context.Background()
	return cm.redisClient.Del(ctx, key)
}

// Clear clears all cache entries
func (cm *CacheManager) Clear() error {
	// Clear local cache
	cm.mu.Lock()
	cm.localCache = make(map[string]CacheItem)
	cm.mu.Unlock()

	// Clear backend
	ctx := context.Background()
	return cm.redisClient.FlushAll(ctx)
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	totalItems := len(cm.localCache)
	totalHits := int64(0)
	oldestAccess := time.Now()

	for _, item := range cm.localCache {
		totalHits += item.HitCount
		if item.LastAccess.Before(oldestAccess) {
			oldestAccess = item.LastAccess
		}
	}

	return map[string]interface{}{
		"local_cache_items": totalItems,
		"total_hits":        totalHits,
		"oldest_access":     oldestAccess,
		"redis_connected":   cm.redisClient != nil,
	}
}

// Cleanup removes expired items from local cache
func (cm *CacheManager) Cleanup() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()
	for key, item := range cm.localCache {
		if now.After(item.ExpiresAt) {
			delete(cm.localCache, key)
		}
	}
}

// StartCleanup starts automatic cleanup goroutine
func (cm *CacheManager) StartCleanup(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			cm.Cleanup()
		}
	}()
}

func (cm *CacheManager) incrementHitCount(key string) {
	ctx := context.Background()
	cm.redisClient.HIncrBy(ctx, "cache:hits", key, 1)
}

// LoadBalancer provides load balancing for multiple instances
type LoadBalancer struct {
	instances    []string
	current      int
	mu           sync.RWMutex
	healthChecks map[string]bool
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(instances []string) *LoadBalancer {
	healthChecks := make(map[string]bool)
	for _, instance := range instances {
		healthChecks[instance] = true
	}

	return &LoadBalancer{
		instances:    instances,
		current:      0,
		healthChecks: healthChecks,
	}
}

// NextInstance returns the next healthy instance using round-robin
func (lb *LoadBalancer) NextInstance() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Find next healthy instance
	start := lb.current
	for i := 0; i < len(lb.instances); i++ {
		instance := lb.instances[lb.current]
		lb.current = (lb.current + 1) % len(lb.instances)

		if lb.healthChecks[instance] {
			return instance
		}
	}

	// If no healthy instances found, return first one anyway
	lb.current = start
	return lb.instances[0]
}

// MarkHealthy marks an instance as healthy
func (lb *LoadBalancer) MarkHealthy(instance string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.healthChecks[instance] = true
}

// MarkUnhealthy marks an instance as unhealthy
func (lb *LoadBalancer) MarkUnhealthy(instance string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.healthChecks[instance] = false
}

// GetHealthStatus returns health status of all instances
func (lb *LoadBalancer) GetHealthStatus() map[string]bool {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	status := make(map[string]bool)
	for k, v := range lb.healthChecks {
		status[k] = v
	}
	return status
}

// DatabaseOptimizer provides database query optimization
type DatabaseOptimizer struct {
	queryStats map[string]QueryStat
	mu         sync.RWMutex
}

type QueryStat struct {
	Query     string
	Count     int64
	TotalTime time.Duration
	AvgTime   time.Duration
	LastRun   time.Time
	SlowCount int64 // Queries taking > 1 second
}

// NewDatabaseOptimizer creates a new database optimizer
func NewDatabaseOptimizer() *DatabaseOptimizer {
	return &DatabaseOptimizer{
		queryStats: make(map[string]QueryStat),
	}
}

// RecordQuery records a query execution
func (dbo *DatabaseOptimizer) RecordQuery(query string, duration time.Duration) {
	queryHash := fmt.Sprintf("%x", md5.Sum([]byte(query)))

	dbo.mu.Lock()
	defer dbo.mu.Unlock()

	stat := dbo.queryStats[queryHash]
	stat.Query = query
	stat.Count++
	stat.TotalTime += duration
	stat.AvgTime = time.Duration(int64(stat.TotalTime) / stat.Count)
	stat.LastRun = time.Now()

	if duration > time.Second {
		stat.SlowCount++
	}

	dbo.queryStats[queryHash] = stat
}

// GetSlowQueries returns queries that are frequently slow
func (dbo *DatabaseOptimizer) GetSlowQueries(threshold time.Duration, minCount int64) []QueryStat {
	dbo.mu.RLock()
	defer dbo.mu.RUnlock()

	var slowQueries []QueryStat
	for _, stat := range dbo.queryStats {
		if stat.Count >= minCount && stat.AvgTime > threshold {
			slowQueries = append(slowQueries, stat)
		}
	}

	return slowQueries
}

// GetQueryStats returns all query statistics
func (dbo *DatabaseOptimizer) GetQueryStats() map[string]QueryStat {
	dbo.mu.RLock()
	defer dbo.mu.RUnlock()

	stats := make(map[string]QueryStat)
	for k, v := range dbo.queryStats {
		stats[k] = v
	}
	return stats
}

// SuggestIndexes suggests indexes based on query patterns
func (dbo *DatabaseOptimizer) SuggestIndexes() []string {
	dbo.mu.RLock()
	defer dbo.mu.RUnlock()

	var suggestions []string

	// Analyze queries for potential indexes
	for _, stat := range dbo.queryStats {
		if stat.Count > 10 { // Frequently executed queries
			// Simple analysis - look for WHERE clauses
			query := stat.Query
			if suggestions := dbo.analyzeQueryForIndexes(query); len(suggestions) > 0 {
				for _, suggestion := range suggestions {
					suggestions = append(suggestions, suggestion)
				}
			}
		}
	}

	return suggestions
}

func (dbo *DatabaseOptimizer) analyzeQueryForIndexes(query string) []string {
	// Simple analysis for demonstration
	// Real implementation would parse SQL and suggest indexes
	var suggestions []string

	if contains(query, "WHERE") && contains(query, "status") {
		suggestions = append(suggestions, "CREATE INDEX idx_models_status ON models(status)")
	}

	if contains(query, "WHERE") && contains(query, "provider_id") {
		suggestions = append(suggestions, "CREATE INDEX idx_models_provider ON models(provider_id)")
	}

	return suggestions
}

// ConnectionPoolManager manages database connection pooling
type ConnectionPoolManager struct {
	maxConnections     int
	minConnections     int
	currentConnections int
	mu                 sync.RWMutex
}

// NewConnectionPoolManager creates a new connection pool manager
func NewConnectionPoolManager(maxConn, minConn int) *ConnectionPoolManager {
	return &ConnectionPoolManager{
		maxConnections:     maxConn,
		minConnections:     minConn,
		currentConnections: minConn,
	}
}

// GetConnection gets a connection from the pool
func (cpm *ConnectionPoolManager) GetConnection() (interface{}, error) {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	if cpm.currentConnections >= cpm.maxConnections {
		return nil, fmt.Errorf("connection pool exhausted")
	}

	cpm.currentConnections++
	// In real implementation, return actual database connection
	return fmt.Sprintf("connection_%d", cpm.currentConnections), nil
}

// ReturnConnection returns a connection to the pool
func (cpm *ConnectionPoolManager) ReturnConnection(conn interface{}) {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()

	if cpm.currentConnections > cpm.minConnections {
		cpm.currentConnections--
	}
}

// GetStats returns connection pool statistics
func (cpm *ConnectionPoolManager) GetStats() map[string]interface{} {
	cpm.mu.RLock()
	defer cpm.mu.RUnlock()

	return map[string]interface{}{
		"max_connections":       cpm.maxConnections,
		"min_connections":       cpm.minConnections,
		"current_connections":   cpm.currentConnections,
		"available_connections": cpm.maxConnections - cpm.currentConnections,
	}
}

// PerformanceMonitor monitors overall system performance
type PerformanceMonitor struct {
	cacheManager *CacheManager
	dbOptimizer  *DatabaseOptimizer
	connPool     *ConnectionPoolManager
	startTime    time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(cacheMgr *CacheManager, dbOpt *DatabaseOptimizer, connPool *ConnectionPoolManager) *PerformanceMonitor {
	return &PerformanceMonitor{
		cacheManager: cacheMgr,
		dbOptimizer:  dbOpt,
		connPool:     connPool,
		startTime:    time.Now(),
	}
}

// GetSystemMetrics returns comprehensive system metrics
func (pm *PerformanceMonitor) GetSystemMetrics() map[string]interface{} {
	cacheStats := pm.cacheManager.GetStats()
	dbStats := pm.dbOptimizer.GetQueryStats()
	connStats := pm.connPool.GetStats()

	slowQueries := pm.dbOptimizer.GetSlowQueries(time.Second, 5)
	indexSuggestions := pm.dbOptimizer.SuggestIndexes()

	return map[string]interface{}{
		"uptime":            time.Since(pm.startTime).String(),
		"cache_stats":       cacheStats,
		"database_stats":    dbStats,
		"connection_stats":  connStats,
		"slow_queries":      slowQueries,
		"index_suggestions": indexSuggestions,
		"performance_score": pm.calculatePerformanceScore(),
	}
}

// calculatePerformanceScore calculates an overall performance score
func (pm *PerformanceMonitor) calculatePerformanceScore() float64 {
	// Simple scoring algorithm
	score := 100.0

	// Penalize for slow queries
	slowQueries := pm.dbOptimizer.GetSlowQueries(time.Second, 1)
	score -= float64(len(slowQueries)) * 5

	// Penalize for high connection usage
	connStats := pm.connPool.GetStats()
	if current, ok := connStats["current_connections"].(int); ok {
		if max, ok := connStats["max_connections"].(int); ok {
			usage := float64(current) / float64(max)
			if usage > 0.8 {
				score -= 10
			}
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
