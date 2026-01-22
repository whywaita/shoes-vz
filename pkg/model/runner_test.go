package model

import (
	"testing"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
)

func TestIsTerminalState(t *testing.T) {
	tests := []struct {
		name  string
		state agentv1.RunnerState
		want  bool
	}{
		{
			name:  "error state",
			state: agentv1.RunnerState_RUNNER_STATE_ERROR,
			want:  true,
		},
		{
			name:  "tearing down state",
			state: agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
			want:  true,
		},
		{
			name:  "running state",
			state: agentv1.RunnerState_RUNNER_STATE_RUNNING,
			want:  false,
		},
		{
			name:  "creating state",
			state: agentv1.RunnerState_RUNNER_STATE_CREATING,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTerminalState(tt.state)
			if got != tt.want {
				t.Errorf("IsTerminalState(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestIsReadyState(t *testing.T) {
	tests := []struct {
		name  string
		state agentv1.RunnerState
		want  bool
	}{
		{
			name:  "ssh ready state",
			state: agentv1.RunnerState_RUNNER_STATE_SSH_READY,
			want:  true,
		},
		{
			name:  "running state",
			state: agentv1.RunnerState_RUNNER_STATE_RUNNING,
			want:  true,
		},
		{
			name:  "creating state",
			state: agentv1.RunnerState_RUNNER_STATE_CREATING,
			want:  false,
		},
		{
			name:  "booting state",
			state: agentv1.RunnerState_RUNNER_STATE_BOOTING,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsReadyState(tt.state)
			if got != tt.want {
				t.Errorf("IsReadyState(%v) = %v, want %v", tt.state, got, tt.want)
			}
		})
	}
}

func TestCanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from agentv1.RunnerState
		to   agentv1.RunnerState
		want bool
	}{
		{
			name: "creating to booting",
			from: agentv1.RunnerState_RUNNER_STATE_CREATING,
			to:   agentv1.RunnerState_RUNNER_STATE_BOOTING,
			want: true,
		},
		{
			name: "booting to ssh ready",
			from: agentv1.RunnerState_RUNNER_STATE_BOOTING,
			to:   agentv1.RunnerState_RUNNER_STATE_SSH_READY,
			want: true,
		},
		{
			name: "ssh ready to running",
			from: agentv1.RunnerState_RUNNER_STATE_SSH_READY,
			to:   agentv1.RunnerState_RUNNER_STATE_RUNNING,
			want: true,
		},
		{
			name: "running to tearing down",
			from: agentv1.RunnerState_RUNNER_STATE_RUNNING,
			to:   agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
			want: true,
		},
		{
			name: "creating to running (invalid)",
			from: agentv1.RunnerState_RUNNER_STATE_CREATING,
			to:   agentv1.RunnerState_RUNNER_STATE_RUNNING,
			want: false,
		},
		{
			name: "any to error",
			from: agentv1.RunnerState_RUNNER_STATE_BOOTING,
			to:   agentv1.RunnerState_RUNNER_STATE_ERROR,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanTransitionTo(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanTransitionTo(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
