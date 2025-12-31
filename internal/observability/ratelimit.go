package observability

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// RateLimitConfig holds rate limiter configuration.
type RateLimitConfig struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	RequestsPerSec  float64       `yaml:"requests_per_sec" json:"requests_per_sec"`
	BurstSize       int           `yaml:"burst_size" json:"burst_size"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
}

// DefaultRateLimitConfig returns default rate limit configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Enabled:         true,
		RequestsPerSec:  10.0,
		BurstSize:       20,
		CleanupInterval: time.Minute,
	}
}

// TokenBucket implements the token bucket algorithm for rate limiting.
type TokenBucket struct {
	mu           sync.Mutex
	tokens       float64
	maxTokens    float64
	refillRate   float64 // tokens per second
	lastRefill   time.Time
}

// NewTokenBucket creates a new token bucket.
func NewTokenBucket(maxTokens float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed and consumes a token if so.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// AllowN checks if n requests are allowed and consumes n tokens if so.
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	needed := float64(n)
	if tb.tokens >= needed {
		tb.tokens -= needed
		return true
	}
	return false
}

// Tokens returns the current number of available tokens.
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
}

// RateLimiter provides rate limiting for multiple keys.
type RateLimiter struct {
	mu              sync.RWMutex
	config          RateLimitConfig
	buckets         map[string]*TokenBucket
	defaultBucket   *TokenBucket
	cleanupTicker   *time.Ticker
	stopCleanup     chan struct{}
}

// NewRateLimiter creates a new rate limiter.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config:        cfg,
		buckets:       make(map[string]*TokenBucket),
		defaultBucket: NewTokenBucket(float64(cfg.BurstSize), cfg.RequestsPerSec),
		stopCleanup:   make(chan struct{}),
	}

	if cfg.CleanupInterval > 0 {
		rl.cleanupTicker = time.NewTicker(cfg.CleanupInterval)
		go rl.cleanupLoop()
	}

	return rl
}

// Allow checks if a request is allowed for the default bucket.
func (rl *RateLimiter) Allow() bool {
	if !rl.config.Enabled {
		return true
	}
	return rl.defaultBucket.Allow()
}

// AllowKey checks if a request is allowed for a specific key.
func (rl *RateLimiter) AllowKey(key string) bool {
	if !rl.config.Enabled {
		return true
	}

	bucket := rl.getBucket(key)
	allowed := bucket.Allow()

	if !allowed && globalMetrics != nil {
		globalMetrics.RateLimitHits.WithLabelValues(key).Inc()
	}

	if globalMetrics != nil {
		globalMetrics.RateLimitRemaining.WithLabelValues(key).Set(bucket.Tokens())
	}

	return allowed
}

func (rl *RateLimiter) getBucket(key string) *TokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, exists := rl.buckets[key]; exists {
		return bucket
	}

	bucket = NewTokenBucket(float64(rl.config.BurstSize), rl.config.RequestsPerSec)
	rl.buckets[key] = bucket
	return bucket
}

func (rl *RateLimiter) cleanupLoop() {
	for {
		select {
		case <-rl.stopCleanup:
			return
		case <-rl.cleanupTicker.C:
			rl.cleanup()
		}
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Remove buckets that are full (haven't been used recently)
	for key, bucket := range rl.buckets {
		if bucket.Tokens() >= float64(rl.config.BurstSize)-0.1 {
			delete(rl.buckets, key)
		}
	}
}

// Close stops the rate limiter.
func (rl *RateLimiter) Close() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
	close(rl.stopCleanup)
}

// Wait waits until a token is available or context is cancelled.
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if !rl.config.Enabled {
		return nil
	}

	if rl.defaultBucket.Allow() {
		return nil
	}

	// Calculate wait time based on refill rate
	waitTime := time.Duration(float64(time.Second) / rl.config.RequestsPerSec)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		if rl.defaultBucket.Allow() {
			return nil
		}
		return ErrRateLimitExceeded
	}
}

// WaitKey waits until a token is available for the key or context is cancelled.
func (rl *RateLimiter) WaitKey(ctx context.Context, key string) error {
	if !rl.config.Enabled {
		return nil
	}

	bucket := rl.getBucket(key)

	if bucket.Allow() {
		return nil
	}

	// Calculate wait time based on refill rate
	waitTime := time.Duration(float64(time.Second) / rl.config.RequestsPerSec)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		if bucket.Allow() {
			return nil
		}
		return ErrRateLimitExceeded
	}
}

// RateLimitMiddleware creates an HTTP middleware for rate limiting.
func RateLimitMiddleware(rl *RateLimiter, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)

			if !rl.AllowKey(key) {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPKeyFunc returns the client IP as the rate limit key.
func IPKeyFunc(r *http.Request) string {
	// Check X-Forwarded-For first for proxied requests
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return r.RemoteAddr
}

// APIKeyFunc returns the API key as the rate limit key.
func APIKeyFunc(r *http.Request) string {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}
	if key := r.Header.Get("Authorization"); key != "" {
		return key
	}
	return IPKeyFunc(r)
}
