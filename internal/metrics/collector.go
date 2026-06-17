// Package metrics provides operational counters backed by Prometheus (architecture §22).
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Collector stores order counters and processing duration in a private registry.
type Collector struct {
	reg *prometheus.Registry

	ordersCreated   prometheus.Counter
	ordersProcessed prometheus.Counter
	ordersFailed    prometheus.Counter
	duration        prometheus.Histogram
}

// NewCollector builds an empty metrics collector with a dedicated registry.
func NewCollector() *Collector {
	reg := prometheus.NewRegistry()
	c := &Collector{reg: reg}

	c.ordersCreated = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orders_created_total",
		Help: "Total number of orders created.",
	})
	c.ordersProcessed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orders_processed_total",
		Help: "Total number of orders processed successfully.",
	})
	c.ordersFailed = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "orders_failed_total",
		Help: "Total number of orders that failed processing.",
	})
	c.duration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "orders_processing_duration_seconds",
		Help:    "Order processing duration in seconds.",
		Buckets: prometheus.DefBuckets,
	})

	reg.MustRegister(c.ordersCreated, c.ordersProcessed, c.ordersFailed, c.duration)
	return c
}

// IncOrdersCreated increments orders_created_total.
func (c *Collector) IncOrdersCreated() {
	c.ordersCreated.Inc()
}

// IncOrdersProcessed increments orders_processed_total.
func (c *Collector) IncOrdersProcessed() {
	c.ordersProcessed.Inc()
}

// IncOrdersFailed increments orders_failed_total.
func (c *Collector) IncOrdersFailed() {
	c.ordersFailed.Inc()
}

// RecordProcessingDuration observes a processing duration sample in seconds.
func (c *Collector) RecordProcessingDuration(seconds float64) {
	c.duration.Observe(seconds)
}

// Handler returns the Prometheus exposition handler for this collector's registry.
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(c.reg, promhttp.HandlerOpts{})
}
