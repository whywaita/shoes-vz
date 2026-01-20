package metrics

import (
	"context"
	"log"
	"time"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	"github.com/whywaita/shoes-vz/internal/server/store"
)

// Collector collects metrics from the store
type Collector struct {
	metrics *Metrics
	store   *store.Store
}

// NewCollector creates a new metrics collector
func NewCollector(metrics *Metrics, store *store.Store) *Collector {
	return &Collector{
		metrics: metrics,
		store:   store,
	}
}

// Start starts the metrics collection loop
func (c *Collector) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Collect initial metrics
	c.Collect()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.Collect()
		}
	}
}

// Collect collects all metrics from the store
func (c *Collector) Collect() {
	c.collectRunnerMetrics()
	c.collectAgentMetrics()
	c.collectCapacityMetrics()
}

// collectRunnerMetrics collects runner-related metrics
func (c *Collector) collectRunnerMetrics() {
	runners := c.store.ListRunners()

	// Count runners by state
	stateCounts := make(map[agentv1.RunnerState]int)
	idleCount := 0
	busyCount := 0
	errorCount := 0

	for _, runner := range runners {
		stateCounts[runner.State]++

		// Count idle/busy runners
		if runner.GuestRunnerState == agentv1.GuestRunnerState_GUEST_RUNNER_STATE_IDLE {
			idleCount++
		} else if runner.GuestRunnerState == agentv1.GuestRunnerState_GUEST_RUNNER_STATE_RUNNING {
			busyCount++
		}

		// Count error runners
		if runner.State == agentv1.RunnerState_RUNNER_STATE_ERROR {
			errorCount++
		}
	}

	// Update metrics
	for state, count := range stateCounts {
		c.metrics.RunnersTotal.WithLabelValues(state.String()).Set(float64(count))
	}

	// Ensure all states are reported (even if count is 0)
	allStates := []agentv1.RunnerState{
		agentv1.RunnerState_RUNNER_STATE_CREATING,
		agentv1.RunnerState_RUNNER_STATE_BOOTING,
		agentv1.RunnerState_RUNNER_STATE_SSH_READY,
		agentv1.RunnerState_RUNNER_STATE_RUNNING,
		agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN,
		agentv1.RunnerState_RUNNER_STATE_ERROR,
	}

	for _, state := range allStates {
		if _, exists := stateCounts[state]; !exists {
			c.metrics.RunnersTotal.WithLabelValues(state.String()).Set(0)
		}
	}

	c.metrics.RunnersIdle.Set(float64(idleCount))
	c.metrics.RunnersBusy.Set(float64(busyCount))
	c.metrics.RunnerErrors.Set(float64(errorCount))
}

// collectAgentMetrics collects agent-related metrics
func (c *Collector) collectAgentMetrics() {
	agents := c.store.ListAgents()

	// Count agents by status
	statusCounts := make(map[agentv1.AgentStatus]int)
	onlineCount := 0

	for _, agent := range agents {
		statusCounts[agent.Status]++

		if agent.Status == agentv1.AgentStatus_AGENT_STATUS_ONLINE {
			onlineCount++
		}

		// Set per-agent metrics
		c.metrics.AgentsCapacityRunners.WithLabelValues(
			agent.AgentId,
			agent.Hostname,
		).Set(float64(agent.Capacity.MaxRunners))

		currentRunners := c.store.GetRunnerCount(agent.AgentId)
		c.metrics.AgentsCurrentRunners.WithLabelValues(
			agent.AgentId,
			agent.Hostname,
		).Set(float64(currentRunners))
	}

	// Update status counts
	for status, count := range statusCounts {
		c.metrics.AgentsTotal.WithLabelValues(status.String()).Set(float64(count))
	}

	// Ensure all statuses are reported
	allStatuses := []agentv1.AgentStatus{
		agentv1.AgentStatus_AGENT_STATUS_ONLINE,
		agentv1.AgentStatus_AGENT_STATUS_OFFLINE,
	}

	for _, status := range allStatuses {
		if _, exists := statusCounts[status]; !exists {
			c.metrics.AgentsTotal.WithLabelValues(status.String()).Set(0)
		}
	}

	c.metrics.AgentsOnline.Set(float64(onlineCount))
}

// collectCapacityMetrics collects capacity-related metrics
func (c *Collector) collectCapacityMetrics() {
	agents := c.store.ListAgents()
	runners := c.store.ListRunners()

	totalCapacity := uint32(0)
	currentRunners := len(runners)

	for _, agent := range agents {
		if agent.Status == agentv1.AgentStatus_AGENT_STATUS_ONLINE {
			totalCapacity += agent.Capacity.MaxRunners
		}
	}

	availableRunners := int(totalCapacity) - currentRunners
	if availableRunners < 0 {
		availableRunners = 0
	}

	utilizationRatio := 0.0
	if totalCapacity > 0 {
		utilizationRatio = float64(currentRunners) / float64(totalCapacity)
	}

	c.metrics.CapacityTotalRunners.Set(float64(totalCapacity))
	c.metrics.CapacityAvailableRunners.Set(float64(availableRunners))
	c.metrics.CapacityUtilizationRatio.Set(utilizationRatio)
}

// RecordRunnerStartup records runner startup duration
func (c *Collector) RecordRunnerStartup(duration time.Duration) {
	c.metrics.RunnerStartupDuration.Observe(duration.Seconds())
	log.Printf("Runner startup took %.2f seconds", duration.Seconds())
}

// RecordRunnerFailure records a runner failure
func (c *Collector) RecordRunnerFailure(reason string) {
	c.metrics.RunnerFailuresTotal.WithLabelValues(reason).Inc()
	log.Printf("Runner failure recorded: %s", reason)
}

// RecordAddInstanceRequest records an AddInstance request
func (c *Collector) RecordAddInstanceRequest(status string, duration time.Duration) {
	c.metrics.AddInstanceRequestsTotal.WithLabelValues(status).Inc()
	c.metrics.AddInstanceDuration.Observe(duration.Seconds())
}

// RecordDeleteInstanceRequest records a DeleteInstance request
func (c *Collector) RecordDeleteInstanceRequest(status string) {
	c.metrics.DeleteInstanceRequestsTotal.WithLabelValues(status).Inc()
}
