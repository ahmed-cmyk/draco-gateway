package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type MiddlewareFunc func(http.Handler) http.Handler

type Registry struct {
	mu    sync.RWMutex
	funcs map[string]MiddlewareFunc
}

func NewRegistry() *Registry {
	return &Registry{
		funcs: map[string]MiddlewareFunc{
			"logging":        Logging,
			"jwt_validation": JWTValidation,
		},
	}
}

func (r *Registry) Get(name string) (MiddlewareFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.funcs[name]
	return fn, ok
}

func (r *Registry) Register(name string, fn MiddlewareFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.funcs[name] = fn
}

// DefaultRegistry is the global registry instance
var DefaultRegistry = NewRegistry()

// RegisterRateLimiter registers the rate limiter middleware with the default registry
func RegisterRateLimiter(duration time.Duration, bucket int) {
	// Initialize the stateful logic
	limiterManager := NewLimiter(rate.Every(duration), bucket)

	// Prime the middleware with its manager
	rateLimitMW := RateLimit(&limiterManager)

	DefaultRegistry.Register("rate_limit", rateLimitMW)
}
