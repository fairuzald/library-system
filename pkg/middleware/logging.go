package middleware

import (
	"net"
	"net/http"
	"time"

	"github.com/fairuzald/library-system/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type RequestLogger struct {
	log *logger.Logger
}

func NewRequestLogger(log *logger.Logger) *RequestLogger {
	return &RequestLogger{
		log: log,
	}
}

func (l *RequestLogger) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := uuid.New().String()

		crw := &customResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		crw.Header().Set("X-Request-ID", requestID)

		next.ServeHTTP(crw, r)

		clientIP := extractClientIP(r)

		duration := time.Since(start)
		l.log.Info("HTTP request",
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.String("query", r.URL.RawQuery),
			zap.String("client_ip", clientIP),
			zap.String("user_agent", r.UserAgent()),
			zap.Int("status", crw.statusCode),
			zap.Duration("duration", duration),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)
	})
}

type customResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (crw *customResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

func extractClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	if xForwardedFor != "" {
		ips := net.ParseIP(xForwardedFor)
		if ips != nil {
			return ips.String()
		}
	}

	xRealIP := r.Header.Get("X-Real-IP")
	if xRealIP != "" {
		ip := net.ParseIP(xRealIP)
		if ip != nil {
			return ip.String()
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
