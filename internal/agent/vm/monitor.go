package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
)

// MonitorStatus represents the status returned by runner-monitor
type MonitorStatus struct {
	State        agentv1.GuestRunnerState `json:"state"`
	RunnerName   string                   `json:"runner_name"`
	Repository   string                   `json:"repository"`
	Labels       []string                 `json:"labels"`
	Job          *JobInfo                 `json:"job,omitempty"`
	ErrorMessage string                   `json:"error_message,omitempty"`
}

// JobInfo contains information about the currently running job
type JobInfo struct {
	JobID        int64     `json:"job_id"`
	RunID        int64     `json:"run_id"`
	RunNumber    int       `json:"run_number"`
	WorkflowName string    `json:"workflow_name"`
	JobName      string    `json:"job_name"`
	StartedAt    time.Time `json:"started_at"`
}

// GetMonitorStatus gets the runner status via HTTP
func (m *vzManager) GetMonitorStatus(ctx context.Context, runnerID string) (*MonitorStatus, error) {
	// Load runtime metadata to get IP address
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	bundleConfig, err := LoadBundleConfig(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load bundle config: %w", err)
	}

	metadata, err := LoadRuntimeMetadata(bundleConfig.RuntimeMetadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load runtime metadata: %w", err)
	}

	if metadata.IPAddress == "" {
		return nil, fmt.Errorf("VM IP address not yet discovered")
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Send HTTP GET request to /status
	url := fmt.Sprintf("http://%s:%d/status", metadata.IPAddress, MonitorTCPPort)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	// Parse JSON response
	var status MonitorStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &status, nil
}
