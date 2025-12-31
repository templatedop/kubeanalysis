package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewHealthManager(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	if hm == nil {
		t.Fatal("expected health manager to be non-nil")
	}
	if hm.version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got %s", hm.version)
	}
}

func TestHealthManagerSetReady(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	if hm.IsReady() {
		t.Error("should not be ready initially")
	}

	hm.SetReady(true)
	if !hm.IsReady() {
		t.Error("should be ready after SetReady(true)")
	}

	hm.SetReady(false)
	if hm.IsReady() {
		t.Error("should not be ready after SetReady(false)")
	}
}

func TestHealthManagerRegisterChecker(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	checker := func(ctx context.Context) error {
		return nil
	}

	hm.RegisterChecker("database", checker)

	health := hm.GetComponentHealth("database")
	if health == nil {
		t.Fatal("expected component health to be registered")
	}
	if health.Name != "database" {
		t.Errorf("expected name 'database', got %s", health.Name)
	}
}

func TestHealthManagerCheckHealth(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	// Register a healthy checker
	hm.RegisterChecker("healthy", func(ctx context.Context) error {
		return nil
	})

	// Register an unhealthy checker
	hm.RegisterChecker("unhealthy", func(ctx context.Context) error {
		return errors.New("service unavailable")
	})

	ctx := context.Background()
	health := hm.CheckHealth(ctx)

	if health.Status != HealthStatusUnhealthy {
		t.Errorf("expected status unhealthy, got %s", health.Status)
	}

	if len(health.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(health.Components))
	}
}

func TestHealthManagerAllHealthy(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	hm.RegisterChecker("service1", func(ctx context.Context) error {
		return nil
	})
	hm.RegisterChecker("service2", func(ctx context.Context) error {
		return nil
	})

	ctx := context.Background()
	health := hm.CheckHealth(ctx)

	if health.Status != HealthStatusHealthy {
		t.Errorf("expected status healthy, got %s", health.Status)
	}
}

func TestLivenessHandler(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	req := httptest.NewRequest("GET", "/live", nil)
	rec := httptest.NewRecorder()

	hm.LivenessHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestReadinessHandlerNotReady(t *testing.T) {
	hm := NewHealthManager("1.0.0")
	// Not ready by default

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	hm.ReadinessHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 when not ready, got %d", rec.Code)
	}
}

func TestReadinessHandlerReady(t *testing.T) {
	hm := NewHealthManager("1.0.0")
	hm.SetReady(true)

	// Register healthy checker
	hm.RegisterChecker("service", func(ctx context.Context) error {
		return nil
	})

	req := httptest.NewRequest("GET", "/ready", nil)
	rec := httptest.NewRecorder()

	hm.ReadinessHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200 when ready, got %d", rec.Code)
	}
}

func TestHealthHandler(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	hm.RegisterChecker("database", func(ctx context.Context) error {
		return nil
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	hm.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("expected Content-Type application/json")
	}
}

func TestHealthHandlerUnhealthy(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	hm.RegisterChecker("failing", func(ctx context.Context) error {
		return errors.New("service down")
	})

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()

	hm.HealthHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for unhealthy, got %d", rec.Code)
	}
}

func TestHealthCheckTimeout(t *testing.T) {
	hm := NewHealthManager("1.0.0")

	// Register a slow checker
	hm.RegisterChecker("slow", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
			return nil
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	health := hm.CheckHealth(ctx)
	duration := time.Since(start)

	// Should complete quickly due to timeout
	if duration > 6*time.Second {
		t.Error("health check should timeout, not wait for slow checker")
	}

	// Component should be unhealthy due to timeout
	for _, comp := range health.Components {
		if comp.Name == "slow" {
			if comp.Status == HealthStatusHealthy {
				t.Error("slow component should be unhealthy due to timeout")
			}
		}
	}
}

func TestHealthStatus(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthStatusHealthy, "healthy"},
		{HealthStatusUnhealthy, "unhealthy"},
		{HealthStatusDegraded, "degraded"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
		}
	}
}
