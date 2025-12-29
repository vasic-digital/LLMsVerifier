package performance

import (
	"fmt"
	"runtime"
	"time"
)

// SystemOptimizer provides basic system optimization utilities
type SystemOptimizer struct {
	lastOptimization  time.Time
	optimizationCount int
}

// NewSystemOptimizer creates a new system optimizer
func NewSystemOptimizer() *SystemOptimizer {
	return &SystemOptimizer{}
}

// OptimizeMemory performs memory optimization
func (so *SystemOptimizer) OptimizeMemory() {
	before := so.getMemoryUsage()

	// Force garbage collection
	runtime.GC()
	runtime.GC() // Run twice for better cleanup

	after := so.getMemoryUsage()

	so.optimizationCount++
	so.lastOptimization = time.Now()

	fmt.Printf("Memory optimization: %d -> %d bytes (%.1f%% reduction)\n",
		before, after, float64(before-after)/float64(before)*100)
}

// getMemoryUsage returns current memory usage
func (so *SystemOptimizer) getMemoryUsage() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

// GetOptimizationStats returns optimization statistics
func (so *SystemOptimizer) GetOptimizationStats() map[string]interface{} {
	return map[string]interface{}{
		"last_optimization":  so.lastOptimization,
		"optimization_count": so.optimizationCount,
		"memory_usage":       so.getMemoryUsage(),
		"goroutines":         runtime.NumGoroutine(),
	}
}
