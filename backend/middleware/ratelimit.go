package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// entry pairs a per-IP limiter with the last time it was used, so idle
// entries can be evicted instead of accumulating forever.
type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit returns a gin middleware that limits requests per client IP
// using a token-bucket limiter (requestsPerMinute refill rate, burst capacity).
//
// A background goroutine periodically sweeps out limiters for IPs that
// haven't made a request in a while, so memory use stays bounded even
// under many distinct client IPs (instead of growing forever).
func RateLimit(requestsPerMinute float64, burst int) gin.HandlerFunc {
	const (
		idleTimeout   = 10 * time.Minute
		sweepInterval = 5 * time.Minute
	)

	var (
		mu       sync.Mutex
		limiters = make(map[string]*entry)
	)

	limitPerSecond := rate.Limit(requestsPerMinute / 60)

	getLimiter := func(ip string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		e, exists := limiters[ip]
		if !exists {
			e = &entry{limiter: rate.NewLimiter(limitPerSecond, burst)}
			limiters[ip] = e
		}
		e.lastSeen = time.Now()
		return e.limiter
	}

	go func() {
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()
		for range ticker.C {
			cutoff := time.Now().Add(-idleTimeout)
			mu.Lock()
			for ip, e := range limiters {
				if e.lastSeen.Before(cutoff) {
					delete(limiters, ip)
				}
			}
			mu.Unlock()
		}
	}()

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
