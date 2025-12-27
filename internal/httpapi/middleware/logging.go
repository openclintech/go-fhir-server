package middleware

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func Logging(l *log.Logger) func(http.Handler) http.Handler {
	if l == nil {
		l = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}

			next.ServeHTTP(rec, r)

			reqID := GetRequestID(r.Context())
			l.Printf("method=%s path=%s status=%d bytes=%d dur_ms=%d request_id=%s",
				r.Method,
				r.URL.Path,
				rec.status,
				rec.bytes,
				time.Since(start).Milliseconds(),
				reqID,
			)
		})
	}
}
