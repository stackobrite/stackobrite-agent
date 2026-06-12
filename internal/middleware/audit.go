package middleware

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func AuditLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}

		next.ServeHTTP(rec, r)

		duration := time.Since(start)
		reqID := r.Header.Get("X-Request-ID")

		log.Printf("[AUDIT] %s %s %d %s %s req_id=%s",
			r.Method,
			r.URL.Path,
			rec.status,
			duration.Round(time.Millisecond),
			r.RemoteAddr,
			reqID,
		)
	})
}
