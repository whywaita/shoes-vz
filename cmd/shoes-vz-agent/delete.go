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

func runDeleteCommand() {
	deleteFlags := flag.NewFlagSet("delete", flag.ExitOnError)
	runnersPath := deleteFlags.String("runners-path", "/opt/myshoes/vz/runners", "Path to runners directory")

	if err := deleteFlags.Parse(os.Args[2:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	// Get runner ID from remaining args
	args := deleteFlags.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: runner-id is required\n")
		fmt.Fprintf(os.Stderr, "Usage: shoes-vz-agent delete [options] <runner-id>\n")
		deleteFlags.PrintDefaults()
		os.Exit(1)
	}

	runnerID := args[0]

	// Create VM manager
	config := &model.AgentConfig{
		RunnersPath: *runnersPath,
	}
	vmManager := vm.NewManager(config, nil)

	// Delete VM
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Printf("Deleting VM: %s\n", runnerID)
	if err := vmManager.Delete(ctx, runnerID); err != nil {
		log.Fatalf("Failed to delete VM: %v", err)
	}

	fmt.Printf("Successfully deleted VM: %s\n", runnerID)
}
