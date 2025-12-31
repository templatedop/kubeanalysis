package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()

	if !cfg.Enabled {
		t.Error("expected rate limiting to be enabled by default")
	}
	if cfg.RequestsPerSec != 10.0 {
		t.Errorf("expected 10 requests per sec, got %f", cfg.RequestsPerSec)
	}
	if cfg.BurstSize != 20 {
		t.Errorf("expected burst size 20, got %d", cfg.BurstSize)
	}
}

func TestTokenBucketAllow(t *testing.T) {
	bucket := NewTokenBucket(5, 1) // 5 tokens max, 1 per second refill

	// Should allow 5 requests immediately
	for i := 0; i < 5; i++ {
		if !bucket.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 6th request should be denied
	if bucket.Allow() {
		t.Error("6th request should be denied")
	}
}

func TestTokenBucketRefill(t *testing.T) {
	bucket := NewTokenBucket(2, 10) // 2 tokens max, 10 per second refill

	// Use all tokens
	bucket.Allow()
	bucket.Allow()

	// Should be denied immediately
	if bucket.Allow() {
		t.Error("should be denied after using all tokens")
	}

	// Wait for refill (100ms = 1 token at 10/sec)
	time.Sleep(150 * time.Millisecond)

	// Should be allowed now
	if !bucket.Allow() {
		t.Error("should be allowed after refill")
	}
}

func TestTokenBucketTokens(t *testing.T) {
	bucket := NewTokenBucket(10, 5)

	initial := bucket.Tokens()
	if initial < 9.99 || initial > 10.01 {
		t.Errorf("expected ~10 tokens initially, got %f", initial)
	}

	bucket.Allow()
	bucket.Allow()

	remaining := bucket.Tokens()
	// Use tolerance for floating point comparison
	if remaining < 7.99 || remaining > 8.01 {
		t.Errorf("expected ~8 tokens after 2 requests, got %f", remaining)
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	cfg := RateLimitConfig{
		Enabled:        false,
		RequestsPerSec: 1,
		BurstSize:      1,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	// Should allow all requests when disabled
	for i := 0; i < 100; i++ {
		if !rl.Allow() {
			t.Errorf("request %d should be allowed when disabled", i)
		}
	}
}

func TestRateLimiterAllowKey(t *testing.T) {
	cfg := RateLimitConfig{
		Enabled:        true,
		RequestsPerSec: 10,
		BurstSize:      3,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	// Each key gets its own bucket
	for i := 0; i < 3; i++ {
		if !rl.AllowKey("key1") {
			t.Errorf("key1 request %d should be allowed", i+1)
		}
	}

	// key1 should be rate limited now
	if rl.AllowKey("key1") {
		t.Error("key1 should be rate limited")
	}

	// key2 should still be allowed
	for i := 0; i < 3; i++ {
		if !rl.AllowKey("key2") {
			t.Errorf("key2 request %d should be allowed", i+1)
		}
	}
}

func TestRateLimiterWait(t *testing.T) {
	cfg := RateLimitConfig{
		Enabled:        true,
		RequestsPerSec: 100, // Fast refill for testing
		BurstSize:      1,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	// Use up the bucket
	rl.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err != nil {
		t.Errorf("Wait should succeed with fast refill: %v", err)
	}
}

func TestRateLimiterWaitTimeout(t *testing.T) {
	cfg := RateLimitConfig{
		Enabled:        true,
		RequestsPerSec: 0.1, // Very slow refill
		BurstSize:      1,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	// Use up the bucket
	rl.Allow()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := rl.Wait(ctx)
	if err == nil {
		t.Error("Wait should timeout with slow refill and short context")
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	cfg := RateLimitConfig{
		Enabled:        true,
		RequestsPerSec: 10,
		BurstSize:      2,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(rl, func(r *http.Request) string {
		return "test-key"
	})

	wrapped := middleware(handler)

	// First 2 requests should succeed
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d should return 200, got %d", i+1, rec.Code)
		}
	}

	// 3rd request should be rate limited
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("3rd request should return 429, got %d", rec.Code)
	}
}

func TestIPKeyFunc(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For header",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			remoteAddr: "5.6.7.8:1234",
			expected:   "1.2.3.4",
		},
		{
			name:       "X-Real-IP header",
			headers:    map[string]string{"X-Real-IP": "1.2.3.4"},
			remoteAddr: "5.6.7.8:1234",
			expected:   "1.2.3.4",
		},
		{
			name:       "RemoteAddr fallback",
			headers:    map[string]string{},
			remoteAddr: "5.6.7.8:1234",
			expected:   "5.6.7.8:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := IPKeyFunc(req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAPIKeyFunc(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-API-Key header",
			headers:    map[string]string{"X-API-Key": "my-api-key"},
			remoteAddr: "1.2.3.4:1234",
			expected:   "my-api-key",
		},
		{
			name:       "Authorization header",
			headers:    map[string]string{"Authorization": "Bearer token123"},
			remoteAddr: "1.2.3.4:1234",
			expected:   "Bearer token123",
		},
		{
			name:       "Fallback to IP",
			headers:    map[string]string{},
			remoteAddr: "1.2.3.4:1234",
			expected:   "1.2.3.4:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			result := APIKeyFunc(req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
