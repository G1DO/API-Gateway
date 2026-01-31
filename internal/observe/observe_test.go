package observe

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// --- Metrics ---

func TestMetricsRegistration(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Verify all metrics are registered by using them
	m.RequestsTotal.WithLabelValues("users", "200", "GET").Inc()
	m.RequestDuration.WithLabelValues("users").Observe(0.05)
	m.BackendHealthy.WithLabelValues("http://A:8080").Set(1)
	m.RateLimitedTotal.WithLabelValues("192.168.1.1").Inc()
	m.CircuitState.WithLabelValues("http://A:8080").Set(0)
	m.ActiveConns.WithLabelValues("http://A:8080").Set(5)

	// Check counter value
	expected := `
# HELP gateway_requests_total Total number of requests processed.
# TYPE gateway_requests_total counter
gateway_requests_total{method="GET",service="users",status="200"} 1
`
	if err := testutil.CollectAndCompare(m.RequestsTotal, strings.NewReader(expected)); err != nil {
		t.Fatalf("metrics mismatch: %v", err)
	}
}

func TestMetricsHistogramBuckets(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	// Record some latencies
	m.RequestDuration.WithLabelValues("api").Observe(0.001)  // 1ms
	m.RequestDuration.WithLabelValues("api").Observe(0.05)   // 50ms
	m.RequestDuration.WithLabelValues("api").Observe(0.5)    // 500ms
	m.RequestDuration.WithLabelValues("api").Observe(2.0)    // 2s

	// Histogram should have recorded 4 observations
	count := testutil.ToFloat64(m.RequestDuration.WithLabelValues("api"))
	if count != 4 {
		t.Fatalf("expected 4 observations, got %.0f", count)
	}
}

func TestMetricsGaugeUpDown(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.ActiveConns.WithLabelValues("http://A:8080").Set(10)
	val := testutil.ToFloat64(m.ActiveConns.WithLabelValues("http://A:8080"))
	if val != 10 {
		t.Fatalf("expected 10, got %.0f", val)
	}

	m.ActiveConns.WithLabelValues("http://A:8080").Set(3)
	val = testutil.ToFloat64(m.ActiveConns.WithLabelValues("http://A:8080"))
	if val != 3 {
		t.Fatalf("expected 3 after decrease, got %.0f", val)
	}
}

// --- Structured Logging ---

func TestNewLoggerOutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	logger.Info("test message", "key", "value")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log output is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if entry["msg"] != "test message" {
		t.Fatalf("expected msg 'test message', got %v", entry["msg"])
	}
	if entry["key"] != "value" {
		t.Fatalf("expected key 'value', got %v", entry["key"])
	}
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	logger.Info("should be filtered")
	if buf.Len() > 0 {
		t.Fatal("info message should be filtered at warn level")
	}

	logger.Warn("should appear")
	if buf.Len() == 0 {
		t.Fatal("warn message should appear at warn level")
	}
}

func TestRequestLoggerAttachesFields(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil))

	reqLogger := RequestLogger(base, "POST", "/api/users", "192.168.1.1", "trace-abc")
	reqLogger.Info("request completed", "status", 200)

	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)

	if entry["method"] != "POST" {
		t.Errorf("expected method POST, got %v", entry["method"])
	}
	if entry["path"] != "/api/users" {
		t.Errorf("expected path /api/users, got %v", entry["path"])
	}
	if entry["client_ip"] != "192.168.1.1" {
		t.Errorf("expected client_ip 192.168.1.1, got %v", entry["client_ip"])
	}
	if entry["trace_id"] != "trace-abc" {
		t.Errorf("expected trace_id trace-abc, got %v", entry["trace_id"])
	}
}

func TestLoggerContext(t *testing.T) {
	logger := slog.Default()
	ctx := WithLogger(context.Background(), logger)

	got := LoggerFrom(ctx)
	if got != logger {
		t.Fatal("should retrieve same logger from context")
	}
}

func TestLoggerContextFallback(t *testing.T) {
	// No logger in context → should return default
	got := LoggerFrom(context.Background())
	if got == nil {
		t.Fatal("should return default logger when none in context")
	}
}

// --- Request Tracing ---

func TestGenerateTraceIDUnique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := GenerateTraceID()
		if ids[id] {
			t.Fatalf("duplicate trace ID: %s", id)
		}
		ids[id] = true
	}
}

func TestGenerateTraceIDLength(t *testing.T) {
	id := GenerateTraceID()
	// 16 bytes = 32 hex characters
	if len(id) != 32 {
		t.Fatalf("expected 32 char hex string, got %d chars: %s", len(id), id)
	}
}

func TestTraceIDFromRequestReusesExisting(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(TraceHeader, "existing-trace-id")

	got := TraceIDFromRequest(req)
	if got != "existing-trace-id" {
		t.Fatalf("expected existing-trace-id, got %s", got)
	}
}

func TestTraceIDFromRequestGeneratesNew(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	got := TraceIDFromRequest(req)
	if got == "" {
		t.Fatal("should generate a trace ID")
	}
	if len(got) != 32 {
		t.Fatalf("expected 32 char hex string, got %s", got)
	}
}

func TestTraceIDContext(t *testing.T) {
	ctx := WithTraceID(context.Background(), "my-trace")
	got := TraceIDFrom(ctx)
	if got != "my-trace" {
		t.Fatalf("expected my-trace, got %s", got)
	}
}

func TestTracingMiddleware(t *testing.T) {
	var gotTraceID string

	handler := TracingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = TraceIDFrom(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// Test 1: no existing trace ID → generates one
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotTraceID == "" {
		t.Fatal("middleware should set trace ID in context")
	}
	if rec.Header().Get(TraceHeader) == "" {
		t.Fatal("middleware should set trace ID in response header")
	}
	if rec.Header().Get(TraceHeader) != gotTraceID {
		t.Fatal("response header and context trace ID should match")
	}

	// Test 2: existing trace ID → reuses it
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set(TraceHeader, "client-trace-123")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	if gotTraceID != "client-trace-123" {
		t.Fatalf("should reuse client trace ID, got %s", gotTraceID)
	}
	if rec2.Header().Get(TraceHeader) != "client-trace-123" {
		t.Fatal("response should contain client trace ID")
	}
}
