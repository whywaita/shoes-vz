package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	shoesv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/shoes/v1"
	"github.com/whywaita/shoes-vz/internal/server/metrics"
	"github.com/whywaita/shoes-vz/internal/server/scheduler"
	"github.com/whywaita/shoes-vz/internal/server/store"
	"github.com/whywaita/shoes-vz/pkg/logging"
)

// Server implements both ShoesService and AgentService
type Server struct {
	shoesv1.UnimplementedShoesServiceServer
	agentv1.UnimplementedAgentServiceServer

	store            *store.Store
	scheduler        scheduler.Scheduler
	metricsCollector *metrics.Collector
	logger           *slog.Logger

	// Map of agent ID to sync stream
	mu      sync.RWMutex
	streams map[string]agentv1.AgentService_SyncServer

	// Pending commands for agents
	commandMu       sync.Mutex
	pendingCommands map[string][]*agentv1.SyncResponse

	// Track runner creation times for metrics
	runnerCreationTimes sync.Map // map[runnerID]time.Time
}

// NewServer creates a new gRPC server
func NewServer(st *store.Store, metricsCollector *metrics.Collector, logger *slog.Logger) *Server {
	s := &Server{
		store:            st,
		scheduler:        scheduler.NewRoundRobinScheduler(st),
		metricsCollector: metricsCollector,
		logger:           logger,
		streams:          make(map[string]agentv1.AgentService_SyncServer),
		pendingCommands:  make(map[string][]*agentv1.SyncResponse),
	}

	// Start background cleanup goroutine
	go s.cleanupErrorRunners()

	return s
}

// AddInstance implements ShoesService.AddInstance
func (s *Server) AddInstance(ctx context.Context, req *shoesv1.AddInstanceRequest) (*shoesv1.AddInstanceResponse, error) {
	startTime := time.Now()
	requestID := logging.RequestIDFromContext(ctx)
	logger := logging.FromContext(ctx, s.logger)

	logger.Info("AddInstance started",
		"runner_name", req.RunnerName,
		"resource_type", req.ResourceType,
	)

	// Select an agent
	agentID, err := s.scheduler.SelectAgent()
	if err != nil {
		s.metricsCollector.RecordAddInstanceRequest("failed_no_agent", time.Since(startTime))
		logger.Error("No available agent", "error", err)
		return nil, status.Errorf(codes.Unavailable, "no available agent: %v", err)
	}

	// Generate runner ID
	runnerID := uuid.New().String()
	cloudID := fmt.Sprintf("shoes-vz-%s", runnerID)

	logger.Info("Creating runner",
		"runner_id", runnerID,
		"cloud_id", cloudID,
		"agent_id", agentID,
	)

	// Track creation time for startup duration metrics
	s.runnerCreationTimes.Store(runnerID, startTime)

	// Register cloud ID mapping
	s.store.RegisterCloudID(cloudID, runnerID)

	// Create runner command
	cmd := &agentv1.SyncResponse{
		Command: &agentv1.SyncResponse_CreateRunner{
			CreateRunner: &agentv1.CreateRunnerCommand{
				RunnerId:    runnerID,
				RunnerName:  req.RunnerName,
				SetupScript: req.SetupScript,
				RequestId:   requestID,
			},
		},
	}

	// Send command to agent
	if err := s.sendCommandToAgent(agentID, cmd); err != nil {
		s.metricsCollector.RecordAddInstanceRequest("failed_send_command", time.Since(startTime))
		s.metricsCollector.RecordRunnerFailure("send_command_failed")
		return nil, status.Errorf(codes.Internal, "failed to send command to agent: %v", err)
	}

	// Wait for runner to reach SSH_READY state
	if err := s.waitForRunnerState(ctx, runnerID, agentv1.RunnerState_RUNNER_STATE_SSH_READY, 5*time.Minute); err != nil {
		s.metricsCollector.RecordAddInstanceRequest("failed_timeout", time.Since(startTime))
		s.metricsCollector.RecordRunnerFailure("startup_timeout")
		return nil, status.Errorf(codes.Internal, "runner failed to start: %v", err)
	}

	// Record startup duration
	if creationTime, ok := s.runnerCreationTimes.Load(runnerID); ok {
		startupDuration := time.Since(creationTime.(time.Time))
		s.metricsCollector.RecordRunnerStartup(startupDuration)
		s.runnerCreationTimes.Delete(runnerID)
	}

	// Get runner info
	runner, err := s.store.GetRunner(runnerID)
	if err != nil {
		s.metricsCollector.RecordAddInstanceRequest("failed_get_runner", time.Since(startTime))
		return nil, status.Errorf(codes.Internal, "failed to get runner info: %v", err)
	}

	s.metricsCollector.RecordAddInstanceRequest("success", time.Since(startTime))

	return &shoesv1.AddInstanceResponse{
		CloudId:   cloudID,
		ShoesType: "shoes-vz",
		IpAddress: runner.IpAddress,
	}, nil
}

// DeleteInstance implements ShoesService.DeleteInstance
func (s *Server) DeleteInstance(ctx context.Context, req *shoesv1.DeleteInstanceRequest) (*shoesv1.DeleteInstanceResponse, error) {
	requestID := logging.RequestIDFromContext(ctx)
	logger := logging.FromContext(ctx, s.logger)

	logger.Info("DeleteInstance started", "cloud_id", req.CloudId)

	// Get runner by cloud ID
	runner, err := s.store.GetRunnerByCloudID(req.CloudId)
	if err != nil {
		s.metricsCollector.RecordDeleteInstanceRequest("failed_not_found")
		logger.Error("Runner not found", "cloud_id", req.CloudId, "error", err)
		return nil, status.Errorf(codes.NotFound, "runner not found: %v", err)
	}

	// Get agent for runner
	agentID, err := s.store.GetAgentForRunner(runner.RunnerId)
	if err != nil {
		s.metricsCollector.RecordDeleteInstanceRequest("failed_get_agent")
		logger.Error("Failed to get agent", "runner_id", runner.RunnerId, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to get agent: %v", err)
	}

	logger.Info("Deleting runner",
		"runner_id", runner.RunnerId,
		"agent_id", agentID,
	)

	// Create delete command
	cmd := &agentv1.SyncResponse{
		Command: &agentv1.SyncResponse_DeleteRunner{
			DeleteRunner: &agentv1.DeleteRunnerCommand{
				RunnerId:  runner.RunnerId,
				RequestId: requestID,
			},
		},
	}

	// Send command to agent
	if err := s.sendCommandToAgent(agentID, cmd); err != nil {
		s.metricsCollector.RecordDeleteInstanceRequest("failed_send_command")
		logger.Error("Failed to send command to agent", "agent_id", agentID, "error", err)
		return nil, status.Errorf(codes.Internal, "failed to send command to agent: %v", err)
	}

	// Wait for runner to be deleted
	if err := s.waitForRunnerDeletion(ctx, runner.RunnerId, 2*time.Minute); err != nil {
		logger.Warn("Failed to wait for runner deletion", "runner_id", runner.RunnerId, "error", err)
	}

	s.metricsCollector.RecordDeleteInstanceRequest("success")
	logger.Info("DeleteInstance completed", "runner_id", runner.RunnerId)

	return &shoesv1.DeleteInstanceResponse{}, nil
}

// RegisterAgent implements AgentService.RegisterAgent
func (s *Server) RegisterAgent(ctx context.Context, req *agentv1.RegisterAgentRequest) (*agentv1.RegisterAgentResponse, error) {
	agentID := uuid.New().String()

	agent := &agentv1.Agent{
		AgentId:  agentID,
		Hostname: req.Hostname,
		Capacity: req.Capacity,
		Status:   agentv1.AgentStatus_AGENT_STATUS_ONLINE,
	}

	s.store.RegisterAgent(agentID, agent)

	s.logger.Info("Agent registered",
		"agent_id", agentID,
		"hostname", req.Hostname,
		"max_runners", req.Capacity.MaxRunners,
	)

	return &agentv1.RegisterAgentResponse{
		AgentId:             agentID,
		SyncIntervalSeconds: 5,
	}, nil
}

// Sync implements AgentService.Sync
func (s *Server) Sync(stream agentv1.AgentService_SyncServer) error {
	var agentID string

	for {
		req, err := stream.Recv()
		if err != nil {
			if agentID != "" {
				s.logger.Info("Agent stream closed", "agent_id", agentID, "error", err)
				s.removeAgentStream(agentID)
				if updateErr := s.store.UpdateAgentStatus(agentID, agentv1.AgentStatus_AGENT_STATUS_OFFLINE); updateErr != nil {
					s.logger.Error("Failed to update agent status", "agent_id", agentID, "error", updateErr)
				}
			}
			return err
		}

		// First message should contain agent ID
		if agentID == "" {
			agentID = req.AgentId
			s.setAgentStream(agentID, stream)
			s.logger.Info("Agent connected", "agent_id", agentID)
		}

		// Update agent status
		if err := s.store.UpdateAgentStatus(agentID, agentv1.AgentStatus_AGENT_STATUS_ONLINE); err != nil {
			s.logger.Error("Failed to update agent status", "agent_id", agentID, "error", err)
		}

		// Update runners
		if err := s.store.UpdateAgentRunners(agentID, req.Runners); err != nil {
			s.logger.Error("Failed to update agent runners",
				"agent_id", agentID,
				"error", err,
			)
		}

		// Send pending commands or noop
		resp := s.getNextCommand(agentID)
		if err := stream.Send(resp); err != nil {
			s.logger.Error("Failed to send response to agent",
				"agent_id", agentID,
				"error", err,
			)
			return err
		}
	}
}

// sendCommandToAgent sends a command to an agent
func (s *Server) sendCommandToAgent(agentID string, cmd *agentv1.SyncResponse) error {
	s.commandMu.Lock()
	defer s.commandMu.Unlock()

	// Queue the command
	s.pendingCommands[agentID] = append(s.pendingCommands[agentID], cmd)
	return nil
}

// getNextCommand gets the next command for an agent
func (s *Server) getNextCommand(agentID string) *agentv1.SyncResponse {
	s.commandMu.Lock()
	defer s.commandMu.Unlock()

	commands := s.pendingCommands[agentID]
	if len(commands) == 0 {
		return &agentv1.SyncResponse{
			Command: &agentv1.SyncResponse_Noop{
				Noop: &agentv1.NoopCommand{},
			},
		}
	}

	// Pop first command
	cmd := commands[0]
	s.pendingCommands[agentID] = commands[1:]
	return cmd
}

// setAgentStream registers a stream for an agent
func (s *Server) setAgentStream(agentID string, stream agentv1.AgentService_SyncServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[agentID] = stream
}

// removeAgentStream removes a stream for an agent
func (s *Server) removeAgentStream(agentID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.streams, agentID)
}

// waitForRunnerState waits for a runner to reach a specific state
func (s *Server) waitForRunnerState(ctx context.Context, runnerID string, targetState agentv1.RunnerState, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			runner, err := s.store.GetRunner(runnerID)
			if err != nil {
				continue
			}

			if runner.State == targetState {
				return nil
			}

			if runner.State == agentv1.RunnerState_RUNNER_STATE_ERROR {
				return fmt.Errorf("runner entered error state: %s", runner.ErrorMessage)
			}
		}
	}
}

// waitForRunnerDeletion waits for a runner to be deleted
func (s *Server) waitForRunnerDeletion(ctx context.Context, runnerID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, err := s.store.GetRunner(runnerID)
			if err != nil {
				// Runner not found means it was deleted
				return nil
			}
		}
	}
}

// GetStore returns the store (for testing/metrics)
func (s *Server) GetStore() *store.Store {
	return s.store
}

// cleanupErrorRunners periodically cleans up runners in ERROR state
func (s *Server) cleanupErrorRunners() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		runners := s.store.ListRunners()
		for _, runner := range runners {
			// Clean up runners that have been in ERROR state for more than 5 minutes
			if runner.State == agentv1.RunnerState_RUNNER_STATE_ERROR {
				// Check if runner has been in error state for a while
				// by checking if it's been at least 5 minutes since creation
				if time.Since(runner.CreatedAt.AsTime()) > 5*time.Minute {
					s.logger.Info("Cleaning up ERROR state runner",
						"runner_id", runner.RunnerId,
						"error_message", runner.ErrorMessage,
					)

					// Get agent for runner
					agentID, err := s.store.GetAgentForRunner(runner.RunnerId)
					if err != nil {
						s.logger.Warn("Failed to get agent for runner, removing from store",
							"runner_id", runner.RunnerId,
							"error", err,
						)
						// If we can't find the agent, just remove from store
						if delErr := s.store.DeleteRunner(runner.RunnerId); delErr != nil {
							s.logger.Error("Failed to delete runner", "runner_id", runner.RunnerId, "error", delErr)
						}
						continue
					}

					// Send delete command to agent
					cmd := &agentv1.SyncResponse{
						Command: &agentv1.SyncResponse_DeleteRunner{
							DeleteRunner: &agentv1.DeleteRunnerCommand{
								RunnerId: runner.RunnerId,
							},
						},
					}

					if err := s.sendCommandToAgent(agentID, cmd); err != nil {
						s.logger.Error("Failed to send delete command for ERROR runner",
							"runner_id", runner.RunnerId,
							"error", err,
						)
					}
				}
			}
		}
	}
}
