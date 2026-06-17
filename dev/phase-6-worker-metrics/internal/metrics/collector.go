// Package metrics provides concurrency-safe operational counters (architecture §22).
package metrics

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Collector stores in-memory counters for orders and processing duration.
type Collector struct {
	created   atomic.Int64
	processed atomic.Int64
	failed    atomic.Int64

	mu            sync.Mutex
	durationSum   float64
	durationCount int64
}

// NewCollector builds an empty metrics collector.
func NewCollector() *Collector {
	return &Collector{}
}

// IncOrdersCreated increments orders_created_total.
func (c *Collector) IncOrdersCreated() {
	c.created.Add(1)
}

// IncOrdersProcessed increments orders_processed_total.
func (c *Collector) IncOrdersProcessed() {
	c.processed.Add(1)
}

// IncOrdersFailed increments orders_failed_total.
func (c *Collector) IncOrdersFailed() {
	c.failed.Add(1)
}

// RecordProcessingDuration adds a processing duration sample in seconds.
func (c *Collector) RecordProcessingDuration(seconds float64) {
	c.mu.Lock()
	c.durationSum += seconds
	c.durationCount++
	c.mu.Unlock()
}

// Snapshot returns current counter values.
func (c *Collector) Snapshot() (created, processed, failed int64, sum float64, count int64) {
	c.mu.Lock()
	sum = c.durationSum
	count = c.durationCount
	c.mu.Unlock()
	return c.created.Load(), c.processed.Load(), c.failed.Load(), sum, count
}

// RenderText produces the architecture §22 plain-text metrics format.
func (c *Collector) RenderText() string {
	created, processed, failed, sum, count := c.Snapshot()
	avg := 0.0
	if count > 0 {
		avg = sum / float64(count)
	}
	return fmt.Sprintf(
		"orders_created_total %d\n"+
			"orders_processed_total %d\n"+
			"orders_failed_total %d\n"+
			"orders_processing_duration_seconds_sum %.2f\n"+
			"orders_processing_duration_seconds_count %d\n"+
			"orders_processing_duration_seconds_avg %.2f\n",
		created, processed, failed, sum, count, avg,
	)
}
