package logging

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "test-request-id"

	ctx = WithRequestID(ctx, requestID)
	got := RequestIDFromContext(ctx)

	if got != requestID {
		t.Errorf("RequestIDFromContext() = %v, want %v", got, requestID)
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := RequestIDFromContext(ctx)

	if got != "" {
		t.Errorf("RequestIDFromContext() = %v, want empty string", got)
	}
}

func TestNewRequestID(t *testing.T) {
	id1 := NewRequestID()
	id2 := NewRequestID()

	if id1 == "" {
		t.Error("NewRequestID() returned empty string")
	}

	if id1 == id2 {
		t.Error("NewRequestID() returned duplicate IDs")
	}
}

func TestFromContext(t *testing.T) {
	logger := NewLogger()
	ctx := context.Background()
	requestID := "test-request-id"

	ctx = WithRequestID(ctx, requestID)
	loggerWithID := FromContext(ctx, logger)

	if loggerWithID == logger {
		t.Error("FromContext() should return a new logger with request_id")
	}
}

func TestFromContext_NoRequestID(t *testing.T) {
	logger := NewLogger()
	ctx := context.Background()

	loggerWithID := FromContext(ctx, logger)

	if loggerWithID != logger {
		t.Error("FromContext() should return the same logger when no request_id in context")
	}
}

func TestExtractOrGenerateRequestID(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  func() context.Context
		wantEmpty bool
	}{
		{
			name: "extract from metadata",
			setupCtx: func() context.Context {
				md := metadata.New(map[string]string{
					MetadataKeyRequestID: "existing-id",
				})
				return metadata.NewIncomingContext(context.Background(), md)
			},
			wantEmpty: false,
		},
		{
			name: "generate when missing",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			got := extractOrGenerateRequestID(ctx)

			if tt.wantEmpty && got != "" {
				t.Errorf("extractOrGenerateRequestID() = %v, want empty", got)
			}
			if !tt.wantEmpty && got == "" {
				t.Error("extractOrGenerateRequestID() returned empty string")
			}
		})
	}
}

func TestWithComponent(t *testing.T) {
	logger := WithComponent("test-component")
	if logger == nil {
		t.Error("WithComponent() returned nil")
	}
}

func TestDefault(t *testing.T) {
	logger := Default()
	if logger == nil {
		t.Error("Default() returned nil")
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Error("NewLogger() returned nil")
	}
}
