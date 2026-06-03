package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

const requestIDHeader = "X-Request-ID"

// RequestLog records one structured access log after each HTTP request.
func RequestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := chimw.GetReqID(r.Context())
		if requestID != "" {
			w.Header().Set(requestIDHeader, requestID)
		}

		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}

		slog.InfoContext(r.Context(), "http request",
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("remote_ip", r.RemoteAddr),
			slog.String("user_agent", r.UserAgent()),
			slog.Int("bytes", ww.BytesWritten()),
		)
	})
}
