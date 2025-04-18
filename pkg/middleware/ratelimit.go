package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/fairuzald/library-system/pkg/utils"
	"go.uber.org/zap"
)

type RateLimiter struct {
	mu           sync.Mutex
	ipLimits     map[string]*TokenBucket
	globalBucket *TokenBucket
	log          *logger.Logger
	ipRate       float64 // tokens per second per IP
	ipBurst      int     // maximum bucket size per IP
	globalRate   float64 // tokens per second for all requests
	globalBurst  int     // maximum bucket size for all requests
	cleanupEvery time.Duration
}

type TokenBucket struct {
	tokens    float64
	lastRefil time.Time
	capacity  int
	rate      float64 // tokens per second
}

func NewRateLimiter(log *logger.Logger, ipRate float64, ipBurst int, globalRate float64, globalBurst int) *RateLimiter {
	rl := &RateLimiter{
		ipLimits:     make(map[string]*TokenBucket),
		globalBucket: &TokenBucket{tokens: float64(globalBurst), lastRefil: time.Now(), capacity: globalBurst, rate: globalRate},
		log:          log,
		ipRate:       ipRate,
		ipBurst:      ipBurst,
		globalRate:   globalRate,
		globalBurst:  globalBurst,
		cleanupEvery: 10 * time.Minute,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	rl.refillBucket(rl.globalBucket, now)
	if rl.globalBucket.tokens < 1 {
		return false
	}
	rl.globalBucket.tokens--

	bucket, exists := rl.ipLimits[ip]
	if !exists {
		bucket = &TokenBucket{
			tokens:    float64(rl.ipBurst),
			lastRefil: now,
			capacity:  rl.ipBurst,
			rate:      rl.ipRate,
		}
		rl.ipLimits[ip] = bucket
	} else {
		rl.refillBucket(bucket, now)
	}

	if bucket.tokens < 1 {
		return false
	}

	bucket.tokens--
	return true
}

func (rl *RateLimiter) refillBucket(bucket *TokenBucket, now time.Time) {
	elapsed := now.Sub(bucket.lastRefil).Seconds()
	bucket.lastRefil = now
	bucket.tokens = min(float64(bucket.capacity), bucket.tokens+(elapsed*bucket.rate))
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupEvery)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, bucket := range rl.ipLimits {
			if bucket.tokens == float64(bucket.capacity) && now.Sub(bucket.lastRefil) > rl.cleanupEvery {
				delete(rl.ipLimits, ip)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := extractClientIP(r)

		if !rl.Allow(clientIP) {
			rl.log.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP),
				zap.String("path", r.URL.Path),
				zap.String("method", r.Method),
			)
			utils.RespondWithError(w, http.StatusTooManyRequests, "Rate limit exceeded", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
