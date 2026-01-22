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

func runStopCommand() {
	stopFlags := flag.NewFlagSet("stop", flag.ExitOnError)
	runnersPath := stopFlags.String("runners-path", "/opt/myshoes/vz/runners", "Path to runners directory")

	if err := stopFlags.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Get runner ID from remaining args
	args := stopFlags.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: runner-id is required\n")
		fmt.Fprintf(os.Stderr, "Usage: shoes-vz-agent stop [options] <runner-id>\n")
		stopFlags.PrintDefaults()
		os.Exit(1)
	}

	runnerID := args[0]

	// Create VM manager
	config := &model.AgentConfig{
		RunnersPath: *runnersPath,
	}
	vmManager := vm.NewManager(config, nil)

	// Stop VM
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	fmt.Printf("Stopping VM: %s\n", runnerID)
	if err := vmManager.Stop(ctx, runnerID); err != nil {
		log.Fatalf("Failed to stop VM: %v", err)
	}

	fmt.Printf("Successfully stopped VM: %s\n", runnerID)
}
