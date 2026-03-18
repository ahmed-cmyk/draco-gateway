package middleware

import (
	"net/http"
	"strings"
	"sync"

	"github.com/charmbracelet/log"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.RWMutex
	r   rate.Limit
	b   int
}

func NewLimiter(limit rate.Limit, bucket int) IPRateLimiter {
	return IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   limit,
		b:   bucket,
	}
}

func (l *IPRateLimiter) fetchLimiter(ip string) *rate.Limiter {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, ok := l.ips[ip]
	if !ok {
		l.ips[ip] = rate.NewLimiter(l.r, l.b)
		return l.ips[ip]
	}

	return limiter
}

func getIP(r *http.Request) string {
	// Try to grab client IP address from `X-Forwarded-For` header
	if headerValue := r.Header.Get("X-Forwarded-For"); headerValue != "" {
		// The header can contain multiple IPs
		ips := strings.Split(headerValue, ",")
		// The first IP in 'X-Forwarded-For' is usually the original client's address
		for i, ip := range ips {
			ips[i] = strings.TrimSpace(ip)
		}
		if len(ips) > 0 && ips[0] != "" {
			return ips[0]
		}
	}

	// Fallback to RemoteAddr if other sources does not contain header
	ipPort := r.RemoteAddr
	if ipPort != "" {
		parts := strings.Split(ipPort, ":")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	return ""
}

func RateLimit(manager *IPRateLimiter) MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get IP address of user making request
			ip := getIP(r)
			if ip == "" {
				log.Errorf("Could not retrieve IP address from header")
				return
			}

			// Now 'manager' is accessible here!
			limiter := manager.fetchLimiter(ip)

			if !limiter.Allow() {
				http.Error(w, "Slow down, Gopher.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
