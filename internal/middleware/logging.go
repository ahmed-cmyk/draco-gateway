package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

var Log *zap.Logger

type responseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func InitLogger() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	defer logger.Sync()

	Log = logger
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Initialize our wrapper with a default 200 (in case WriteHeader isn't called)
		rw := &responseWriter{w, http.StatusOK}

		// Pass our wrapper instead of the original 'w'
		next.ServeHTTP(rw, r)

		Log.Info("request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rw.StatusCode),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", r.RemoteAddr),
			zap.String("user_agent", r.UserAgent()),
		)
	})
}
