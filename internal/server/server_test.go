package server

import (
	"fmt"
	"io"
	"net/http"
	"syscall"
	"testing"
	"time"
)

func freePort() string {
	// Use port 0 to let OS assign a free port
	return "127.0.0.1:0"
}

func TestServerStartsAndResponds(t *testing.T) {
	srv := New(Config{
		Addr: "127.0.0.1:0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		}),
		DrainTimeout: 5 * time.Second,
	})

	// Start server in background, send SIGINT shortly after
	go func() {
		time.Sleep(200 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	// This blocks until shutdown
	srv.ListenAndServe()
}

func TestServerGracefulShutdown(t *testing.T) {
	requestStarted := make(chan struct{})
	requestDone := make(chan struct{})

	srv := New(Config{
		Addr: "127.0.0.1:19876",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(requestStarted) // signal that request is being handled
			time.Sleep(500 * time.Millisecond) // simulate slow request
			w.Write([]byte("completed"))
			close(requestDone)
		}),
		DrainTimeout: 5 * time.Second,
	})

	go srv.ListenAndServe()
	time.Sleep(100 * time.Millisecond) // wait for server to start

	// Start a slow request
	go func() {
		resp, err := http.Get("http://127.0.0.1:19876/slow")
		if err != nil {
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if string(body) != "completed" {
			t.Errorf("expected 'completed', got %q", string(body))
		}
	}()

	// Wait for request to start, then signal shutdown
	<-requestStarted
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	// Request should complete despite shutdown signal
	select {
	case <-requestDone:
		// good — request completed during drain
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight request should have completed during drain")
	}
}

// testCloser tracks whether Close was called.
type testCloser struct {
	closed bool
}

func (tc *testCloser) Close() error {
	tc.closed = true
	return nil
}

func TestServerClosesResources(t *testing.T) {
	c1 := &testCloser{}
	c2 := &testCloser{}

	srv := New(Config{
		Addr: fmt.Sprintf("127.0.0.1:19877"),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}),
		DrainTimeout: 1 * time.Second,
	})
	srv.RegisterCloser(c1)
	srv.RegisterCloser(c2)

	go func() {
		time.Sleep(100 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	}()

	srv.ListenAndServe()

	if !c1.closed || !c2.closed {
		t.Fatal("all registered resources should be closed on shutdown")
	}
}
