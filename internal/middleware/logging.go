package middleware

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initialize our wrapper with a default 200 (in case WriteHeader isn't called)
		rw := &responseWriter{w, http.StatusOK}

		start := time.Now()

		// Pass our wrapper instead of the original 'w'
		next.ServeHTTP(rw, r)

		elapsed := time.Since(start)

		log.Debugf("[%d] %s %s (took %d ms)", rw.statusCode, r.Method, r.URL.Path, elapsed.Milliseconds())
	})
}
