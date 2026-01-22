package model

import "errors"

var (
	// ErrRunnerNotFound is returned when a runner is not found
	ErrRunnerNotFound = errors.New("runner not found")

	// ErrAgentNotFound is returned when an agent is not found
	ErrAgentNotFound = errors.New("agent not found")

	// ErrNoAvailableAgent is returned when no agent has capacity
	ErrNoAvailableAgent = errors.New("no available agent")

	// ErrInvalidTransition is returned when a state transition is invalid
	ErrInvalidTransition = errors.New("invalid state transition")

	// ErrVMCreationFailed is returned when VM creation fails
	ErrVMCreationFailed = errors.New("VM creation failed")

	// ErrVMStartFailed is returned when VM start fails
	ErrVMStartFailed = errors.New("VM start failed")

	// ErrSSHTimeout is returned when SSH connection times out
	ErrSSHTimeout = errors.New("SSH connection timeout")

	// ErrSetupScriptFailed is returned when setup script execution fails
	ErrSetupScriptFailed = errors.New("setup script failed")

	// ErrTemplateNotFound is returned when template is not found
	ErrTemplateNotFound = errors.New("template not found")

	// ErrCloneFailed is returned when APFS clone fails
	ErrCloneFailed = errors.New("APFS clone failed")
)
