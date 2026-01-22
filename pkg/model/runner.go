package model

import (
	"time"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
)

// RunnerInfo contains runtime information about a runner
type RunnerInfo struct {
	ID           string
	Name         string
	AgentID      string
	State        agentv1.RunnerState
	GuestState   agentv1.GuestRunnerState
	IPAddress    string // Guest VM IP address for SSH
	CreatedAt    time.Time
	ErrorMessage string
	SetupScript  string
	BundlePath   string
	MachineID    string
}

// IsTerminalState returns true if the runner is in a terminal state
func IsTerminalState(state agentv1.RunnerState) bool {
	switch state {
	case agentv1.RunnerState_RUNNER_STATE_ERROR,
		agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN:
		return true
	default:
		return false
	}
}

// IsReadyState returns true if the runner is ready for use
func IsReadyState(state agentv1.RunnerState) bool {
	return state == agentv1.RunnerState_RUNNER_STATE_SSH_READY ||
		state == agentv1.RunnerState_RUNNER_STATE_RUNNING
}

// CanTransitionTo checks if a state transition is valid
func CanTransitionTo(from, to agentv1.RunnerState) bool {
	// Error state can be reached from any state
	if to == agentv1.RunnerState_RUNNER_STATE_ERROR {
		return true
	}

	// Define valid transitions
	validTransitions := map[agentv1.RunnerState][]agentv1.RunnerState{
		agentv1.RunnerState_RUNNER_STATE_UNSPECIFIED: {
			agentv1.RunnerState_RUNNER_STATE_CREATING,
		},
		agentv1.RunnerState_RUNNER_STATE_CREATING: {
			agentv1.RunnerState_RUNNER_STATE_BOOTING,
			agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
		},
		agentv1.RunnerState_RUNNER_STATE_BOOTING: {
			agentv1.RunnerState_RUNNER_STATE_SSH_READY,
			agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
		},
		agentv1.RunnerState_RUNNER_STATE_SSH_READY: {
			agentv1.RunnerState_RUNNER_STATE_RUNNING,
			agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
		},
		agentv1.RunnerState_RUNNER_STATE_RUNNING: {
			agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
		},
		agentv1.RunnerState_RUNNER_STATE_ERROR: {
			agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
		},
		agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN: {},
	}

	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}

	return false
}
