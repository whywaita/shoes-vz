package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/whywaita/shoes-vz/internal/agent/vm"
	"github.com/whywaita/shoes-vz/pkg/model"
)

func runStartCommand() {
	startFlags := flag.NewFlagSet("start", flag.ExitOnError)
	runnersPath := startFlags.String("runners-path", "/opt/myshoes/vz/runners", "Path to runners directory")

	if err := startFlags.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Get runner ID from remaining args
	args := startFlags.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: runner-id is required\n")
		fmt.Fprintf(os.Stderr, "Usage: shoes-vz-agent start [options] <runner-id>\n")
		startFlags.PrintDefaults()
		os.Exit(1)
	}

	runnerID := args[0]

	// Create VM manager
	config := &model.AgentConfig{
		RunnersPath: *runnersPath,
	}
	// Note: IP notification server is not needed for start command
	// as it only starts already created VMs
	vmManager := vm.NewManager(config, nil)

	// Start VM
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Printf("Starting VM: %s\n", runnerID)
	ipAddress, err := vmManager.Start(ctx, runnerID)
	if err != nil {
		log.Fatalf("Failed to start VM: %v", err)
	}

	fmt.Printf("Successfully started VM: %s (IP: %s)\n", runnerID, ipAddress)
}
