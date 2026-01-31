package observe

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

// Metrics holds all gateway Prometheus metrics.
type Metrics struct {
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	BackendHealthy   *prometheus.GaugeVec
	RateLimitedTotal *prometheus.CounterVec
	CircuitState     *prometheus.GaugeVec
	ActiveConns      *prometheus.GaugeVec
}

// NewMetrics creates and registers all gateway metrics.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_requests_total",
				Help: "Total number of requests processed.",
			},
			[]string{"service", "status", "method"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "gateway_request_duration_seconds",
				Help: "Request duration in seconds.",
				// Buckets: 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s
				Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"service"},
		),
		BackendHealthy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_backend_healthy",
				Help: "Whether a backend is healthy (1) or not (0).",
			},
			[]string{"backend"},
		),
		RateLimitedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gateway_rate_limited_total",
				Help: "Total number of rate-limited requests.",
			},
			[]string{"client"},
		),
		CircuitState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_circuit_state",
				Help: "Circuit breaker state: 0=closed, 1=open, 2=half-open.",
			},
			[]string{"backend"},
		),
		ActiveConns: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gateway_active_connections",
				Help: "Number of active connections per backend.",
			},
			[]string{"backend"},
		),
	}

	reg.MustRegister(
		m.RequestsTotal,
		m.RequestDuration,
		m.BackendHealthy,
		m.RateLimitedTotal,
		m.CircuitState,
		m.ActiveConns,
	)

	return m
}

// Handler returns the HTTP handler for the /metrics endpoint.
func Handler() http.Handler {
	return promhttp.Handler()
}
