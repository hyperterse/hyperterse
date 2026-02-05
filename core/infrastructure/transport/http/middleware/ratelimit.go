package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter defines the interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RedisRateLimiter implements rate limiting using Redis
type RedisRateLimiter struct {
	client *redis.Client
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client) *RedisRateLimiter {
	return &RedisRateLimiter{client: client}
}

// Allow checks if a request should be allowed based on rate limit
func (r *RedisRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	// Use Redis sliding window log algorithm
	now := time.Now()
	windowStart := now.Add(-window)

	// Remove old entries
	r.client.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.Unix(), 10))

	// Count current entries
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return false, err
	}

	if count >= int64(limit) {
		return false, nil
	}

	// Add current request
	_, err = r.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.Unix()),
		Member: now.UnixNano(),
	}).Result()
	if err != nil {
		return false, err
	}

	// Set expiry
	r.client.Expire(ctx, key, window)

	return true, nil
}

// RateLimit middleware for rate limiting
func RateLimit(limiter RateLimiter, limit int, window time.Duration, keyFunc func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				// If no key, allow request
				next.ServeHTTP(w, r)
				return
			}

			allowed, err := limiter.Allow(r.Context(), key, limit, window)
			if err != nil {
				// On error, allow request but log error
				// In production, you might want to fail closed
				next.ServeHTTP(w, r)
				return
			}

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error":"rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitByIP creates rate limit middleware that limits by IP address
func RateLimitByIP(limiter RateLimiter, limit int, window time.Duration) func(http.Handler) http.Handler {
	return RateLimit(limiter, limit, window, func(r *http.Request) string {
		// Get IP from X-Forwarded-For header or RemoteAddr
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.RemoteAddr
		}
		return "ratelimit:" + ip
	})
}
