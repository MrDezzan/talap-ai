package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimit returns a gin middleware that limits requests per client IP
// using a token-bucket limiter (requestsPerMinute refill rate, burst capacity).
//
// NOTE: this keeps one *rate.Limiter per IP forever in memory. For a first
// pass that's fine, but a production version should periodically sweep and
// evict limiters for IPs that haven't been seen in a while (e.g. a
// time.Ticker goroutine removing entries older than N minutes) to avoid
// unbounded memory growth under many distinct client IPs.
func RateLimit(requestsPerMinute float64, burst int) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		limiters = make(map[string]*rate.Limiter)
	)

	limitPerSecond := rate.Limit(requestsPerMinute / 60)

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		limiter, exists := limiters[ip]
		if !exists {
			limiter = rate.NewLimiter(limitPerSecond, burst)
			limiters[ip] = limiter
		}
		return limiter
	}

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := getLimiter(ip)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, slow down"})
			return
		}

		c.Next()
	}
}
