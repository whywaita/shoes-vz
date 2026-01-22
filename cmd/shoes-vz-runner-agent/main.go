package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/whywaita/shoes-vz/internal/monitor"
	"github.com/whywaita/shoes-vz/pkg/logging"
	"github.com/whywaita/shoes-vz/pkg/model"
)

func main() {
	var (
		listenAddr = flag.String("listen", ":8080", "HTTP server listen address")
		runnerPath = flag.String("runner-path", "", "Path to GitHub Actions runner directory")
		runnerID   = flag.String("runner-id", "", "Runner ID for IP notification")
		hostIP     = flag.String("host-ip", "192.168.64.1", "Host IP for IP notification")
		agentPort  = flag.Int("agent-port", 8081, "shoes-vz-agent HTTP port")
	)
	flag.Parse()

	logger := logging.WithComponent("runner-agent")

	// Default runner path to user's home directory
	if *runnerPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			logger.Error("Failed to get home directory", "error", err)
			os.Exit(1)
		}
		*runnerPath = filepath.Join(home, "_work", "_runner")
	}

	// Check environment variable for runner ID
	if *runnerID == "" {
		*runnerID = os.Getenv("SHOES_VZ_RUNNER_ID")
	}

	logger.Info("Starting shoes-vz-runner-agent")
	logger.Info("Starting on TCP", "listen_addr", *listenAddr)
	logger.Info("Using runner path", "path", *runnerPath)

	// Get machine UUID from IOPlatformUUID (set by Virtualization Framework)
	runnerIDToUse := *runnerID
	if runnerIDToUse == "" {
		// Check environment variable
		runnerIDToUse = os.Getenv("SHOES_VZ_RUNNER_ID")
	}
	if runnerIDToUse == "" {
		// Get from ioreg (IOPlatformUUID)
		uuid, err := monitor.GetMachineUUID()
		if err != nil {
			logger.Error("Failed to get machine UUID", "error", err)
			os.Exit(1)
		}
		runnerIDToUse = uuid
		logger.Info("Using machine UUID as runner ID", "runner_id", runnerIDToUse)
	} else {
		logger.Info("Runner ID configured", "runner_id", runnerIDToUse)
	}

	// Start IP notification in the background
	go func() {
		// Send IP notification
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := monitor.NotifyIP(ctx, runnerIDToUse, *hostIP, *agentPort); err != nil {
			logger.Error("Failed to notify IP", "error", err)
		} else {
			logger.Info("Successfully notified IP to shoes-vz-agent")
		}
	}()

	config := &model.MonitorConfig{
		ListenAddr:   *listenAddr,
		RunnerPath:   *runnerPath,
		PollInterval: 10 * time.Second,
	}

	server := monitor.NewServer(config)
	if err := server.Start(); err != nil {
		logger.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
