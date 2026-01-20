package store

import (
	"testing"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
)

func TestStore_RegisterAndGetAgent(t *testing.T) {
	s := NewStore()

	agent := &agentv1.Agent{
		AgentId:  "agent-1",
		Hostname: "test-host",
		Capacity: &agentv1.AgentCapacity{
			MaxRunners: 4,
		},
		Status: agentv1.AgentStatus_AGENT_STATUS_ONLINE,
	}

	s.RegisterAgent(agent.AgentId, agent)

	got, err := s.GetAgent(agent.AgentId)
	if err != nil {
		t.Fatalf("GetAgent() error = %v", err)
	}

	if got.AgentId != agent.AgentId {
		t.Errorf("GetAgent() AgentId = %v, want %v", got.AgentId, agent.AgentId)
	}

	if got.Hostname != agent.Hostname {
		t.Errorf("GetAgent() Hostname = %v, want %v", got.Hostname, agent.Hostname)
	}
}

func TestStore_HasCapacity(t *testing.T) {
	s := NewStore()

	agent := &agentv1.Agent{
		AgentId:  "agent-1",
		Hostname: "test-host",
		Capacity: &agentv1.AgentCapacity{
			MaxRunners: 2,
		},
		Status: agentv1.AgentStatus_AGENT_STATUS_ONLINE,
	}

	s.RegisterAgent(agent.AgentId, agent)

	// Initially, agent should have capacity
	hasCapacity, err := s.HasCapacity(agent.AgentId)
	if err != nil {
		t.Fatalf("HasCapacity() error = %v", err)
	}
	if !hasCapacity {
		t.Error("HasCapacity() = false, want true")
	}

	// Add runners up to capacity
	runners := []*agentv1.Runner{
		{
			RunnerId: "runner-1",
			AgentId:  agent.AgentId,
		},
		{
			RunnerId: "runner-2",
			AgentId:  agent.AgentId,
		},
	}

	if err := s.UpdateAgentRunners(agent.AgentId, runners); err != nil {
		t.Fatalf("UpdateAgentRunners() error = %v", err)
	}

	// Now agent should be at capacity
	hasCapacity, err = s.HasCapacity(agent.AgentId)
	if err != nil {
		t.Fatalf("HasCapacity() error = %v", err)
	}
	if hasCapacity {
		t.Error("HasCapacity() = true, want false")
	}
}

func TestStore_RegisterCloudID(t *testing.T) {
	s := NewStore()

	cloudID := "cloud-123"
	runnerID := "runner-123"

	runner := &agentv1.Runner{
		RunnerId: runnerID,
		AgentId:  "agent-1",
	}

	s.runners[runnerID] = runner
	s.RegisterCloudID(cloudID, runnerID)

	got, err := s.GetRunnerByCloudID(cloudID)
	if err != nil {
		t.Fatalf("GetRunnerByCloudID() error = %v", err)
	}

	if got.RunnerId != runnerID {
		t.Errorf("GetRunnerByCloudID() RunnerId = %v, want %v", got.RunnerId, runnerID)
	}
}

func TestStore_GetOnlineAgents(t *testing.T) {
	s := NewStore()

	agents := []*agentv1.Agent{
		{
			AgentId:  "agent-1",
			Hostname: "host-1",
			Status:   agentv1.AgentStatus_AGENT_STATUS_ONLINE,
		},
		{
			AgentId:  "agent-2",
			Hostname: "host-2",
			Status:   agentv1.AgentStatus_AGENT_STATUS_OFFLINE,
		},
		{
			AgentId:  "agent-3",
			Hostname: "host-3",
			Status:   agentv1.AgentStatus_AGENT_STATUS_ONLINE,
		},
	}

	for _, agent := range agents {
		s.RegisterAgent(agent.AgentId, agent)
	}

	onlineAgents := s.GetOnlineAgents()

	if len(onlineAgents) != 2 {
		t.Errorf("GetOnlineAgents() count = %v, want 2", len(onlineAgents))
	}

	for _, agent := range onlineAgents {
		if agent.Status != agentv1.AgentStatus_AGENT_STATUS_ONLINE {
			t.Errorf("GetOnlineAgents() contains offline agent: %v", agent.AgentId)
		}
	}
}

func TestStore_HasCapacity_ExcludesErrorRunners(t *testing.T) {
	s := NewStore()

	agent := &agentv1.Agent{
		AgentId:  "agent-1",
		Hostname: "test-host",
		Capacity: &agentv1.AgentCapacity{
			MaxRunners: 2,
		},
		Status: agentv1.AgentStatus_AGENT_STATUS_ONLINE,
	}

	s.RegisterAgent(agent.AgentId, agent)

	// Add one running runner and one ERROR runner
	runners := []*agentv1.Runner{
		{
			RunnerId: "runner-1",
			AgentId:  agent.AgentId,
			State:    agentv1.RunnerState_RUNNER_STATE_RUNNING,
		},
		{
			RunnerId: "runner-2",
			AgentId:  agent.AgentId,
			State:    agentv1.RunnerState_RUNNER_STATE_ERROR,
		},
	}

	if err := s.UpdateAgentRunners(agent.AgentId, runners); err != nil {
		t.Fatalf("UpdateAgentRunners() error = %v", err)
	}

	// Agent should still have capacity because ERROR runner is excluded
	hasCapacity, err := s.HasCapacity(agent.AgentId)
	if err != nil {
		t.Fatalf("HasCapacity() error = %v", err)
	}
	if !hasCapacity {
		t.Error("HasCapacity() = false, want true (ERROR runner should be excluded)")
	}

	// GetRunnerCount should return 1 (excluding ERROR runner)
	count := s.GetRunnerCount(agent.AgentId)
	if count != 1 {
		t.Errorf("GetRunnerCount() = %v, want 1 (ERROR runner should be excluded)", count)
	}
}
