package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthStatus represents the health status.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// ComponentHealth represents health of a single component.
type ComponentHealth struct {
	Name    string       `json:"name"`
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
	LastCheck time.Time  `json:"last_check"`
}

// HealthResponse is the response from health endpoints.
type HealthResponse struct {
	Status     HealthStatus      `json:"status"`
	Components []ComponentHealth `json:"components,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Version    string            `json:"version,omitempty"`
}

// HealthChecker defines a function that checks health of a component.
type HealthChecker func(ctx context.Context) error

// HealthManager manages health checks for multiple components.
type HealthManager struct {
	mu         sync.RWMutex
	checkers   map[string]HealthChecker
	status     map[string]*ComponentHealth
	version    string
	ready      bool
}

// NewHealthManager creates a new health manager.
func NewHealthManager(version string) *HealthManager {
	return &HealthManager{
		checkers: make(map[string]HealthChecker),
		status:   make(map[string]*ComponentHealth),
		version:  version,
	}
}

// RegisterChecker registers a health checker for a component.
func (hm *HealthManager) RegisterChecker(name string, checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checkers[name] = checker
	hm.status[name] = &ComponentHealth{
		Name:   name,
		Status: HealthStatusUnhealthy,
	}
}

// SetReady marks the application as ready to serve traffic.
func (hm *HealthManager) SetReady(ready bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.ready = ready
}

// IsReady returns whether the application is ready.
func (hm *HealthManager) IsReady() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.ready
}

// CheckHealth runs all health checks.
func (hm *HealthManager) CheckHealth(ctx context.Context) HealthResponse {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	var components []ComponentHealth
	overallStatus := HealthStatusHealthy

	for name, checker := range hm.checkers {
		health := hm.checkComponent(ctx, name, checker)
		components = append(components, health)
		hm.status[name] = &health

		// Update overall status
		if health.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if health.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}

		// Update metrics
		if globalMetrics != nil {
			globalMetrics.SetHealthStatus(name, health.Status == HealthStatusHealthy)
		}
	}

	return HealthResponse{
		Status:     overallStatus,
		Components: components,
		Timestamp:  time.Now(),
		Version:    hm.version,
	}
}

func (hm *HealthManager) checkComponent(ctx context.Context, name string, checker HealthChecker) ComponentHealth {
	// Create a timeout context for the health check
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := checker(checkCtx)

	health := ComponentHealth{
		Name:      name,
		LastCheck: time.Now(),
	}

	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.Message = err.Error()
	} else {
		health.Status = HealthStatusHealthy
	}

	return health
}

// GetComponentHealth returns health of a specific component.
func (hm *HealthManager) GetComponentHealth(name string) *ComponentHealth {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.status[name]
}

// LivenessHandler returns an HTTP handler for liveness probes.
// Liveness checks if the application is running (not deadlocked).
func (hm *HealthManager) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "alive",
		})
	}
}

// ReadinessHandler returns an HTTP handler for readiness probes.
// Readiness checks if the application is ready to serve traffic.
func (hm *HealthManager) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if !hm.IsReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "not_ready",
				"message": "Application is not ready to serve traffic",
			})
			return
		}

		// Run health checks
		ctx := r.Context()
		health := hm.CheckHealth(ctx)

		if health.Status == HealthStatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(health)
	}
}

// HealthHandler returns an HTTP handler for detailed health checks.
func (hm *HealthManager) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		ctx := r.Context()
		health := hm.CheckHealth(ctx)

		switch health.Status {
		case HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still serving, but degraded
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		json.NewEncoder(w).Encode(health)
	}
}

// StartBackgroundChecks starts periodic background health checks.
func (hm *HealthManager) StartBackgroundChecks(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				hm.CheckHealth(ctx)
			}
		}
	}()
}
