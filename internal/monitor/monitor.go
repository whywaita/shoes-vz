package monitor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
)

const (
	runnerFile  = ".runner"
	serviceFile = ".service"
)

// RunnerStatus represents the status of the GitHub Actions runner
type RunnerStatus struct {
	State      agentv1.GuestRunnerState `json:"state"`
	RunnerName string                   `json:"runner_name,omitempty"`
	Repository string                   `json:"repository,omitempty"`
	Labels     []string                 `json:"labels,omitempty"`
	Job        *JobInfo                 `json:"job,omitempty"`
}

// JobInfo contains information about the running job
type JobInfo struct {
	JobID        string    `json:"job_id,omitempty"`
	RunID        string    `json:"run_id,omitempty"`
	RunNumber    string    `json:"run_number,omitempty"`
	WorkflowName string    `json:"workflow_name,omitempty"`
	JobName      string    `json:"job_name,omitempty"`
	StartedAt    time.Time `json:"started_at,omitempty"`
}

// Monitor monitors the GitHub Actions runner status
type Monitor struct {
	runnerPath string
}

// NewMonitor creates a new Monitor instance
func NewMonitor(runnerPath string) *Monitor {
	return &Monitor{
		runnerPath: runnerPath,
	}
}

// GetStatus returns the current status of the runner
func (m *Monitor) GetStatus() (*RunnerStatus, error) {
	status := &RunnerStatus{
		State: agentv1.GuestRunnerState_GUEST_RUNNER_STATE_OFFLINE,
	}

	// Check if .runner file exists
	runnerFilePath := filepath.Join(m.runnerPath, runnerFile)
	if _, err := os.Stat(runnerFilePath); os.IsNotExist(err) {
		return status, nil
	}

	// Read runner configuration
	runnerData, err := os.ReadFile(runnerFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read runner file: %w", err)
	}

	var runnerConfig struct {
		AgentName  string   `json:"agentName"`
		PoolID     int      `json:"poolId"`
		ServerURL  string   `json:"serverUrl"`
		GitHubURL  string   `json:"gitHubUrl"`
		WorkFolder string   `json:"workFolder"`
		Labels     []string `json:"labels"`
	}

	if err := json.Unmarshal(runnerData, &runnerConfig); err != nil {
		return nil, fmt.Errorf("failed to parse runner file: %w", err)
	}

	status.RunnerName = runnerConfig.AgentName
	status.Labels = runnerConfig.Labels

	// Extract repository from GitHubURL
	// GitHubURL format: https://github.com/owner/repo
	if runnerConfig.GitHubURL != "" {
		parts := strings.Split(runnerConfig.GitHubURL, "/")
		if len(parts) >= 2 {
			status.Repository = strings.Join(parts[len(parts)-2:], "/")
		}
	}

	// Check runner process
	isRunning, err := m.isRunnerProcessRunning()
	if err != nil {
		return nil, fmt.Errorf("failed to check runner process: %w", err)
	}

	if !isRunning {
		status.State = agentv1.GuestRunnerState_GUEST_RUNNER_STATE_OFFLINE
		return status, nil
	}

	// Check if job is running
	isJobRunning, jobInfo, err := m.isJobRunning()
	if err != nil {
		return nil, fmt.Errorf("failed to check job status: %w", err)
	}

	if isJobRunning {
		status.State = agentv1.GuestRunnerState_GUEST_RUNNER_STATE_RUNNING
		status.Job = jobInfo
	} else {
		status.State = agentv1.GuestRunnerState_GUEST_RUNNER_STATE_IDLE
	}

	return status, nil
}

// isRunnerProcessRunning checks if the runner process is running
func (m *Monitor) isRunnerProcessRunning() (bool, error) {
	// Check for Runner.Listener or Runner.Worker process
	// This is a simplified check - in production, you'd use process enumeration
	serviceFilePath := filepath.Join(m.runnerPath, serviceFile)
	_, err := os.Stat(serviceFilePath)
	return err == nil, nil
}

// isJobRunning checks if a job is currently running
func (m *Monitor) isJobRunning() (bool, *JobInfo, error) {
	// Check for environment variables that indicate a job is running
	// These are set by the GitHub Actions runner when a job starts
	jobID := os.Getenv("GITHUB_JOB")
	runID := os.Getenv("GITHUB_RUN_ID")

	if jobID == "" || runID == "" {
		return false, nil, nil
	}

	jobInfo := &JobInfo{
		JobID:        jobID,
		RunID:        runID,
		RunNumber:    os.Getenv("GITHUB_RUN_NUMBER"),
		WorkflowName: os.Getenv("GITHUB_WORKFLOW"),
		JobName:      os.Getenv("GITHUB_JOB"),
		StartedAt:    time.Now(), // Approximate - would need to track actual start time
	}

	return true, jobInfo, nil
}
