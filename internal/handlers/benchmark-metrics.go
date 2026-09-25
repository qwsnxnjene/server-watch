package handlers

import (
	"net/http"
	"server-watch/internal/prometheus"
	"strconv"
	"time"
)

type responseWriter struct {
	ResponseWriter http.ResponseWriter
	status         int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method := r.Method
		path := r.Pattern

		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		start := time.Now()

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		status := rw.status

		prometheus.HTTPLatency.WithLabelValues(
			path,
			method,
			strconv.Itoa(status)).Observe(duration.Seconds())
	})
}
