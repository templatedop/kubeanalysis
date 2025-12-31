// Package server provides the HTTP server with graceful shutdown.
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kubeanalysis/kubeanalysis/internal/observability"
	"go.uber.org/zap"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
	Host              string        `yaml:"host" json:"host"`
	Port              int           `yaml:"port" json:"port"`
	ReadTimeout       time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout" json:"shutdown_timeout"`
	EnableMetrics     bool          `yaml:"enable_metrics" json:"enable_metrics"`
	MetricsPath       string        `yaml:"metrics_path" json:"metrics_path"`
	HealthPath        string        `yaml:"health_path" json:"health_path"`
	ReadinessPath     string        `yaml:"readiness_path" json:"readiness_path"`
	LivenessPath      string        `yaml:"liveness_path" json:"liveness_path"`
}

// DefaultServerConfig returns default server configuration.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Host:            "0.0.0.0",
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     120 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		EnableMetrics:   true,
		MetricsPath:     "/metrics",
		HealthPath:      "/health",
		ReadinessPath:   "/ready",
		LivenessPath:    "/live",
	}
}

// Server represents the HTTP server with graceful shutdown.
type Server struct {
	config        ServerConfig
	httpServer    *http.Server
	mux           *http.ServeMux
	healthManager *observability.HealthManager
	rateLimiter   *observability.RateLimiter
	logger        *zap.Logger

	// Lifecycle management
	mu         sync.Mutex
	running    bool
	shutdownCh chan struct{}
	doneCh     chan struct{}

	// Shutdown hooks
	shutdownHooks []func(context.Context) error
}

// NewServer creates a new server.
func NewServer(cfg ServerConfig, version string) *Server {
	logger := observability.Logger()

	s := &Server{
		config:        cfg,
		mux:           http.NewServeMux(),
		healthManager: observability.NewHealthManager(version),
		logger:        logger,
		shutdownCh:    make(chan struct{}),
		doneCh:        make(chan struct{}),
	}

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      s.mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	// Register default endpoints
	s.registerDefaultEndpoints()

	return s
}

// SetRateLimiter sets the rate limiter for the server.
func (s *Server) SetRateLimiter(rl *observability.RateLimiter) {
	s.rateLimiter = rl
}

// HealthManager returns the health manager.
func (s *Server) HealthManager() *observability.HealthManager {
	return s.healthManager
}

// Handle registers a handler for a path.
func (s *Server) Handle(pattern string, handler http.Handler) {
	if s.rateLimiter != nil {
		handler = observability.RateLimitMiddleware(s.rateLimiter, observability.IPKeyFunc)(handler)
	}
	handler = s.loggingMiddleware(handler)
	s.mux.Handle(pattern, handler)
}

// HandleFunc registers a handler function for a path.
func (s *Server) HandleFunc(pattern string, handler http.HandlerFunc) {
	s.Handle(pattern, handler)
}

func (s *Server) registerDefaultEndpoints() {
	// Health endpoints (no rate limiting or logging for these)
	s.mux.HandleFunc(s.config.LivenessPath, s.healthManager.LivenessHandler())
	s.mux.HandleFunc(s.config.ReadinessPath, s.healthManager.ReadinessHandler())
	s.mux.HandleFunc(s.config.HealthPath, s.healthManager.HealthHandler())

	// Metrics endpoint
	if s.config.EnableMetrics {
		s.mux.Handle(s.config.MetricsPath, observability.MetricsHandler())
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		s.logger.Info("HTTP request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", wrapped.statusCode),
			zap.Duration("duration", duration),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// OnShutdown registers a function to be called during shutdown.
func (s *Server) OnShutdown(hook func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownHooks = append(s.shutdownHooks, hook)
}

// Start starts the server.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info("Starting server",
		zap.String("addr", s.httpServer.Addr),
	)

	// Mark as ready
	s.healthManager.SetReady(true)

	// Start background health checks
	ctx := context.Background()
	s.healthManager.StartBackgroundChecks(ctx, 30*time.Second)

	// Start HTTP server
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Server error", zap.Error(err))
		}
	}()

	return nil
}

// StartWithGracefulShutdown starts the server and handles graceful shutdown.
func (s *Server) StartWithGracefulShutdown() error {
	if err := s.Start(); err != nil {
		return err
	}

	// Wait for shutdown signal
	s.waitForShutdown()

	return nil
}

func (s *Server) waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		s.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
	case <-s.shutdownCh:
		s.logger.Info("Shutdown requested")
	}

	if err := s.Shutdown(); err != nil {
		s.logger.Error("Shutdown error", zap.Error(err))
	}

	close(s.doneCh)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	s.logger.Info("Shutting down server",
		zap.Duration("timeout", s.config.ShutdownTimeout),
	)

	// Mark as not ready
	s.healthManager.SetReady(false)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// Run shutdown hooks
	var wg sync.WaitGroup
	for _, hook := range s.shutdownHooks {
		wg.Add(1)
		go func(h func(context.Context) error) {
			defer wg.Done()
			if err := h(ctx); err != nil {
				s.logger.Error("Shutdown hook error", zap.Error(err))
			}
		}(hook)
	}

	// Wait for hooks or timeout
	hooksDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(hooksDone)
	}()

	select {
	case <-hooksDone:
		s.logger.Info("Shutdown hooks completed")
	case <-ctx.Done():
		s.logger.Warn("Shutdown hooks timed out")
	}

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown error: %w", err)
	}

	s.logger.Info("Server shutdown complete")
	return nil
}

// Stop triggers a shutdown.
func (s *Server) Stop() {
	close(s.shutdownCh)
}

// Done returns a channel that's closed when the server is done.
func (s *Server) Done() <-chan struct{} {
	return s.doneCh
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.httpServer.Addr
}
