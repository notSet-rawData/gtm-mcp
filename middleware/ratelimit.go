package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const maxVisitors = 10000

// RateLimiter provides per-IP rate limiting for HTTP endpoints.
type RateLimiter struct {
	mu             sync.Mutex
	visitors       map[string]*visitor
	rate           rate.Limit
	burst          int
	trustedProxies map[string]bool
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter creates a rate limiter with the given requests per second and burst.
// trustedProxies is an optional list of IP addresses whose X-Forwarded-For headers
// will be trusted for client IP extraction.
func NewRateLimiter(rps float64, burst int, trustedProxies ...string) *RateLimiter {
	tp := make(map[string]bool)
	for _, p := range trustedProxies {
		tp[strings.TrimSpace(p)] = true
	}
	rl := &RateLimiter{
		visitors:       make(map[string]*visitor),
		rate:           rate.Limit(rps),
		burst:          burst,
		trustedProxies: tp,
	}
	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) getVisitor(ip string) (*rate.Limiter, bool) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		if len(rl.visitors) >= maxVisitors {
			return nil, false
		}
		limiter := rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = &visitor{limiter: limiter, lastSeen: time.Now()}
		return limiter, true
	}
	v.lastSeen = time.Now()
	return v.limiter, true
}

// cleanup removes stale visitors every minute.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// extractClientIP returns the client IP from the request.
// Only trusts X-Forwarded-For if the direct connection comes from a trusted proxy.
func (rl *RateLimiter) extractClientIP(r *http.Request) string {
	remoteIP := r.RemoteAddr
	// Strip port from RemoteAddr (e.g. "192.168.1.1:12345" -> "192.168.1.1")
	if idx := strings.LastIndex(remoteIP, ":"); idx != -1 {
		remoteIP = remoteIP[:idx]
	}

	// Only trust X-Forwarded-For if the request comes from a known trusted proxy
	if len(rl.trustedProxies) > 0 && rl.trustedProxies[remoteIP] {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			return strings.TrimSpace(strings.SplitN(forwarded, ",", 2)[0])
		}
	}

	return remoteIP
}

func rateLimitReject(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "1")
	w.WriteHeader(http.StatusTooManyRequests)
	w.Write([]byte(`{"error":"rate_limit_exceeded","error_description":"Too many requests. Please retry later."}`))
}

// Middleware returns an HTTP middleware that rate limits by client IP.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := rl.extractClientIP(r)

		limiter, ok := rl.getVisitor(ip)
		if !ok || !limiter.Allow() {
			rateLimitReject(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// MiddlewareFunc wraps an http.HandlerFunc with rate limiting.
func (rl *RateLimiter) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := rl.extractClientIP(r)

		limiter, ok := rl.getVisitor(ip)
		if !ok || !limiter.Allow() {
			rateLimitReject(w)
			return
		}

		next(w, r)
	}
}

// MaxBytesMiddleware wraps a handler with a request body size limit.
func MaxBytesMiddleware(maxBytes int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next(w, r)
	}
}
