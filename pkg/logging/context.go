package logging

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID adds request_id to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts request_id from context
func RequestIDFromContext(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// NewRequestID generates a new unique request ID
func NewRequestID() string {
	return uuid.New().String()
}

// FromContext returns a logger with request_id from context
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		return base.With("request_id", requestID)
	}
	return base
}
