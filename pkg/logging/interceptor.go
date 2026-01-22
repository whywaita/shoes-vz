package logging

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const MetadataKeyRequestID = "x-request-id"

// UnaryServerInterceptor returns a gRPC server interceptor for logging
func UnaryServerInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		// Extract or generate request ID
		requestID := extractOrGenerateRequestID(ctx)
		ctx = WithRequestID(ctx, requestID)

		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)

		// Log request
		logAttrs := []any{
			"method", info.FullMethod,
			"duration_ms", duration.Milliseconds(),
		}
		if err != nil {
			logAttrs = append(logAttrs, "error", err.Error())
		}

		FromContext(ctx, logger).Info("gRPC request", logAttrs...)

		return resp, err
	}
}

// UnaryClientInterceptor returns a gRPC client interceptor for logging
func UnaryClientInterceptor(logger *slog.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Ensure request ID in metadata
		requestID := RequestIDFromContext(ctx)
		if requestID == "" {
			requestID = NewRequestID()
			ctx = WithRequestID(ctx, requestID)
		}

		ctx = metadata.AppendToOutgoingContext(ctx, MetadataKeyRequestID, requestID)

		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		logAttrs := []any{
			"method", method,
			"duration_ms", duration.Milliseconds(),
		}
		if err != nil {
			logAttrs = append(logAttrs, "error", err.Error())
		}

		FromContext(ctx, logger).Info("gRPC client call", logAttrs...)

		return err
	}
}

func extractOrGenerateRequestID(ctx context.Context) string {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get(MetadataKeyRequestID); len(vals) > 0 {
			return vals[0]
		}
	}
	return NewRequestID()
}
