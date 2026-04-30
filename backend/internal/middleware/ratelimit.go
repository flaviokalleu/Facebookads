package middleware

import (
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type rateBucket struct {
	count    int
	resetAt  time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
}

var globalLimiter = &rateLimiter{
	buckets: make(map[string]*rateBucket),
	limit:   100,
	window:  time.Minute,
}

// RateLimit enforces limit requests per window per user (by JWT user_id or IP).
func RateLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := UserID(c)
		if key == "" {
			key = c.IP()
		}

		globalLimiter.mu.Lock()
		b, ok := globalLimiter.buckets[key]
		now := time.Now()
		if !ok || now.After(b.resetAt) {
			globalLimiter.buckets[key] = &rateBucket{count: 1, resetAt: now.Add(globalLimiter.window)}
			globalLimiter.mu.Unlock()
			return c.Next()
		}
		b.count++
		remaining := globalLimiter.limit - b.count
		resetAt := b.resetAt
		globalLimiter.mu.Unlock()

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", globalLimiter.limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(remaining, 0)))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt.Unix()))

		if remaining < 0 {
			c.Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetAt).Seconds())))
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return c.Next()
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
