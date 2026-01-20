package scheduler

import (
	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	"github.com/whywaita/shoes-vz/internal/server/store"
	"github.com/whywaita/shoes-vz/pkg/model"
)

// Scheduler selects the best agent for a new runner
type Scheduler interface {
	SelectAgent() (string, error)
}

// roundRobinScheduler implements a simple round-robin scheduler
type roundRobinScheduler struct {
	store *store.Store
}

// NewRoundRobinScheduler creates a new round-robin scheduler
func NewRoundRobinScheduler(s *store.Store) Scheduler {
	return &roundRobinScheduler{
		store: s,
	}
}

// SelectAgent selects an agent with available capacity
// Uses a simple strategy: select the agent with the most available capacity
func (s *roundRobinScheduler) SelectAgent() (string, error) {
	agents := s.store.GetOnlineAgents()
	if len(agents) == 0 {
		return "", model.ErrNoAvailableAgent
	}

	var (
		selectedAgent        *agentv1.Agent
		maxAvailableCapacity uint32
	)

	for _, agent := range agents {
		hasCapacity, err := s.store.HasCapacity(agent.AgentId)
		if err != nil || !hasCapacity {
			continue
		}

		currentRunners := s.store.GetRunnerCount(agent.AgentId)
		available := agent.Capacity.MaxRunners - uint32(currentRunners)

		if selectedAgent == nil || available > maxAvailableCapacity {
			selectedAgent = agent
			maxAvailableCapacity = available
		}
	}

	if selectedAgent == nil {
		return "", model.ErrNoAvailableAgent
	}

	return selectedAgent.AgentId, nil
}
