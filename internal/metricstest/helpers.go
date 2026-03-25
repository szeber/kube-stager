// Package metricstest provides test helpers for asserting on Prometheus metric values.
// This package is intentionally separate from internal/metrics to avoid shipping
// prometheus/testutil in production binaries.
package metricstest

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// GetCounterValue returns the current value of a counter metric with the given labels.
// Returns 0 if the metric has not been observed yet.
func GetCounterValue(counter *prometheus.CounterVec, labels ...string) float64 {
	m, err := counter.GetMetricWithLabelValues(labels...)
	if err != nil {
		return 0
	}
	return testutil.ToFloat64(m)
}
