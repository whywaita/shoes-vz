package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	"github.com/whywaita/shoes-vz/internal/agent/ipnotify"
	"github.com/whywaita/shoes-vz/internal/agent/runner"
	"github.com/whywaita/shoes-vz/internal/agent/sync"
	"github.com/whywaita/shoes-vz/internal/agent/vm"
	"github.com/whywaita/shoes-vz/pkg/logging"
	"github.com/whywaita/shoes-vz/pkg/model"
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			runListCommand()
			return
		case "delete":
			runDeleteCommand()
			return
		case "stop":
			runStopCommand()
			return
		case "start":
			runStartCommand()
			return
		case "exec":
			runExecCommand()
			return
		case "run":
			// Explicit "run" subcommand
			// Remove "run" from args and continue to runAgentCommand
			os.Args = append(os.Args[:1], os.Args[2:]...)
		case "-h", "--help", "help":
			printUsage()
			return
		default:
			// If it starts with -, treat as flag and run agent
			if os.Args[1][0] != '-' {
				fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n", os.Args[1])
				printUsage()
				os.Exit(1)
			}
		}
	}

	// Default: run agent
	runAgentCommand()
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: shoes-vz-agent [command] [options]

Commands:
  run         Run the agent (default)
  list        List all VMs in the runners directory
  start       Start a stopped VM
  stop        Stop a running VM
  delete      Delete a VM and its bundle
  exec        Execute a command on a VM via SSH
  help        Show this help message

Run Options:
`)
	flag.PrintDefaults()
}

func runAgentCommand() {
	var (
		serverAddr     = flag.String("server", "localhost:50051", "Server gRPC address")
		hostname       = flag.String("hostname", "", "Agent hostname (default: system hostname)")
		maxRunners     = flag.Uint("max-runners", 2, "Maximum number of concurrent runners (max: 2)")
		templatePath   = flag.String("template-path", "/opt/myshoes/vz/templates/macos-26", "Path to VM template")
		runnersPath    = flag.String("runners-path", "/opt/myshoes/vz/runners", "Path to runners directory")
		sshKeyPath     = flag.String("ssh-key", "", "Path to SSH private key")
		ipNotifyPort   = flag.Uint("ip-notify-port", 8081, "Port for IP notification HTTP server")
		enableGraphics = flag.Bool("enable-graphics", false, "Enable graphics display for VMs (opens GUI window)")
	)
	flag.Parse()

	logger := logging.WithComponent("agent")

	// Validate max-runners
	if *maxRunners > 2 {
		logger.Error("max-runners must be 2 or less", "specified", *maxRunners)
		os.Exit(1)
	}
	if *maxRunners == 0 {
		logger.Error("max-runners must be at least 1", "specified", *maxRunners)
		os.Exit(1)
	}

	// Get hostname if not provided
	if *hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			logger.Error("Failed to get hostname", "error", err)
			os.Exit(1)
		}
		*hostname = h
	}

	logger.Info("Starting shoes-vz-agent",
		"server", *serverAddr,
		"hostname", *hostname,
		"max_runners", *maxRunners,
		"template_path", *templatePath,
		"runners_path", *runnersPath,
	)

	// Create agent configuration
	config := &model.AgentConfig{
		ServerAddr:     *serverAddr,
		Hostname:       *hostname,
		MaxRunners:     uint32(*maxRunners),
		TemplatePath:   *templatePath,
		RunnersPath:    *runnersPath,
		SSHKeyPath:     *sshKeyPath,
		SyncInterval:   5 * time.Second,
		EnableGraphics: *enableGraphics,
	}

	// Create IP notification server
	ipNotifyServer := ipnotify.NewServer(int(*ipNotifyPort))
	if err := ipNotifyServer.Start(); err != nil {
		logger.Error("Failed to start IP notification server", "error", err)
		os.Exit(1)
	}
	defer func() {
		ctx := context.Background()
		if err := ipNotifyServer.Stop(ctx); err != nil {
			logger.Error("Failed to stop IP notification server", "error", err)
		}
	}()

	// Create components
	runnerManager := runner.NewManager()
	vmManager := vm.NewManager(config, ipNotifyServer)
	syncClient := sync.NewClient(
		config.ServerAddr,
		config.SyncInterval,
		runnerManager,
		vmManager,
		logger,
	)

	// Connect to server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	capacity := &agentv1.AgentCapacity{
		MaxRunners:  config.MaxRunners,
		CpuCores:    uint32(runtime.NumCPU()),
		MemoryBytes: 0, // TODO: Get actual memory size
	}

	if err := syncClient.Connect(ctx, config.Hostname, capacity); err != nil {
		logger.Error("Failed to connect to server", "error", err)
		os.Exit(1)
	}
	defer syncClient.Close()

	// Start sync loop
	go func() {
		if err := syncClient.Start(ctx); err != nil {
			logger.Error("Sync loop error", "error", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sigChan:
		logger.Info("Received shutdown signal")
	case <-ctx.Done():
		logger.Info("Context cancelled")
	}

	logger.Info("Shutting down agent")
}
