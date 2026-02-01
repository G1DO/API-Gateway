package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/G1D0/Api-Gateway/internal/circuitbreaker"
	"github.com/G1D0/Api-Gateway/internal/ratelimit"
)

// --- Chain ---

func TestChainOrder(t *testing.T) {
	var order []string

	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-before")
				next.ServeHTTP(w, r)
				order = append(order, name+"-after")
			})
		}
	}

	handler := Chain(mw("first"), mw("second"), mw("third"))(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "handler")
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	expected := []string{
		"first-before", "second-before", "third-before",
		"handler",
		"third-after", "second-after", "first-after",
	}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestChainEmpty(t *testing.T) {
	called := false
	handler := Chain()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler should be called with empty chain")
	}
}

// --- ResponseCapture ---

func TestResponseCaptureStatusCode(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := NewResponseCapture(rec)

	rc.WriteHeader(http.StatusNotFound)
	if rc.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", rc.StatusCode)
	}
}

func TestResponseCaptureDefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := NewResponseCapture(rec)

	// No WriteHeader called → default 200
	if rc.StatusCode != 200 {
		t.Fatalf("expected default 200, got %d", rc.StatusCode)
	}
}

func TestResponseCaptureWriteBytes(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := NewResponseCapture(rec)

	rc.Write([]byte("hello"))
	rc.Write([]byte(" world"))

	if rc.Written != 11 {
		t.Fatalf("expected 11 bytes, got %d", rc.Written)
	}
}

// --- Tracing ---

func TestTracingGeneratesID(t *testing.T) {
	var gotTraceID string
	handler := Tracing()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = TraceIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotTraceID == "" {
		t.Fatal("should generate trace ID")
	}
	if len(gotTraceID) != 32 {
		t.Fatalf("expected 32 char hex, got %d: %s", len(gotTraceID), gotTraceID)
	}
	if rec.Header().Get("X-Request-ID") != gotTraceID {
		t.Fatal("response header should match context trace ID")
	}
}

func TestTracingReusesExisting(t *testing.T) {
	var gotTraceID string
	handler := Tracing()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTraceID = TraceIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "client-trace-abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotTraceID != "client-trace-abc" {
		t.Fatalf("should reuse client trace ID, got %s", gotTraceID)
	}
}

// --- Logging ---

func TestLoggingOutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/users", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log is not valid JSON: %v\noutput: %s", err, buf.String())
	}
	if entry["method"] != "POST" {
		t.Errorf("expected POST, got %v", entry["method"])
	}
	if entry["path"] != "/api/users" {
		t.Errorf("expected /api/users, got %v", entry["path"])
	}
	// status is float64 in JSON
	if entry["status"] != float64(201) {
		t.Errorf("expected 201, got %v", entry["status"])
	}
}

// --- Rate Limit ---

func TestRateLimitAllows(t *testing.T) {
	limiter := ratelimit.NewPerClient(10, 10.0, 10*time.Minute)
	defer limiter.Close()

	handler := RateLimit(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRateLimitRejects(t *testing.T) {
	limiter := ratelimit.NewPerClient(2, 0, 10*time.Minute) // 2 tokens, no refill
	defer limiter.Close()

	handler := RateLimit(limiter)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust tokens
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Third should be rejected
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 429 {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
	if rec.Header().Get("Retry-After") == "" {
		t.Fatal("should set Retry-After header")
	}
}

// --- Circuit Breaker ---

func TestCircuitBreakerAllows(t *testing.T) {
	cb := circuitbreaker.NewPerBackend(3, 100*time.Millisecond)
	backendFunc := func(r *http.Request) string { return "backend-A" }

	handler := CircuitBreaker(cb, backendFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestCircuitBreakerRejectsWhenOpen(t *testing.T) {
	cb := circuitbreaker.NewPerBackend(2, 100*time.Millisecond)
	backendFunc := func(r *http.Request) string { return "backend-A" }

	// Return 500 to trigger failures
	handler := CircuitBreaker(cb, backendFunc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	// Trigger 2 failures to open circuit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	// Circuit should be open → 503
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 503 {
		t.Fatalf("expected 503 when circuit open, got %d", rec.Code)
	}
}

// --- Full Chain Integration ---

func TestFullChain(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	limiter := ratelimit.NewPerClient(100, 10.0, 10*time.Minute)
	defer limiter.Close()

	handler := Chain(
		Tracing(),
		Logging(logger),
		RateLimit(limiter),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify trace ID is available deep in the chain
		traceID := TraceIDFrom(r.Context())
		if traceID == "" {
			t.Fatal("trace ID should be available in handler")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("response should have trace ID")
	}

	// Verify log was written with all fields
	var entry map[string]interface{}
	json.Unmarshal(buf.Bytes(), &entry)
	if entry["method"] != "GET" {
		t.Error("log should contain method")
	}
	if entry["trace_id"] == nil || entry["trace_id"] == "" {
		t.Error("log should contain trace_id")
	}
}
