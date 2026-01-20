package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/whywaita/shoes-vz/internal/agent/vm"
	"github.com/whywaita/shoes-vz/pkg/logging"
	"github.com/whywaita/shoes-vz/pkg/model"
)

func runExecCommand() {
	execFlags := flag.NewFlagSet("exec", flag.ExitOnError)
	runnersPath := execFlags.String("runners-path", "/opt/myshoes/vz/runners", "Path to runners directory")
	sshKeyPath := execFlags.String("ssh-key", "", "Path to SSH private key")

	if err := execFlags.Parse(os.Args[2:]); err != nil {
		logger := logging.WithComponent("agent")
		logger.Error("Failed to parse flags", "error", err)
		os.Exit(1)
	}

	logger := logging.WithComponent("agent")

	// Get runner ID and command from remaining args
	args := execFlags.Args()
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: runner-id and command are required\n")
		fmt.Fprintf(os.Stderr, "Usage: shoes-vz-agent exec [options] <runner-id> <command> [args...]\n")
		execFlags.PrintDefaults()
		os.Exit(1)
	}

	runnerID := args[0]
	command := args[1]
	cmdArgs := args[2:]

	// Create VM manager
	config := &model.AgentConfig{
		RunnersPath: *runnersPath,
		SSHKeyPath:  *sshKeyPath,
	}
	vmManager := vm.NewManager(config, nil)

	// Execute command
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	logger.Info("Executing command on VM",
		"runner_id", runnerID,
		"command", command,
		"args", strings.Join(cmdArgs, " "),
	)

	output, exitCode, err := vmManager.Exec(ctx, runnerID, command, cmdArgs)
	if err != nil {
		logger.Error("Failed to execute command",
			"error", err,
			"exit_code", exitCode,
		)
		// Print output even on error (might contain useful error messages)
		if len(output) > 0 {
			fmt.Fprintf(os.Stderr, "Output:\n%s\n", string(output))
		}
		os.Exit(exitCode)
	}

	// Print output
	fmt.Print(string(output))

	logger.Info("Command executed successfully",
		"runner_id", runnerID,
		"exit_code", exitCode,
	)
}
