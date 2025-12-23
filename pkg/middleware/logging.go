package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	bytesWritten int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Logging is a middleware that logs HTTP requests with structured logging.
func Logging(logger *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Get request ID from context
			requestID := GetRequestID(r.Context())

			// Wrap response writer
			wrapped := &responseWriter{
				ResponseWriter: w,
				status:        0,
				bytesWritten:  0,
			}

			// Log request start
			logger.Debug().
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Msg("Request started")

			// Call next handler
			next.ServeHTTP(wrapped, r)

			// Calculate duration
			duration := time.Since(start)

			// Log request completion
			event := logger.Info()
			if wrapped.status >= 400 {
				event = logger.Warn()
			}
			if wrapped.status >= 500 {
				event = logger.Error()
			}

			event.
				Str("request_id", requestID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", wrapped.status).
				Dur("duration", duration).
				Int("bytes", wrapped.bytesWritten).
				Str("remote_addr", r.RemoteAddr).
				Msg("Request completed")
		})
	}
}

// LoggingFunc is a function-based variant of the Logging middleware.
func LoggingFunc(logger *zerolog.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := GetRequestID(r.Context())

		wrapped := &responseWriter{
			ResponseWriter: w,
			status:        0,
			bytesWritten:  0,
		}

		logger.Debug().
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("Request started")

		next(wrapped, r)

		duration := time.Since(start)

		event := logger.Info()
		if wrapped.status >= 400 {
			event = logger.Warn()
		}
		if wrapped.status >= 500 {
			event = logger.Error()
		}

		event.
			Str("request_id", requestID).
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wrapped.status).
			Dur("duration", duration).
			Int("bytes", wrapped.bytesWritten).
			Msg("Request completed")
	}
}
