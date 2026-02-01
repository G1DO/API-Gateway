package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server wraps http.Server with graceful shutdown support.
type Server struct {
	httpServer   *http.Server
	drainTimeout time.Duration
	logger       *slog.Logger
	closers      []io.Closer // background resources to close on shutdown
}

// Config holds server configuration.
type Config struct {
	Addr         string        // listen address, e.g., ":9000"
	Handler      http.Handler
	DrainTimeout time.Duration // max time to wait for in-flight requests
	Logger       *slog.Logger
}

// New creates a server with graceful shutdown support.
func New(cfg Config) *Server {
	if cfg.DrainTimeout == 0 {
		cfg.DrainTimeout = 30 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	return &Server{
		httpServer: &http.Server{
			Addr:    cfg.Addr,
			Handler: cfg.Handler,
		},
		drainTimeout: cfg.DrainTimeout,
		logger:       cfg.Logger,
	}
}

// RegisterCloser adds a resource to be closed during shutdown.
// Use this for health checkers, rate limiter GC, hot reloaders, etc.
func (s *Server) RegisterCloser(c io.Closer) {
	s.closers = append(s.closers, c)
}

// ListenAndServe starts the server and blocks until shutdown completes.
//
// Shutdown sequence:
//  1. Wait for SIGTERM or SIGINT
//  2. Stop accepting new connections
//  3. Wait for in-flight requests to finish (up to drainTimeout)
//  4. Close registered background resources
//  5. Return
func (s *Server) ListenAndServe() error {
	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("server starting", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-errCh:
		return err // server failed to start
	case sig := <-sigCh:
		s.logger.Info("shutdown signal received", "signal", sig.String())
	}

	// Graceful shutdown
	s.logger.Info("draining connections", "timeout", s.drainTimeout.String())

	ctx, cancel := context.WithTimeout(context.Background(), s.drainTimeout)
	defer cancel()

	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		s.logger.Error("shutdown error, forcing close", "error", err)
		s.httpServer.Close()
	}

	// Close background resources
	for _, c := range s.closers {
		if err := c.Close(); err != nil {
			s.logger.Warn("error closing resource", "error", err)
		}
	}

	s.logger.Info("shutdown complete")
	return nil
}
