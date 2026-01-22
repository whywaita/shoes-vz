package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	shoesv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/shoes/v1"
	grpcserver "github.com/whywaita/shoes-vz/internal/server/grpc"
	"github.com/whywaita/shoes-vz/internal/server/metrics"
	"github.com/whywaita/shoes-vz/internal/server/store"
	"github.com/whywaita/shoes-vz/pkg/logging"
)

func main() {
	var (
		grpcAddr    = flag.String("grpc-addr", ":50051", "gRPC server listen address")
		metricsAddr = flag.String("metrics-addr", ":9090", "Metrics server listen address")
	)
	flag.Parse()

	logger := logging.WithComponent("server")

	logger.Info("Starting shoes-vz-server",
		"grpc_addr", *grpcAddr,
		"metrics_addr", *metricsAddr,
	)

	// Create metrics
	m := metrics.NewMetrics()
	st := store.NewStore()
	collector := metrics.NewCollector(m, st)

	// Create gRPC server with store and metrics collector
	server := grpcserver.NewServer(st, collector, logger)

	// Start metrics collection loop
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go collector.Start(ctx, 10*time.Second)

	// Start metrics HTTP server
	metricsServer := &http.Server{
		Addr:    *metricsAddr,
		Handler: promhttp.Handler(),
	}

	go func() {
		logger.Info("Metrics server starting", "addr", *metricsAddr)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Metrics server error", "error", err)
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		logger.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(logging.UnaryServerInterceptor(logger)),
	)
	shoesv1.RegisterShoesServiceServer(grpcServer, server)
	agentv1.RegisterAgentServiceServer(grpcServer, server)

	go func() {
		logger.Info("gRPC server starting", "addr", *grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("Failed to serve", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Received shutdown signal")

	// Graceful shutdown
	logger.Info("Shutting down servers")

	// Stop metrics collection
	cancel()

	// Shutdown metrics server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Metrics server shutdown error", "error", err)
	}

	// Shutdown gRPC server with timeout
	// Use a goroutine with timeout instead of indefinite GracefulStop
	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		logger.Info("gRPC server stopped gracefully")
	case <-time.After(5 * time.Second):
		logger.Info("gRPC server shutdown timeout, forcing stop")
		grpcServer.Stop()
	}

	logger.Info("Server stopped")
}
