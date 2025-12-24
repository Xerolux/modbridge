package ratelimit

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrRateLimitExceeded is returned when the rate limit is exceeded.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// TokenBucket implements the token bucket algorithm for rate limiting.
type TokenBucket struct {
	capacity  int           // Maximum number of tokens
	tokens    float64       // Current number of tokens
	refillRate float64      // Tokens added per second
	lastRefill time.Time    // Last refill time
	mu        sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter.
// capacity: maximum number of tokens (burst size)
// refillRate: tokens added per second
func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     float64(capacity),
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request can proceed and consumes a token if available.
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

// AllowN checks if n tokens are available and consumes them if so.
func (tb *TokenBucket) AllowN(n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}

	return false
}

// refill adds tokens based on elapsed time.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// Add tokens based on elapsed time
	tb.tokens += elapsed * tb.refillRate

	// Cap at capacity
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
}

// Available returns the number of tokens currently available.
func (tb *TokenBucket) Available() int {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()
	return int(tb.tokens)
}

// Reset resets the token bucket to full capacity.
func (tb *TokenBucket) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tokens = float64(tb.capacity)
	tb.lastRefill = time.Now()
}

// Limiter manages rate limiting for multiple keys (e.g., IPs, proxies).
type Limiter struct {
	limiters sync.Map // map[string]*TokenBucket
	capacity int
	rate     float64
}

// NewLimiter creates a new multi-key rate limiter.
func NewLimiter(capacity int, rate float64) *Limiter {
	return &Limiter{
		capacity: capacity,
		rate:     rate,
	}
}

// Allow checks if a request for the given key can proceed.
func (l *Limiter) Allow(key string) bool {
	limiter := l.getLimiter(key)
	return limiter.Allow()
}

// AllowN checks if n requests for the given key can proceed.
func (l *Limiter) AllowN(key string, n int) bool {
	limiter := l.getLimiter(key)
	return limiter.AllowN(n)
}

// getLimiter gets or creates a token bucket for a key.
func (l *Limiter) getLimiter(key string) *TokenBucket {
	value, ok := l.limiters.Load(key)
	if ok {
		return value.(*TokenBucket)
	}

	// Create new limiter
	limiter := NewTokenBucket(l.capacity, l.rate)
	value, _ = l.limiters.LoadOrStore(key, limiter)
	return value.(*TokenBucket)
}

// Reset resets the rate limiter for a specific key.
func (l *Limiter) Reset(key string) {
	if value, ok := l.limiters.Load(key); ok {
		value.(*TokenBucket).Reset()
	}
}

// Remove removes the rate limiter for a specific key.
func (l *Limiter) Remove(key string) {
	l.limiters.Delete(key)
}

// Stats returns statistics for a specific key.
func (l *Limiter) Stats(key string) int {
	if value, ok := l.limiters.Load(key); ok {
		return value.(*TokenBucket).Available()
	}
	return l.capacity
}

// Cleanup removes inactive limiters (call periodically).
func (l *Limiter) Cleanup(inactiveThreshold time.Duration) {
	now := time.Now()
	l.limiters.Range(func(key, value interface{}) bool {
		limiter := value.(*TokenBucket)
		limiter.mu.Lock()
		inactive := now.Sub(limiter.lastRefill) > inactiveThreshold
		limiter.mu.Unlock()

		if inactive {
			l.limiters.Delete(key)
		}
		return true
	})
}
