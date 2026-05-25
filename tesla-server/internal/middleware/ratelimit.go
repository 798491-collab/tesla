package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitorInfo
	rate     int
	interval time.Duration
}

type visitorInfo struct {
	count    int
	lastSeen time.Time
}

func newRateLimiter(rate int, interval time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitorInfo),
		rate:     rate,
		interval: interval,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > rl.interval {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists || time.Since(v.lastSeen) > rl.interval {
		rl.visitors[ip] = &visitorInfo{count: 1, lastSeen: time.Now()}
		return true
	}

	v.count++
	v.lastSeen = time.Now()
	return v.count <= rl.rate
}

var (
	authLimiter    = newRateLimiter(10, time.Minute)
	apiLimiter     = newRateLimiter(60, time.Minute)
	controlLimiter = newRateLimiter(30, time.Minute)
)

func RateLimitAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !authLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RateLimitAPI() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !apiLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RateLimitControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !controlLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "Too many vehicle control requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
