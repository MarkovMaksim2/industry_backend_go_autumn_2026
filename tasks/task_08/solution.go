package main

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	limiter := &Limiter{
		clock:  clock,
		rate:   max(ratePerSec, 0),
		burst:  burst,
		tokens: float64(burst),
	}

	if clock != nil {
		limiter.last = clock.Now()
	}

	return limiter
}
func (l *Limiter) AllowN(n int) bool {
	if l.burst <= 0 || l.clock == nil || n <= 0 || n > l.burst {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.clock.Now().After(l.last) {
		l.tokens += float64(l.clock.Now().Sub(l.last).Seconds()) * l.rate
		l.tokens = min(float64(l.burst), l.tokens)
		l.last = l.clock.Now()
	}

	if l.tokens < float64(n) {
		return false
	}
	l.tokens -= float64(n)

	return true
}
