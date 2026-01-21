package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultHostIP    = "192.168.64.1"
	DefaultAgentPort = 8081
)

// NotifyIP notifies the shoes-vz-agent of this runner's IP address.
// It retries every 2 seconds until successful or the context is canceled.
func NotifyIP(ctx context.Context, runnerID, hostIP string, agentPort int) error {
	ownIP, err := getOwnIP()
	if err != nil {
		return fmt.Errorf("failed to get own IP: %w", err)
	}

	notification := map[string]string{
		"runner_id":  runnerID,
		"ip_address": ownIP,
	}

	body, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/notify-ip", hostIP, agentPort)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled: %w", ctx.Err())
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				continue
			}

			_ = resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}

// getOwnIP returns the IP address of this machine in the 192.168.64.0/24 subnet.
func getOwnIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			// Check if IP is in 192.168.64.0/24 subnet
			if ip[0] == 192 && ip[1] == 168 && ip[2] == 64 {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no IP address found in 192.168.64.0/24 subnet")
}

// RunnerConfig represents the structure of .runner file
type RunnerConfig struct {
	AgentID   int    `json:"agentId"`
	AgentName string `json:"agentName"`
	PoolID    int    `json:"poolId"`
	PoolName  string `json:"poolName"`
}

// LoadRunnerIDFromFile loads the runner ID (agentName) from .runner file
// Returns empty string if file doesn't exist or can't be parsed
func LoadRunnerIDFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var config RunnerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}

	return config.AgentName
}

// GetMachineUUID retrieves the IOPlatformUUID from the system using ioreg.
// This UUID is set by the Virtualization Framework's MachineIdentifier.
func GetMachineUUID() (string, error) {
	// Run ioreg command to get IOPlatformUUID
	cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run ioreg: %w", err)
	}

	// Parse output to extract IOPlatformUUID
	// Expected format: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
	re := regexp.MustCompile(`"IOPlatformUUID"\s*=\s*"([0-9A-Fa-f-]+)"`)
	matches := re.FindSubmatch(output)
	if len(matches) < 2 {
		return "", fmt.Errorf("IOPlatformUUID not found in ioreg output")
	}

	uuid := strings.ToLower(string(matches[1]))
	return uuid, nil
}
