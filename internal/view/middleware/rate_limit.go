package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/kamisamamayuri-cyber/pgwarden/internal/view/web/i18n"
	"github.com/labstack/echo/v4"
)

// RateLimitConfig defines the config for RateLimit middleware.
type RateLimitConfig struct {
	// Limit is the maximum number of requests to allow per period.
	Limit int
	// Period is the duration in which the limit is enforced.
	Period time.Duration
}

// RateLimit creates a rate limiting middleware.
func (Middleware) RateLimit(config RateLimitConfig) echo.MiddlewareFunc {
	var mu sync.Mutex
	var hits = make(map[string]int)

	// Reset the hits map every "period".
	go func() {
		for {
			time.Sleep(config.Period)
			mu.Lock()
			hits = make(map[string]int)
			mu.Unlock()
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			// Only the counter update is under the lock: holding it across
			// next(c) would serialize all requests behind this middleware.
			mu.Lock()
			limited := hits[ip] >= config.Limit
			if !limited {
				hits[ip]++
			}
			mu.Unlock()

			if limited {
				return c.String(http.StatusTooManyRequests, i18n.ErrTooManyRequests)
			}
			return next(c)
		}
	}
}
