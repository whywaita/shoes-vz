package store

import (
	"fmt"
	"sync"
	"time"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	"github.com/whywaita/shoes-vz/pkg/model"
)

// Store manages the state of agents and runners
type Store struct {
	mu      sync.RWMutex
	agents  map[string]*agentv1.Agent
	runners map[string]*agentv1.Runner
	// Map runner ID to agent ID
	runnerToAgent map[string]string
	// Map cloud ID (from myshoes) to runner ID
	cloudIDToRunner map[string]string
}

// NewStore creates a new Store
func NewStore() *Store {
	return &Store{
		agents:          make(map[string]*agentv1.Agent),
		runners:         make(map[string]*agentv1.Runner),
		runnerToAgent:   make(map[string]string),
		cloudIDToRunner: make(map[string]string),
	}
}

// RegisterAgent registers a new agent
func (s *Store) RegisterAgent(agentID string, agent *agentv1.Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.agents[agentID] = agent
}

// GetAgent retrieves an agent by ID
func (s *Store) GetAgent(agentID string) (*agentv1.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return nil, model.ErrAgentNotFound
	}

	return agent, nil
}

// ListAgents returns all agents
func (s *Store) ListAgents() []*agentv1.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agents := make([]*agentv1.Agent, 0, len(s.agents))
	for _, a := range s.agents {
		agents = append(agents, a)
	}

	return agents
}

// UpdateAgentStatus updates an agent's status
func (s *Store) UpdateAgentStatus(agentID string, status agentv1.AgentStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return model.ErrAgentNotFound
	}

	agent.Status = status
	return nil
}

// UpdateAgentRunners updates the runners for an agent
func (s *Store) UpdateAgentRunners(agentID string, runners []*agentv1.Runner) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.agents[agentID]
	if !exists {
		return model.ErrAgentNotFound
	}

	// Build a set of runner IDs from the received list
	receivedRunnerIDs := make(map[string]bool)
	for _, r := range runners {
		receivedRunnerIDs[r.RunnerId] = true
	}

	// Find runners that belong to this agent but are not in the received list
	var runnersToDelete []string
	for runnerID, aID := range s.runnerToAgent {
		if aID == agentID && !receivedRunnerIDs[runnerID] {
			runnersToDelete = append(runnersToDelete, runnerID)
		}
	}

	// Delete stale runners
	for _, runnerID := range runnersToDelete {
		// Find and remove cloud ID mapping
		for cloudID, rID := range s.cloudIDToRunner {
			if rID == runnerID {
				delete(s.cloudIDToRunner, cloudID)
				break
			}
		}
		delete(s.runners, runnerID)
		delete(s.runnerToAgent, runnerID)
	}

	// Update runner information
	for _, r := range runners {
		s.runners[r.RunnerId] = r
		s.runnerToAgent[r.RunnerId] = agentID
	}

	return nil
}

// GetRunner retrieves a runner by ID
func (s *Store) GetRunner(runnerID string) (*agentv1.Runner, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runner, exists := s.runners[runnerID]
	if !exists {
		return nil, model.ErrRunnerNotFound
	}

	return runner, nil
}

// GetRunnerByCloudID retrieves a runner by cloud ID
func (s *Store) GetRunnerByCloudID(cloudID string) (*agentv1.Runner, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runnerID, exists := s.cloudIDToRunner[cloudID]
	if !exists {
		return nil, model.ErrRunnerNotFound
	}

	runner, exists := s.runners[runnerID]
	if !exists {
		return nil, model.ErrRunnerNotFound
	}

	return runner, nil
}

// GetAgentForRunner retrieves the agent managing a runner
func (s *Store) GetAgentForRunner(runnerID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentID, exists := s.runnerToAgent[runnerID]
	if !exists {
		return "", model.ErrRunnerNotFound
	}

	return agentID, nil
}

// ListRunners returns all runners
func (s *Store) ListRunners() []*agentv1.Runner {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runners := make([]*agentv1.Runner, 0, len(s.runners))
	for _, r := range s.runners {
		runners = append(runners, r)
	}

	return runners
}

// ListRunnersByAgent returns all runners for an agent
func (s *Store) ListRunnersByAgent(agentID string) []*agentv1.Runner {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runners []*agentv1.Runner
	for runnerID, aID := range s.runnerToAgent {
		if aID == agentID {
			if runner, exists := s.runners[runnerID]; exists {
				runners = append(runners, runner)
			}
		}
	}

	return runners
}

// RegisterCloudID associates a cloud ID with a runner ID
func (s *Store) RegisterCloudID(cloudID, runnerID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cloudIDToRunner[cloudID] = runnerID
}

// DeleteRunner removes a runner
func (s *Store) DeleteRunner(runnerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runners[runnerID]; !exists {
		return model.ErrRunnerNotFound
	}

	// Find and remove cloud ID mapping
	for cloudID, rID := range s.cloudIDToRunner {
		if rID == runnerID {
			delete(s.cloudIDToRunner, cloudID)
			break
		}
	}

	delete(s.runners, runnerID)
	delete(s.runnerToAgent, runnerID)
	return nil
}

// MarkAgentOffline marks an agent as offline
func (s *Store) MarkAgentOffline(agentID string, timeout time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return model.ErrAgentNotFound
	}

	agent.Status = agentv1.AgentStatus_AGENT_STATUS_OFFLINE
	return nil
}

// GetOnlineAgents returns all online agents
func (s *Store) GetOnlineAgents() []*agentv1.Agent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var agents []*agentv1.Agent
	for _, a := range s.agents {
		if a.Status == agentv1.AgentStatus_AGENT_STATUS_ONLINE {
			agents = append(agents, a)
		}
	}

	return agents
}

// GetRunnerCount returns the number of active runners on an agent
// Excludes runners in terminal states (ERROR, TEARING_DOWN)
func (s *Store) GetRunnerCount(agentID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for runnerID, aID := range s.runnerToAgent {
		if aID == agentID {
			runner, exists := s.runners[runnerID]
			if !exists {
				continue
			}
			// Exclude terminal states from count
			if runner.State == agentv1.RunnerState_RUNNER_STATE_ERROR ||
				runner.State == agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN {
				continue
			}
			count++
		}
	}

	return count
}

// HasCapacity checks if an agent has capacity for more runners
// Excludes runners in terminal states (ERROR, TEARING_DOWN) from count
func (s *Store) HasCapacity(agentID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agent, exists := s.agents[agentID]
	if !exists {
		return false, model.ErrAgentNotFound
	}

	if agent.Status != agentv1.AgentStatus_AGENT_STATUS_ONLINE {
		return false, fmt.Errorf("agent is offline")
	}

	currentCount := 0
	for runnerID, aID := range s.runnerToAgent {
		if aID == agentID {
			runner, exists := s.runners[runnerID]
			if !exists {
				continue
			}
			// Exclude terminal states from count
			if runner.State == agentv1.RunnerState_RUNNER_STATE_ERROR ||
				runner.State == agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN {
				continue
			}
			currentCount++
		}
	}

	return uint32(currentCount) < agent.Capacity.MaxRunners, nil
}
