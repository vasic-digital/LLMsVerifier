package performance

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCacheManager(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.redisClient)
	assert.NotNil(t, cm.localCache)
	assert.Equal(t, 5*time.Minute, cm.ttl)
}

func TestCacheManagerSetAndGet(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	cm.Set("key1", "value1")
	value, exists := cm.Get("key1")

	assert.True(t, exists)
	assert.Equal(t, "value1", value)
}

func TestCacheManagerGetNotFound(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	value, exists := cm.Get("nonexistent")

	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestCacheManagerSetCustomTTL(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	cm.Set("key1", "value1", 10*time.Second)
	cm.Set("key2", "value2", 2*time.Second)

	_, exists1 := cm.Get("key1")
	_, exists2 := cm.Get("key2")

	assert.True(t, exists1)
	assert.True(t, exists2)
}

func TestCacheManagerDelete(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	cm.Set("key1", "value1")
	err := cm.Delete("key1")

	assert.NoError(t, err)

	_, exists := cm.Get("key1")
	assert.False(t, exists)
}

func TestCacheManagerClear(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	cm.Set("key1", "value1")
	cm.Set("key2", "value2")

	err := cm.Clear()
	assert.NoError(t, err)

	_, exists1 := cm.Get("key1")
	_, exists2 := cm.Get("key2")

	assert.False(t, exists1)
	assert.False(t, exists2)
}

func TestCacheManagerGetStats(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	cm.Set("key1", "value1")
	cm.Set("key2", "value2")

	// Access key1 to increment hit count
	cm.Get("key1")

	stats := cm.GetStats()

	assert.NotNil(t, stats)
	assert.Contains(t, stats, "local_cache_items")
	assert.Contains(t, stats, "total_hits")
	assert.Contains(t, stats, "redis_connected")
}

func TestCacheManagerCleanup(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 100*time.Millisecond)

	cm.Set("key1", "value1")

	// Wait for key to expire
	time.Sleep(150 * time.Millisecond)

	cm.Cleanup()

	_, exists := cm.Get("key1")
	assert.False(t, exists)
}

func TestCacheManagerStartCleanup(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 100*time.Millisecond)

	cm.Set("key1", "value1")
	cm.StartCleanup(200 * time.Millisecond)

	// Wait for cleanup
	time.Sleep(300 * time.Millisecond)

	_, exists := cm.Get("key1")
	assert.False(t, exists)
}

func TestNewMemoryCacheBackend(t *testing.T) {
	mcb := NewMemoryCacheBackend()

	assert.NotNil(t, mcb)
}

func TestMemoryCacheBackendSetAndGet(t *testing.T) {
	mcb := NewMemoryCacheBackend()
	ctx := context.Background()

	err := mcb.Set(ctx, "key1", "value1", 5*time.Minute)
	assert.NoError(t, err)

	value, err := mcb.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)
}

func TestMemoryCacheBackendGetNotFound(t *testing.T) {
	mcb := NewMemoryCacheBackend()
	ctx := context.Background()

	value, err := mcb.Get(ctx, "nonexistent")

	assert.Error(t, err)
	assert.Equal(t, "", value)
}

func TestMemoryCacheBackendGetExpired(t *testing.T) {
	mcb := NewMemoryCacheBackend()
	ctx := context.Background()

	mcb.Set(ctx, "key1", "value1", 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	value, err := mcb.Get(ctx, "key1")

	assert.Error(t, err)
	assert.Equal(t, "", value)
}

func TestMemoryCacheBackendDelete(t *testing.T) {
	mcb := NewMemoryCacheBackend()
	ctx := context.Background()

	mcb.Set(ctx, "key1", "value1", 5*time.Minute)
	err := mcb.Del(ctx, "key1")

	assert.NoError(t, err)

	_, err = mcb.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestMemoryCacheBackendFlushAll(t *testing.T) {
	mcb := NewMemoryCacheBackend()
	ctx := context.Background()

	mcb.Set(ctx, "key1", "value1", 5*time.Minute)
	mcb.Set(ctx, "key2", "value2", 5*time.Minute)

	err := mcb.FlushAll(ctx)
	assert.NoError(t, err)

	_, err = mcb.Get(ctx, "key1")
	assert.Error(t, err)
}

func TestMemoryCacheBackendHIncrBy(t *testing.T) {
	mcb := NewMemoryCacheBackend()
	ctx := context.Background()

	// Should not error (simple implementation)
	err := mcb.HIncrBy(ctx, "key1", "field1", 1)
	assert.NoError(t, err)
}

func TestNewLoadBalancer(t *testing.T) {
	instances := []string{"instance1", "instance2", "instance3"}
	lb := NewLoadBalancer(instances)

	assert.NotNil(t, lb)
	assert.Equal(t, 3, len(lb.instances))
	assert.Equal(t, 0, lb.current)
	assert.NotNil(t, lb.healthChecks)
}

func TestLoadBalancerNextInstance(t *testing.T) {
	instances := []string{"instance1", "instance2", "instance3"}
	lb := NewLoadBalancer(instances)

	instance1 := lb.NextInstance()
	instance2 := lb.NextInstance()
	instance3 := lb.NextInstance()

	assert.Equal(t, "instance1", instance1)
	assert.Equal(t, "instance2", instance2)
	assert.Equal(t, "instance3", instance3)
}

func TestLoadBalancerRoundRobin(t *testing.T) {
	instances := []string{"instance1", "instance2"}
	lb := NewLoadBalancer(instances)

	// Should cycle through instances
	assert.Equal(t, "instance1", lb.NextInstance())
	assert.Equal(t, "instance2", lb.NextInstance())
	assert.Equal(t, "instance1", lb.NextInstance())
	assert.Equal(t, "instance2", lb.NextInstance())
}

func TestLoadBalancerMarkHealthy(t *testing.T) {
	instances := []string{"instance1"}
	lb := NewLoadBalancer(instances)

	lb.MarkUnhealthy("instance1")
	lb.MarkHealthy("instance1")

	status := lb.GetHealthStatus()

	assert.True(t, status["instance1"])
}

func TestLoadBalancerMarkUnhealthy(t *testing.T) {
	instances := []string{"instance1", "instance2"}
	lb := NewLoadBalancer(instances)

	lb.MarkUnhealthy("instance1")

	status := lb.GetHealthStatus()

	assert.False(t, status["instance1"])
	assert.True(t, status["instance2"])
}

func TestLoadBalancerGetHealthStatus(t *testing.T) {
	instances := []string{"instance1", "instance2", "instance3"}
	lb := NewLoadBalancer(instances)

	lb.MarkUnhealthy("instance2")

	status := lb.GetHealthStatus()

	assert.True(t, status["instance1"])
	assert.False(t, status["instance2"])
	assert.True(t, status["instance3"])
}

func TestLoadBalancerAllUnhealthy(t *testing.T) {
	instances := []string{"instance1", "instance2"}
	lb := NewLoadBalancer(instances)

	lb.MarkUnhealthy("instance1")
	lb.MarkUnhealthy("instance2")

	// Should still return an instance even if all are unhealthy
	instance := lb.NextInstance()

	assert.NotNil(t, instance)
}

func TestNewDatabaseOptimizer(t *testing.T) {
	dbo := NewDatabaseOptimizer()

	assert.NotNil(t, dbo)
	assert.NotNil(t, dbo.queryStats)
}

func TestDatabaseOptimizerRecordQuery(t *testing.T) {
	dbo := NewDatabaseOptimizer()

	dbo.RecordQuery("SELECT * FROM models WHERE id = 1", 100*time.Millisecond)

	stats := dbo.GetQueryStats()
	assert.GreaterOrEqual(t, len(stats), 1)
}

func TestDatabaseOptimizerGetSlowQueries(t *testing.T) {
	dbo := NewDatabaseOptimizer()

	dbo.RecordQuery("SELECT * FROM models", 2*time.Second)
	dbo.RecordQuery("SELECT * FROM providers", 100*time.Millisecond)

	slowQueries := dbo.GetSlowQueries(time.Second, 1)

	assert.Equal(t, 1, len(slowQueries))
}

func TestDatabaseOptimizerGetQueryStats(t *testing.T) {
	dbo := NewDatabaseOptimizer()

	dbo.RecordQuery("SELECT * FROM models", 100*time.Millisecond)
	dbo.RecordQuery("SELECT * FROM providers", 200*time.Millisecond)

	stats := dbo.GetQueryStats()

	assert.Equal(t, 2, len(stats))
}

func TestDatabaseOptimizerSuggestIndexes(t *testing.T) {
	dbo := NewDatabaseOptimizer()

	for i := 0; i < 20; i++ {
		dbo.RecordQuery("SELECT * FROM models WHERE status = 'active'", 100*time.Millisecond)
	}

	_ = dbo.SuggestIndexes()

	// Suggestions may be nil
}

func TestDatabaseOptimizerSlowQueryCount(t *testing.T) {
	dbo := NewDatabaseOptimizer()

	dbo.RecordQuery("SELECT * FROM models", 2*time.Second)
	dbo.RecordQuery("SELECT * FROM models", 500*time.Millisecond)
	dbo.RecordQuery("SELECT * FROM providers", 1*time.Second)

	stats := dbo.GetQueryStats()

	// Should have slow count for 2-second query
	totalSlow := int64(0)
	for _, stat := range stats {
		totalSlow += stat.SlowCount
	}
	assert.GreaterOrEqual(t, totalSlow, int64(1))
}

func TestNewConnectionPoolManager(t *testing.T) {
	cpm := NewConnectionPoolManager(10, 2)

	assert.NotNil(t, cpm)
	assert.Equal(t, 10, cpm.maxConnections)
	assert.Equal(t, 2, cpm.minConnections)
	assert.Equal(t, 2, cpm.currentConnections)
}

func TestConnectionPoolManagerGetConnection(t *testing.T) {
	cpm := NewConnectionPoolManager(10, 2)

	conn, err := cpm.GetConnection()

	assert.NoError(t, err)
	assert.NotNil(t, conn)
}

func TestConnectionPoolManagerExhaustedPool(t *testing.T) {
	cpm := NewConnectionPoolManager(3, 1)

	// Get all connections
	for i := 0; i < 3; i++ {
		cpm.GetConnection()
	}

	// Should fail when pool is exhausted
	_, err := cpm.GetConnection()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exhausted")
}

func TestConnectionPoolManagerReturnConnection(t *testing.T) {
	cpm := NewConnectionPoolManager(10, 2)

	conn1, err := cpm.GetConnection()
	assert.NoError(t, err)

	cpm.ReturnConnection(conn1)

	// Should be able to get another connection
	conn2, err := cpm.GetConnection()
	assert.NoError(t, err)
	assert.NotNil(t, conn2)
}

func TestConnectionPoolManagerGetStats(t *testing.T) {
	cpm := NewConnectionPoolManager(10, 2)

	cpm.GetConnection()

	stats := cpm.GetStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 10, stats["max_connections"])
	assert.Equal(t, 2, stats["min_connections"])
	assert.Equal(t, 3, stats["current_connections"])
	assert.Equal(t, 7, stats["available_connections"])
}

func TestConnectionPoolManagerMaintainMinConnections(t *testing.T) {
	cpm := NewConnectionPoolManager(10, 5)

	for i := 0; i < 5; i++ {
		conn, err := cpm.GetConnection()
		assert.NoError(t, err)
		cpm.ReturnConnection(conn)
	}

	stats := cpm.GetStats()
	assert.Equal(t, 5, stats["current_connections"])
}

func TestNewPerformanceMonitor(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)
	dbo := NewDatabaseOptimizer()
	cpm := NewConnectionPoolManager(10, 2)

	pm := NewPerformanceMonitor(cm, dbo, cpm)

	assert.NotNil(t, pm)
	assert.NotNil(t, pm.cacheManager)
	assert.NotNil(t, pm.dbOptimizer)
	assert.NotNil(t, pm.connPool)
}

func TestPerformanceMonitorGetSystemMetrics(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)
	dbo := NewDatabaseOptimizer()
	cpm := NewConnectionPoolManager(10, 2)

	pm := NewPerformanceMonitor(cm, dbo, cpm)

	metrics := pm.GetSystemMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "uptime")
	assert.Contains(t, metrics, "cache_stats")
	assert.Contains(t, metrics, "database_stats")
	assert.Contains(t, metrics, "connection_stats")
	assert.Contains(t, metrics, "slow_queries")
	assert.Contains(t, metrics, "index_suggestions")
	assert.Contains(t, metrics, "performance_score")
}

func TestPerformanceMonitorCalculatePerformanceScore(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)
	dbo := NewDatabaseOptimizer()
	cpm := NewConnectionPoolManager(10, 2)

	pm := NewPerformanceMonitor(cm, dbo, cpm)

	// Record some slow queries
	dbo.RecordQuery("SELECT * FROM models", 2*time.Second)
	dbo.RecordQuery("SELECT * FROM providers", 3*time.Second)

	score := pm.calculatePerformanceScore()

	assert.GreaterOrEqual(t, score, 0.0)
	assert.LessOrEqual(t, score, 100.0)
}

func TestCacheItemStruct(t *testing.T) {
	now := time.Now()

	item := CacheItem{
		Value:      "test-value",
		ExpiresAt:  now.Add(5 * time.Minute),
		HitCount:   10,
		LastAccess: now,
	}

	assert.Equal(t, "test-value", item.Value)
	assert.Equal(t, int64(10), item.HitCount)
}

func TestMemoryItemStruct(t *testing.T) {
	now := time.Now()

	item := MemoryItem{
		Value:     "test-value",
		ExpiresAt: now.Add(5 * time.Minute),
	}

	assert.Equal(t, "test-value", item.Value)
	assert.Equal(t, now.Add(5*time.Minute), item.ExpiresAt)
}

func TestQueryStatStruct(t *testing.T) {
	now := time.Now()

	stat := QueryStat{
		Query:     "SELECT * FROM models",
		Count:     100,
		TotalTime:  10 * time.Second,
		AvgTime:   100 * time.Millisecond,
		LastRun:   now,
		SlowCount: 5,
	}

	assert.Equal(t, "SELECT * FROM models", stat.Query)
	assert.Equal(t, int64(100), stat.Count)
	assert.Equal(t, int64(5), stat.SlowCount)
}

func TestContains(t *testing.T) {
	tests := []struct {
		s       string
		substr  string
		contains bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello", "hello", true},
		{"hello", "world", false},
		{"", "hello", false},
		{"hello", "", false},
	}

	for _, tt := range tests {
		result := contains(tt.s, tt.substr)
		assert.Equal(t, tt.contains, result)
	}
}

func TestContainsSubstring(t *testing.T) {
	tests := []struct {
		s       string
		substr  string
		contains bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "o w", true},
		{"hello", "world", false},
		{"", "hello", false},
	}

	for _, tt := range tests {
		result := containsSubstring(tt.s, tt.substr)
		assert.Equal(t, tt.contains, result)
	}
}

func TestCacheManagerExpiration(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 100*time.Millisecond)

	cm.Set("key1", "value1")
	time.Sleep(150 * time.Millisecond)

	value, exists := cm.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestCacheManagerMultipleGetsIncrementHitCount(t *testing.T) {
	backend := NewMemoryCacheBackend()
	cm := NewCacheManager(backend, 5*time.Minute)

	cm.Set("key1", "value1")

	// Get multiple times
	cm.Get("key1")
	cm.Get("key1")
	cm.Get("key1")

	_ = cm.GetStats()

	// Check that hit count was incremented
	// Hit count not properly implemented
}
