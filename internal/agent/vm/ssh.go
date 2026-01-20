package vm

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/whywaita/shoes-vz/pkg/logging"
)

// waitForSSH waits until SSH is ready on the VM
func waitForSSH(ctx context.Context, runnerID, ipAddress string, keyPath string, timeout time.Duration) error {
	logger := logging.WithComponent("vm")

	if ipAddress == "" {
		return fmt.Errorf("IP address is empty")
	}

	logger.Info("Waiting for SSH", "runner_id", runnerID, "ip_address", ipAddress, "timeout", timeout)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	attemptCount := 0
	for {
		select {
		case <-ctx.Done():
			logger.Error("SSH wait timeout", "runner_id", runnerID, "ip_address", ipAddress, "attempts", attemptCount, "error", ctx.Err())
			return fmt.Errorf("SSH wait timeout after %d attempts: %w", attemptCount, ctx.Err())
		case <-ticker.C:
			attemptCount++
			if err := checkSSH(ipAddress, keyPath); err == nil {
				logger.Info("SSH ready", "runner_id", runnerID, "ip_address", ipAddress, "attempts", attemptCount)
				return nil
			} else {
				logger.Debug("SSH check failed, retrying", "runner_id", runnerID, "ip_address", ipAddress, "attempt", attemptCount, "error", err)
			}
		}
	}
}

// checkSSH checks if SSH is ready
func checkSSH(ipAddress string, keyPath string) error {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=1",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}

	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}

	// Connect to runner@<ip> on default SSH port (22)
	args = append(args, fmt.Sprintf("runner@%s", ipAddress), "true")

	cmd := exec.Command("ssh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("SSH check failed: %w (output: %s)", err, string(output))
	}

	return nil
}

// runSSHScript runs a script via SSH
func runSSHScript(ctx context.Context, runnerID, ipAddress string, keyPath, script string) error {
	logger := logging.WithComponent("vm")

	if ipAddress == "" {
		return fmt.Errorf("IP address is empty")
	}

	logger.Info("Running SSH script", "runner_id", runnerID, "ip_address", ipAddress, "script_length", len(script))

	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
	}

	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}

	args = append(args, fmt.Sprintf("runner@%s", ipAddress), script)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("SSH script execution failed", "runner_id", runnerID, "ip_address", ipAddress, "error", err, "output", string(output))
		return fmt.Errorf("SSH script execution failed: %w, output: %s", err, string(output))
	}

	logger.Info("SSH script completed", "runner_id", runnerID, "ip_address", ipAddress, "output_length", len(output))
	return nil
}
