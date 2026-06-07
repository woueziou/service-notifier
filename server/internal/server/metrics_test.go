package server

import (
	"math"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// readMetricGauge extracts the gauge value from any prometheus Metric.
func readMetricGauge(m prometheus.Metric) float64 {
	var metric dto.Metric
	if err := m.Write(&metric); err != nil {
		return math.NaN()
	}
	if metric.Gauge == nil {
		return math.NaN()
	}
	return metric.Gauge.GetValue()
}

func TestMetricsCollector_New(t *testing.T) {
	m := NewMetricsCollector()
	if m.RequestCount == nil {
		t.Error("RequestCount should be initialized")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration should be initialized")
	}
	if m.RequestsInFlight == nil {
		t.Error("RequestsInFlight should be initialized")
	}
}

func TestSetQueueDepth_GaugeReturnsValue(t *testing.T) {
	m := NewMetricsCollector()
	m.SetQueueDepth(func() float64 {
		return 42.0
	})

	val := readMetricGauge(m.QueueDepth)
	if val != 42.0 {
		t.Errorf("expected 42, got %f", val)
	}
}

func TestSetQueueDepth_ZeroOnEmpty(t *testing.T) {
	m := NewMetricsCollector()
	m.SetQueueDepth(func() float64 {
		return 0
	})

	val := readMetricGauge(m.QueueDepth)
	if val != 0 {
		t.Errorf("expected 0, got %f", val)
	}
}

func TestSetDLQDepth_GaugeReturnsValue(t *testing.T) {
	m := NewMetricsCollector()
	m.SetDLQDepth(func() float64 {
		return 7.0
	})

	val := readMetricGauge(m.DLQDepth)
	if val != 7.0 {
		t.Errorf("expected 7, got %f", val)
	}
}

func TestSetDLQDepth_ZeroOnEmpty(t *testing.T) {
	m := NewMetricsCollector()
	m.SetDLQDepth(func() float64 {
		return 0
	})

	val := readMetricGauge(m.DLQDepth)
	if val != 0 {
		t.Errorf("expected 0, got %f", val)
	}
}

func TestSetQueueDepth_CalledOnEachRead(t *testing.T) {
	m := NewMetricsCollector()
	counter := 0

	m.SetQueueDepth(func() float64 {
		counter++
		return float64(counter)
	})

	v1 := readMetricGauge(m.QueueDepth)
	v2 := readMetricGauge(m.QueueDepth)
	v3 := readMetricGauge(m.QueueDepth)

	if v1 != 1 || v2 != 2 || v3 != 3 {
		t.Errorf("expected sequential reads 1,2,3 got %f,%f,%f", v1, v2, v3)
	}
}

func TestRequestsInFlight_IncrementsAndDecrements(t *testing.T) {
	m := NewMetricsCollector()

	before := readMetricGauge(m.RequestsInFlight)
	m.RequestsInFlight.Inc()
	middle := readMetricGauge(m.RequestsInFlight)
	m.RequestsInFlight.Dec()
	after := readMetricGauge(m.RequestsInFlight)

	if middle-before != 1 {
		t.Errorf("inc should increase by 1: before=%f middle=%f", before, middle)
	}
	if after != before {
		t.Errorf("dec should return to original: before=%f after=%f", before, after)
	}
}

func TestMetricsMiddleware_ReturnsHandler(t *testing.T) {
	m := NewMetricsCollector()
	handler := MetricsMiddleware(m)
	if handler == nil {
		t.Error("MetricsMiddleware should return a handler")
	}
}

func TestPrometheusRegistration_NoPanic(t *testing.T) {
	// Verify that NewMetricsCollector doesn't panic when registering.
	// We run in a subtest so panics don't cascade.
	t.Run("register", func(t *testing.T) {
		_ = NewMetricsCollector()
	})
}
