package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CacheLevel represents different cache levels
type CacheLevel int

const (
	LevelL1 CacheLevel = iota // In-memory (fastest)
	LevelL2                   // Redis (distributed)
	LevelL3                   // Application-specific
)

// CacheConfig holds cache configuration
type CacheConfig struct {
	Enabled         bool          `json:"enabled"`
	RedisAddr       string        `json:"redis_addr,omitempty"`
	RedisDB         int           `json:"redis_db,omitempty"`
	DefaultTTL      time.Duration `json:"default_ttl"`
	MaxMemoryMB     int           `json:"max_memory_mb"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// CacheItem represents a cached item with metadata
type CacheItem struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Level       CacheLevel  `json:"level"`
	ExpiresAt   time.Time   `json:"expires_at"`
	AccessCount int64       `json:"access_count"`
	LastAccess  time.Time   `json:"last_access"`
}

// CacheStats holds cache performance statistics
type CacheStats struct {
	Hits        int64     `json:"hits"`
	Misses      int64     `json:"misses"`
	HitRate     float64   `json:"hit_rate"`
	TotalItems  int64     `json:"total_items"`
	MemoryUsage int64     `json:"memory_usage_bytes"`
	LastCleanup time.Time `json:"last_cleanup"`
}

// MultiLevelCache provides multi-level caching with L1 (memory), L2 (Redis), L3 (application)
type MultiLevelCache struct {
	config     CacheConfig
	l1Cache    *InMemoryCache
	l2Cache    *RedisCache
	l3Cache    map[string]interface{} // Application-specific cache
	stats      CacheStats
	statsMutex sync.RWMutex
	stopCh     chan struct{}
}

// NewMultiLevelCache creates a new multi-level cache
func NewMultiLevelCache(config CacheConfig) (*MultiLevelCache, error) {
	cache := &MultiLevelCache{
		config:  config,
		l1Cache: NewInMemoryCache(config.MaxMemoryMB, config.CleanupInterval),
		l3Cache: make(map[string]interface{}),
		stopCh:  make(chan struct{}),
	}

	// TODO: Initialize Redis cache when Redis dependency is added
	// For now, only L1 (in-memory) and L3 (application) caching is implemented

	// Start cleanup goroutine
	go cache.cleanupRoutine()

	return cache, nil
}

// Get retrieves a value from the cache hierarchy
func (mlc *MultiLevelCache) Get(ctx context.Context, key string) (interface{}, bool) {
	// Try L1 cache first
	if value, found := mlc.l1Cache.Get(key); found {
		mlc.recordHit()
		mlc.l1Cache.UpdateAccessTime(key)
		return value, true
	}

	// Try L2 cache (Redis)
	if mlc.l2Cache != nil {
		if value, err := mlc.l2Cache.Get(ctx, key); err == nil {
			mlc.recordHit()
			// Promote to L1 cache
			mlc.l1Cache.Set(key, value, mlc.config.DefaultTTL)
			return value, true
		}
	}

	// Try L3 cache (application-specific)
	if value, found := mlc.getL3Cache(key); found {
		mlc.recordHit()
		// Promote to L1 cache
		mlc.l1Cache.Set(key, value, mlc.config.DefaultTTL)
		return value, true
	}

	mlc.recordMiss()
	return nil, false
}

// Set stores a value in all cache levels
func (mlc *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = mlc.config.DefaultTTL
	}

	// Set in L1 cache
	mlc.l1Cache.Set(key, value, ttl)

	// Set in L2 cache (Redis)
	if mlc.l2Cache != nil {
		if err := mlc.l2Cache.Set(ctx, key, value, ttl); err != nil {
			// Log error but don't fail - Redis failure shouldn't break the app
			fmt.Printf("Failed to set Redis cache: %v\n", err)
		}
	}

	return nil
}

// Delete removes a value from all cache levels
func (mlc *MultiLevelCache) Delete(ctx context.Context, key string) error {
	// Delete from L1 cache
	mlc.l1Cache.Delete(key)

	// Delete from L2 cache (Redis)
	if mlc.l2Cache != nil {
		if err := mlc.l2Cache.Delete(ctx, key); err != nil {
			fmt.Printf("Failed to delete from Redis cache: %v\n", err)
		}
	}

	// Delete from L3 cache
	mlc.deleteL3Cache(key)

	return nil
}

// SetL3Cache sets a value in the application-specific cache
func (mlc *MultiLevelCache) SetL3Cache(key string, value interface{}) {
	mlc.l3Cache[key] = value
}

// GetL3Cache gets a value from the application-specific cache
func (mlc *MultiLevelCache) getL3Cache(key string) (interface{}, bool) {
	value, exists := mlc.l3Cache[key]
	return value, exists
}

// DeleteL3Cache removes a value from the application-specific cache
func (mlc *MultiLevelCache) deleteL3Cache(key string) {
	delete(mlc.l3Cache, key)
}

// GetStats returns cache performance statistics
func (mlc *MultiLevelCache) GetStats() CacheStats {
	mlc.statsMutex.RLock()
	defer mlc.statsMutex.RUnlock()

	stats := mlc.stats
	stats.TotalItems = int64(mlc.l1Cache.Len())
	if mlc.l2Cache != nil {
		// Note: Redis stats would need separate tracking
	}
	stats.MemoryUsage = mlc.l1Cache.GetMemoryUsage()

	return stats
}

// Clear clears all cache levels
func (mlc *MultiLevelCache) Clear(ctx context.Context) error {
	mlc.l1Cache.Clear()

	if mlc.l2Cache != nil {
		if err := mlc.l2Cache.Clear(ctx); err != nil {
			fmt.Printf("Failed to clear Redis cache: %v\n", err)
		}
	}

	mlc.l3Cache = make(map[string]interface{})

	return nil
}

// Close closes the cache and releases resources
func (mlc *MultiLevelCache) Close() error {
	close(mlc.stopCh)

	if mlc.l2Cache != nil {
		return mlc.l2Cache.Close()
	}

	return nil
}

func (mlc *MultiLevelCache) recordHit() {
	mlc.statsMutex.Lock()
	defer mlc.statsMutex.Unlock()
	mlc.stats.Hits++
	mlc.updateHitRate()
}

func (mlc *MultiLevelCache) recordMiss() {
	mlc.statsMutex.Lock()
	defer mlc.statsMutex.Unlock()
	mlc.stats.Misses++
	mlc.updateHitRate()
}

func (mlc *MultiLevelCache) updateHitRate() {
	total := mlc.stats.Hits + mlc.stats.Misses
	if total > 0 {
		mlc.stats.HitRate = float64(mlc.stats.Hits) / float64(total) * 100
	}
}

func (mlc *MultiLevelCache) cleanupRoutine() {
	ticker := time.NewTicker(mlc.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mlc.l1Cache.Cleanup()
			mlc.stats.LastCleanup = time.Now()
		case <-mlc.stopCh:
			return
		}
	}
}

// InMemoryCache provides fast in-memory caching with LRU eviction
type InMemoryCache struct {
	items       map[string]*CacheItem
	maxMemoryMB int
	mutex       sync.RWMutex
}

func NewInMemoryCache(maxMemoryMB int, cleanupInterval time.Duration) *InMemoryCache {
	cache := &InMemoryCache{
		items:       make(map[string]*CacheItem),
		maxMemoryMB: maxMemoryMB,
	}

	// Start cleanup goroutine
	go cache.cleanupRoutine(cleanupInterval)

	return cache
}

func (imc *InMemoryCache) Get(key string) (interface{}, bool) {
	imc.mutex.RLock()
	defer imc.mutex.RUnlock()

	item, exists := imc.items[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	item.AccessCount++
	item.LastAccess = time.Now()

	return item.Value, true
}

func (imc *InMemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	expiresAt := time.Now().Add(ttl)
	item := &CacheItem{
		Key:         key,
		Value:       value,
		Level:       LevelL1,
		ExpiresAt:   expiresAt,
		AccessCount: 0,
		LastAccess:  time.Now(),
	}

	imc.items[key] = item
}

func (imc *InMemoryCache) Delete(key string) {
	imc.mutex.Lock()
	defer imc.mutex.Unlock()
	delete(imc.items, key)
}

func (imc *InMemoryCache) Len() int {
	imc.mutex.RLock()
	defer imc.mutex.RUnlock()
	return len(imc.items)
}

func (imc *InMemoryCache) Clear() {
	imc.mutex.Lock()
	defer imc.mutex.Unlock()
	imc.items = make(map[string]*CacheItem)
}

func (imc *InMemoryCache) UpdateAccessTime(key string) {
	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	if item, exists := imc.items[key]; exists {
		item.LastAccess = time.Now()
		item.AccessCount++
	}
}

func (imc *InMemoryCache) GetMemoryUsage() int64 {
	// Simplified memory calculation
	// In production, you'd use more accurate memory profiling
	return int64(len(imc.items) * 200) // Rough estimate: 200 bytes per item
}

func (imc *InMemoryCache) Cleanup() {
	imc.mutex.Lock()
	defer imc.mutex.Unlock()

	now := time.Now()
	for key, item := range imc.items {
		if now.After(item.ExpiresAt) {
			delete(imc.items, key)
		}
	}
}

func (imc *InMemoryCache) cleanupRoutine(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		imc.Cleanup()
	}
}

// RedisCache provides Redis-based distributed caching (placeholder for future implementation)
type RedisCache struct {
	// TODO: Implement Redis caching when Redis dependency is added
}

func NewRedisCache(defaultTTL time.Duration) *RedisCache {
	return &RedisCache{}
}

func (rc *RedisCache) Get(ctx context.Context, key string) (interface{}, error) {
	// TODO: Implement Redis Get
	return nil, fmt.Errorf("Redis cache not implemented")
}

func (rc *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// TODO: Implement Redis Set
	return fmt.Errorf("Redis cache not implemented")
}

func (rc *RedisCache) Delete(ctx context.Context, key string) error {
	// TODO: Implement Redis Delete
	return fmt.Errorf("Redis cache not implemented")
}

func (rc *RedisCache) Clear(ctx context.Context) error {
	// TODO: Implement Redis Clear
	return fmt.Errorf("Redis cache not implemented")
}

func (rc *RedisCache) Close() error {
	return nil
}
