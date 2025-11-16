package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(logger *slog.Logger) *Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &Logger{logger}
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	code int
}

func (wrw *wrappedResponseWriter) Flush() {
	if flusher, ok := wrw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (wrw *wrappedResponseWriter) WriteHeader(code int) {
	wrw.code = code
	wrw.ResponseWriter.WriteHeader(code)
}

func (l *Logger) Handle(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sse" {
			next.ServeHTTP(w, r)
			return
		}
		wrw := &wrappedResponseWriter{
			ResponseWriter: w,
		}
		t0 := time.Now()
		next.ServeHTTP(wrw, r)
		if wrw.code < 400 {
			l.logger.InfoContext(r.Context(), fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto), "remote_addr", r.RemoteAddr, "code", wrw.code, "took", time.Since(t0))
		} else if wrw.code < 500 {
			l.logger.WarnContext(r.Context(), fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto), "remote_addr", r.RemoteAddr, "code", wrw.code, "took", time.Since(t0))
		} else {
			l.logger.ErrorContext(r.Context(), fmt.Sprintf("%s %s %s", r.Method, r.RequestURI, r.Proto), "remote_addr", r.RemoteAddr, "code", wrw.code, "took", time.Since(t0))
		}
	}
}
