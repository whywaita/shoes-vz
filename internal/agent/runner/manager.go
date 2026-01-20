package runner

import (
	"context"
	"fmt"
	"sync"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	"github.com/whywaita/shoes-vz/pkg/model"
)

// Manager manages the lifecycle of runners on this agent
type Manager struct {
	mu      sync.RWMutex
	runners map[string]*model.RunnerInfo
}

// NewManager creates a new Manager instance
func NewManager() *Manager {
	return &Manager{
		runners: make(map[string]*model.RunnerInfo),
	}
}

// Create creates a new runner
func (m *Manager) Create(ctx context.Context, runnerID, runnerName, setupScript string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.runners[runnerID]; exists {
		return fmt.Errorf("runner %s already exists", runnerID)
	}

	runner := &model.RunnerInfo{
		ID:          runnerID,
		Name:        runnerName,
		State:       agentv1.RunnerState_RUNNER_STATE_CREATING,
		SetupScript: setupScript,
	}

	m.runners[runnerID] = runner
	return nil
}

// Get retrieves a runner by ID
func (m *Manager) Get(runnerID string) (*model.RunnerInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runner, exists := m.runners[runnerID]
	if !exists {
		return nil, model.ErrRunnerNotFound
	}

	return runner, nil
}

// List returns all runners
func (m *Manager) List() []*model.RunnerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	runners := make([]*model.RunnerInfo, 0, len(m.runners))
	for _, r := range m.runners {
		runners = append(runners, r)
	}

	return runners
}

// UpdateState updates the state of a runner
func (m *Manager) UpdateState(runnerID string, state agentv1.RunnerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	runner, exists := m.runners[runnerID]
	if !exists {
		return model.ErrRunnerNotFound
	}

	if !model.CanTransitionTo(runner.State, state) {
		return fmt.Errorf("%w: %s -> %s", model.ErrInvalidTransition, runner.State, state)
	}

	runner.State = state
	return nil
}

// UpdateGuestState updates the guest runner state
func (m *Manager) UpdateGuestState(runnerID string, guestState agentv1.GuestRunnerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	runner, exists := m.runners[runnerID]
	if !exists {
		return model.ErrRunnerNotFound
	}

	runner.GuestState = guestState
	return nil
}

// SetError sets an error for a runner
func (m *Manager) SetError(runnerID string, errMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	runner, exists := m.runners[runnerID]
	if !exists {
		return model.ErrRunnerNotFound
	}

	runner.State = agentv1.RunnerState_RUNNER_STATE_ERROR
	runner.ErrorMessage = errMsg
	return nil
}

// Delete removes a runner
func (m *Manager) Delete(runnerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.runners[runnerID]; !exists {
		return model.ErrRunnerNotFound
	}

	delete(m.runners, runnerID)
	return nil
}

// Count returns the number of runners
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.runners)
}
