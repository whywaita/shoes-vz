package ipnotify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/whywaita/shoes-vz/pkg/logging"
)

// IPNotification represents the JSON payload sent from runner-agent to shoes-vz-agent
type IPNotification struct {
	RunnerID  string `json:"runner_id"`
	IPAddress string `json:"ip_address"`
}

// PendingRequest represents a pending IP notification request
type PendingRequest struct {
	RunnerID string
	Ch       chan IPInfo
}

// IPInfo contains IP address and the UUID from the guest
type IPInfo struct {
	IPAddress string
	UUID      string
}

// Server is an HTTP server that receives IP notifications from runner-agents
type Server struct {
	listenAddr     string
	server         *http.Server
	pendingQueue   []PendingRequest  // Queue of pending requests (FIFO)
	uuidToRunnerID map[string]string // Maps guest UUID to runner ID
	mu             sync.RWMutex
}

// NewServer creates a new IP notification server
func NewServer(port int) *Server {
	return &Server{
		listenAddr:     fmt.Sprintf(":%d", port),
		pendingQueue:   make([]PendingRequest, 0),
		uuidToRunnerID: make(map[string]string),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	logger := logging.WithComponent("ipnotify")

	mux := http.NewServeMux()
	mux.HandleFunc("/notify-ip", s.handleIPNotification)

	s.server = &http.Server{
		Addr:    s.listenAddr,
		Handler: mux,
	}

	go func() {
		logger.Info("Starting IP notification server", "listen_addr", s.listenAddr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("IP notification server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// WaitForIP waits for an IP notification from any runner and associates it with the given runner ID
// This handles the case where the guest UUID is different from the runner ID
func (s *Server) WaitForIP(ctx context.Context, runnerID string, timeout time.Duration) (string, error) {
	ch := make(chan IPInfo, 1)

	s.mu.Lock()
	// Add to pending queue (FIFO)
	s.pendingQueue = append(s.pendingQueue, PendingRequest{
		RunnerID: runnerID,
		Ch:       ch,
	})
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		// Remove from queue if still pending
		for i, req := range s.pendingQueue {
			if req.RunnerID == runnerID {
				s.pendingQueue = append(s.pendingQueue[:i], s.pendingQueue[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case info := <-ch:
		// Store UUID to runner ID mapping
		s.mu.Lock()
		s.uuidToRunnerID[info.UUID] = runnerID
		s.mu.Unlock()

		logger := logging.WithComponent("ipnotify")
		logger.Info("Mapped UUID to runner", "uuid", info.UUID, "runner_id", runnerID, "ip_address", info.IPAddress)
		return info.IPAddress, nil
	case <-timeoutCtx.Done():
		return "", fmt.Errorf("timeout waiting for IP notification for runner %s", runnerID)
	}
}

// handleIPNotification handles POST /notify-ip requests
func (s *Server) handleIPNotification(w http.ResponseWriter, r *http.Request) {
	logger := logging.WithComponent("ipnotify")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var notification IPNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		logger.Error("Failed to decode IP notification", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if notification.RunnerID == "" || notification.IPAddress == "" {
		logger.Warn("Missing runner_id or ip_address in notification")
		http.Error(w, "Missing runner_id or ip_address", http.StatusBadRequest)
		return
	}

	logger.Info("Received IP notification", "uuid", notification.RunnerID, "ip_address", notification.IPAddress)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if UUID is already mapped to a runner ID
	if runnerID, exists := s.uuidToRunnerID[notification.RunnerID]; exists {
		logger.Info("UUID already mapped", "uuid", notification.RunnerID, "runner_id", runnerID)
		// Find the pending request for this runner
		for _, req := range s.pendingQueue {
			if req.RunnerID == runnerID {
				select {
				case req.Ch <- IPInfo{IPAddress: notification.IPAddress, UUID: notification.RunnerID}:
					w.WriteHeader(http.StatusOK)
					if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
						logger.Error("Failed to encode response", "error", err)
					}
					return
				default:
					logger.Warn("Channel full or closed", "runner_id", runnerID)
				}
			}
		}
	}

	// If no mapping exists, assign to the first pending request (FIFO)
	if len(s.pendingQueue) == 0 {
		logger.Warn("No pending requests", "uuid", notification.RunnerID)
		http.Error(w, "No pending requests", http.StatusNotFound)
		return
	}

	// Get first pending request
	req := s.pendingQueue[0]
	logger.Info("Assigning UUID to first pending runner", "uuid", notification.RunnerID, "runner_id", req.RunnerID)

	select {
	case req.Ch <- IPInfo{IPAddress: notification.IPAddress, UUID: notification.RunnerID}:
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			logger.Error("Failed to encode response", "error", err)
		}
	default:
		logger.Error("Channel full or closed", "runner_id", req.RunnerID)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}
