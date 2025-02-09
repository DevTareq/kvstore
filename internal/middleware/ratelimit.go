package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter keeps track of request counts per client
type RateLimiter struct {
	requests map[string]int
	mu       sync.Mutex
	interval time.Duration
	limit    int
}

// NewRateLimiter initializes the rate limiter
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]int),
		interval: interval,
		limit:    limit,
	}

	go func() {
		for range time.Tick(interval) {
			rl.mu.Lock()
			rl.requests = make(map[string]int) // Reset counts
			rl.mu.Unlock()
		}
	}()
	return rl
}

// LimitMiddleware applies rate limiting per IP
func (rl *RateLimiter) LimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr

		rl.mu.Lock()
		rl.requests[clientIP]++
		count := rl.requests[clientIP]
		rl.mu.Unlock()

		if count > rl.limit {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
