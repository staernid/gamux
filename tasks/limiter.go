package tasks

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter defines the rate limiting seam for bandwidth management.
type Limiter interface {
	WaitN(ctx context.Context, n int) error
	SetLimit(bytesPerSec int64)
	Limit() int64
}

// TokenBucketLimiter uses a token bucket to enforce bandwidth limits.
type TokenBucketLimiter struct {
	mu      sync.RWMutex
	limiter *rate.Limiter
	limit   int64
}

// NewTokenBucketLimiter creates a TokenBucketLimiter with the specified bytes/sec limit (0 = unlimited).
func NewTokenBucketLimiter(bytesPerSec int64) *TokenBucketLimiter {
	t := &TokenBucketLimiter{
		limit: bytesPerSec,
	}
	if bytesPerSec > 0 {
		burst := int(bytesPerSec)
		if burst < 65536 {
			burst = 65536
		}
		t.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burst)
	}
	return t
}

// WaitN waits for n tokens to become available.
func (t *TokenBucketLimiter) WaitN(ctx context.Context, n int) error {
	t.mu.RLock()
	lim := t.limiter
	t.mu.RUnlock()

	if lim == nil || n <= 0 {
		return nil
	}

	burst := lim.Burst()
	for n > burst {
		if err := lim.WaitN(ctx, burst); err != nil {
			return err
		}
		n -= burst
	}

	return lim.WaitN(ctx, n)
}

// SetLimit updates the rate limit dynamically.
func (t *TokenBucketLimiter) SetLimit(bytesPerSec int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.limit = bytesPerSec
	if bytesPerSec <= 0 {
		t.limiter = nil
		return
	}

	burst := int(bytesPerSec)
	if burst < 65536 {
		burst = 65536
	}

	if t.limiter == nil {
		t.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), burst)
	} else {
		t.limiter.SetLimit(rate.Limit(bytesPerSec))
		t.limiter.SetBurst(burst)
	}
}

// Limit returns the current bytes/sec limit.
func (t *TokenBucketLimiter) Limit() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.limit
}

// NoopLimiter is an unconstrained limiter.
type NoopLimiter struct{}

// NewNoopLimiter creates a NoopLimiter.
func NewNoopLimiter() *NoopLimiter {
	return &NoopLimiter{}
}

// WaitN immediately returns without waiting.
func (n *NoopLimiter) WaitN(ctx context.Context, bytes int) error {
	return nil
}

// SetLimit does nothing.
func (n *NoopLimiter) SetLimit(bytesPerSec int64) {}

// Limit returns 0 (unlimited).
func (n *NoopLimiter) Limit() int64 {
	return 0
}
