package sync

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	agentv1 "github.com/whywaita/shoes-vz/gen/go/shoes/vz/agent/v1"
	"github.com/whywaita/shoes-vz/internal/agent/runner"
	"github.com/whywaita/shoes-vz/internal/agent/vm"
	"github.com/whywaita/shoes-vz/pkg/logging"
)

// Client manages bidirectional sync with the server
type Client struct {
	agentID       string
	serverAddr    string
	syncInterval  time.Duration
	runnerManager *runner.Manager
	vmManager     vm.Manager
	conn          *grpc.ClientConn
	client        agentv1.AgentServiceClient
	commandChan   chan *agentv1.SyncResponse
	logger        *slog.Logger
}

// NewClient creates a new sync client
func NewClient(
	serverAddr string,
	syncInterval time.Duration,
	runnerManager *runner.Manager,
	vmManager vm.Manager,
	logger *slog.Logger,
) *Client {
	return &Client{
		serverAddr:    serverAddr,
		syncInterval:  syncInterval,
		runnerManager: runnerManager,
		vmManager:     vmManager,
		commandChan:   make(chan *agentv1.SyncResponse, 10),
		logger:        logger,
	}
}

// Connect establishes connection to the server and registers the agent
func (c *Client) Connect(ctx context.Context, hostname string, capacity *agentv1.AgentCapacity) error {
	conn, err := grpc.NewClient(c.serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	c.conn = conn
	c.client = agentv1.NewAgentServiceClient(conn)

	// Register agent
	resp, err := c.client.RegisterAgent(ctx, &agentv1.RegisterAgentRequest{
		Hostname: hostname,
		Capacity: capacity,
	})
	if err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	c.agentID = resp.AgentId
	c.syncInterval = time.Duration(resp.SyncIntervalSeconds) * time.Second

	c.logger.Info("Agent registered",
		"agent_id", c.agentID,
		"sync_interval", c.syncInterval,
	)
	return nil
}

// Start starts the sync loop
func (c *Client) Start(ctx context.Context) error {
	// Create a cancellable context for the sync loop
	syncCtx, syncCancel := context.WithCancel(ctx)
	defer syncCancel()

	stream, err := c.client.Sync(syncCtx)
	if err != nil {
		return fmt.Errorf("failed to start sync stream: %w", err)
	}

	// Start receiving commands from server
	go c.receiveCommands(stream)

	// Start periodic sync
	go c.periodicSync(syncCtx, stream)

	// Process commands
	// When this returns, cancel the sync context to stop periodicSync
	err = c.processCommands(syncCtx)
	syncCancel() // Ensure periodicSync stops
	return err
}

// receiveCommands receives commands from the server
func (c *Client) receiveCommands(stream grpc.BidiStreamingClient[agentv1.SyncRequest, agentv1.SyncResponse]) {
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			c.logger.Info("Server closed the stream")
			close(c.commandChan)
			return
		}
		if err != nil {
			c.logger.Error("Error receiving from stream", "error", err)
			close(c.commandChan)
			return
		}

		c.commandChan <- resp
	}
}

// periodicSync sends periodic status updates to the server
func (c *Client) periodicSync(ctx context.Context, stream grpc.BidiStreamingClient[agentv1.SyncRequest, agentv1.SyncResponse]) {
	ticker := time.NewTicker(c.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.sendSync(stream); err != nil {
				c.logger.Error("Error sending sync", "error", err)
			}
		}
	}
}

// sendSync sends a sync request to the server
func (c *Client) sendSync(stream grpc.BidiStreamingClient[agentv1.SyncRequest, agentv1.SyncResponse]) error {
	runners := c.runnerManager.List()
	protoRunners := make([]*agentv1.Runner, len(runners))

	for i, r := range runners {
		protoRunners[i] = &agentv1.Runner{
			RunnerId:         r.ID,
			RunnerName:       r.Name,
			AgentId:          c.agentID,
			State:            r.State,
			IpAddress:        r.IPAddress,
			CreatedAt:        timestamppb.New(r.CreatedAt),
			ErrorMessage:     r.ErrorMessage,
			GuestRunnerState: r.GuestState,
		}
	}

	req := &agentv1.SyncRequest{
		AgentId:       c.agentID,
		ActiveRunners: uint32(c.runnerManager.Count()),
		Runners:       protoRunners,
	}

	return stream.Send(req)
}

// SendImmediateSync sends an immediate sync (for state changes)
func (c *Client) SendImmediateSync(ctx context.Context, stream grpc.BidiStreamingClient[agentv1.SyncRequest, agentv1.SyncResponse]) error {
	return c.sendSync(stream)
}

// processCommands processes commands received from the server
func (c *Client) processCommands(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case cmd, ok := <-c.commandChan:
			if !ok {
				return fmt.Errorf("command channel closed")
			}

			if err := c.handleCommand(ctx, cmd); err != nil {
				c.logger.Error("Error handling command", "error", err)
			}
		}
	}
}

// handleCommand handles a single command from the server
func (c *Client) handleCommand(ctx context.Context, cmd *agentv1.SyncResponse) error {
	switch cmdType := cmd.Command.(type) {
	case *agentv1.SyncResponse_CreateRunner:
		return c.handleCreateRunner(ctx, cmdType.CreateRunner)
	case *agentv1.SyncResponse_DeleteRunner:
		return c.handleDeleteRunner(ctx, cmdType.DeleteRunner)
	case *agentv1.SyncResponse_Noop:
		// No operation
		return nil
	default:
		return fmt.Errorf("unknown command type: %T", cmdType)
	}
}

// handleCreateRunner handles a create runner command
func (c *Client) handleCreateRunner(ctx context.Context, cmd *agentv1.CreateRunnerCommand) error {
	// Create context with request_id for logging
	ctx = logging.WithRequestID(ctx, cmd.RequestId)
	logger := logging.FromContext(ctx, c.logger)

	logger.Info("Creating runner",
		"runner_id", cmd.RunnerId,
		"runner_name", cmd.RunnerName,
	)

	// Create runner in manager
	if err := c.runnerManager.Create(ctx, cmd.RunnerId, cmd.RunnerName, cmd.SetupScript); err != nil {
		logger.Error("Failed to create runner", "error", err)
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Start runner creation in background
	go c.createRunnerAsync(ctx, cmd.RunnerId, cmd.RunnerName, cmd.SetupScript)

	return nil
}

// createRunnerAsync creates a runner asynchronously
func (c *Client) createRunnerAsync(ctx context.Context, runnerID, runnerName, setupScript string) {
	logger := logging.FromContext(ctx, c.logger)

	// Update state: CREATING
	c.runnerManager.UpdateState(runnerID, agentv1.RunnerState_RUNNER_STATE_CREATING)

	// Create VM
	vmInfo, err := c.vmManager.Create(ctx, runnerID)
	if err != nil {
		logger.Error("VM creation failed", "runner_id", runnerID, "error", err)
		c.runnerManager.SetError(runnerID, fmt.Sprintf("VM creation failed: %v", err))
		return
	}

	// Update state: BOOTING
	c.runnerManager.UpdateState(runnerID, agentv1.RunnerState_RUNNER_STATE_BOOTING)

	// Start VM
	if err := c.vmManager.Start(ctx, runnerID); err != nil {
		logger.Error("VM start failed", "runner_id", runnerID, "error", err)
		c.runnerManager.SetError(runnerID, fmt.Sprintf("VM start failed: %v", err))
		return
	}

	// Wait for SSH
	if err := c.vmManager.WaitForSSH(ctx, runnerID); err != nil {
		logger.Error("SSH wait failed", "runner_id", runnerID, "error", err)
		c.runnerManager.SetError(runnerID, fmt.Sprintf("SSH wait failed: %v", err))
		return
	}

	// Update state: SSH_READY
	c.runnerManager.UpdateState(runnerID, agentv1.RunnerState_RUNNER_STATE_SSH_READY)
	logger.Info("Runner SSH ready", "runner_id", runnerID)

	// Run setup script
	if err := c.vmManager.RunSetupScript(ctx, runnerID, setupScript); err != nil {
		logger.Error("Setup script failed", "runner_id", runnerID, "error", err)
		c.runnerManager.SetError(runnerID, fmt.Sprintf("Setup script failed: %v", err))
		return
	}

	// Update state: RUNNING
	c.runnerManager.UpdateState(runnerID, agentv1.RunnerState_RUNNER_STATE_RUNNING)

	// Update IP address
	if runner, err := c.runnerManager.Get(runnerID); err == nil {
		runner.IPAddress = vmInfo.IPAddress
	}

	logger.Info("Runner is now running",
		"runner_id", runnerID,
		"ip_address", vmInfo.IPAddress,
	)
}

// handleDeleteRunner handles a delete runner command
func (c *Client) handleDeleteRunner(ctx context.Context, cmd *agentv1.DeleteRunnerCommand) error {
	// Create context with request_id for logging
	ctx = logging.WithRequestID(ctx, cmd.RequestId)
	logger := logging.FromContext(ctx, c.logger)

	logger.Info("Deleting runner", "runner_id", cmd.RunnerId)

	// Update state: TEARING_DOWN
	c.runnerManager.UpdateState(cmd.RunnerId, agentv1.RunnerState_RUNNER_STATE_TEARING_DOWN)

	// Stop VM
	if err := c.vmManager.Stop(ctx, cmd.RunnerId); err != nil {
		logger.Warn("Failed to stop VM", "runner_id", cmd.RunnerId, "error", err)
		// Continue with deletion even if stop fails
	}

	// Delete VM
	if err := c.vmManager.Delete(ctx, cmd.RunnerId); err != nil {
		logger.Error("Failed to delete VM", "runner_id", cmd.RunnerId, "error", err)
		// Don't return error if bundle directory was already deleted
		// This can happen if the VM was cleaned up externally
		errMsg := err.Error()
		if !strings.Contains(errMsg, "no such file") && !strings.Contains(errMsg, "not exist") {
			return fmt.Errorf("failed to delete VM: %w", err)
		}
		logger.Warn("VM bundle already deleted, continuing", "runner_id", cmd.RunnerId)
	}

	// Remove from manager
	// If runner is not found, it means it was already deleted, which is fine
	if err := c.runnerManager.Delete(cmd.RunnerId); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "not found") {
			logger.Info("Runner already removed from manager", "runner_id", cmd.RunnerId)
		} else {
			logger.Error("Failed to remove runner", "runner_id", cmd.RunnerId, "error", err)
			return fmt.Errorf("failed to remove runner: %w", err)
		}
	}

	logger.Info("Runner deleted successfully", "runner_id", cmd.RunnerId)
	return nil
}

// Close closes the connection to the server
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
