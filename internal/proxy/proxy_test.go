package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeBalancer always returns the same address.
type fakeBalancer struct {
	addr string
}

func (f *fakeBalancer) Next() string { return f.addr }

func TestProxyForwardsRequestAndResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "ok")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello from backend"))
	}))
	defer backend.Close()

	p := NewProxy(&fakeBalancer{addr: backend.URL})
	frontend := httptest.NewServer(p)
	defer frontend.Close()

	resp, err := http.Get(frontend.URL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Backend") != "ok" {
		t.Fatal("backend response header not forwarded")
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello from backend" {
		t.Fatalf("expected 'hello from backend', got %q", string(body))
	}
}

func TestProxyForwardsPath(t *testing.T) {
	var gotPath string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p := NewProxy(&fakeBalancer{addr: backend.URL})
	frontend := httptest.NewServer(p)
	defer frontend.Close()

	http.Get(frontend.URL + "/api/v1/users")

	if gotPath != "/api/v1/users" {
		t.Fatalf("expected path /api/v1/users, got %q", gotPath)
	}
}

func TestProxyForwardsMethod(t *testing.T) {
	var gotMethod string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p := NewProxy(&fakeBalancer{addr: backend.URL})
	frontend := httptest.NewServer(p)
	defer frontend.Close()

	req, _ := http.NewRequest(http.MethodPost, frontend.URL+"/data", strings.NewReader("body"))
	http.DefaultClient.Do(req)

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
}

func TestProxyForwardsRequestBody(t *testing.T) {
	var gotBody string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p := NewProxy(&fakeBalancer{addr: backend.URL})
	frontend := httptest.NewServer(p)
	defer frontend.Close()

	req, _ := http.NewRequest(http.MethodPost, frontend.URL+"/", strings.NewReader("request payload"))
	http.DefaultClient.Do(req)

	if gotBody != "request payload" {
		t.Fatalf("expected 'request payload', got %q", gotBody)
	}
}

func TestProxyStripsHopByHopHeaders(t *testing.T) {
	hopByHop := []string{
		"Connection", "Keep-Alive", "Proxy-Authenticate",
		"Proxy-Authorization", "Te", "Trailers",
		"Transfer-Encoding", "Upgrade",
	}

	var gotHeaders http.Header
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	p := NewProxy(&fakeBalancer{addr: backend.URL})
	frontend := httptest.NewServer(p)
	defer frontend.Close()

	req, _ := http.NewRequest(http.MethodGet, frontend.URL+"/", nil)
	for _, h := range hopByHop {
		req.Header.Set(h, "should-be-stripped")
	}
	req.Header.Set("X-Custom", "should-pass")
	http.DefaultClient.Do(req)

	for _, h := range hopByHop {
		if gotHeaders.Get(h) == "should-be-stripped" {
			t.Errorf("hop-by-hop header %q was forwarded", h)
		}
	}
	if gotHeaders.Get("X-Custom") != "should-pass" {
		t.Error("custom header was not forwarded")
	}
}

func TestProxyReturns502WhenBackendDown(t *testing.T) {
	// Point at a backend that doesn't exist
	p := NewProxy(&fakeBalancer{addr: "http://127.0.0.1:1"})
	frontend := httptest.NewServer(p)
	defer frontend.Close()

	resp, err := http.Get(frontend.URL + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}
}

func TestProxyForwardsResponseHeaders(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Response-Id", "abc123")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	p := NewProxy(&fakeBalancer{addr: backend.URL})
	frontend := httptest.NewServer(p)
	defer frontend.Close()

	resp, _ := http.Get(frontend.URL + "/")
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Response-Id") != "abc123" {
		t.Fatal("response header X-Response-Id not forwarded")
	}
}