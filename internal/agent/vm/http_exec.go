package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ExecRequest represents a command execution request for runner-agent
type ExecRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// ExecResponse represents a command execution response from runner-agent
type ExecResponse struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// execViaHTTP executes a command on the VM via HTTP using runner-agent
func execViaHTTP(ctx context.Context, ipAddress string, port int, command string, args []string) ([]byte, int, error) {
	// Create HTTP client
	client := &http.Client{
		Timeout: 2 * time.Minute,
	}

	// Prepare exec request
	execReq := ExecRequest{
		Command: command,
		Args:    args,
	}
	reqBody, err := json.Marshal(execReq)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send HTTP POST request to /exec
	url := fmt.Sprintf("http://%s:%d/exec", ipAddress, port)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, -1, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, -1, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var execResp ExecResponse
	if err := json.NewDecoder(resp.Body).Decode(&execResp); err != nil {
		return nil, -1, fmt.Errorf("failed to decode response: %w", err)
	}

	if execResp.Error != "" {
		return []byte(execResp.Output), execResp.ExitCode, fmt.Errorf("command execution failed: %s", execResp.Error)
	}

	return []byte(execResp.Output), execResp.ExitCode, nil
}
