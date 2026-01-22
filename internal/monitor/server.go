package monitor

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"

	"github.com/whywaita/shoes-vz/pkg/model"
)

// Server is the HTTP server for the runner monitor
type Server struct {
	monitor    *Monitor
	listenAddr string
}

// NewServer creates a new Server instance
func NewServer(config *model.MonitorConfig) *Server {
	return &Server{
		monitor:    NewMonitor(config.RunnerPath),
		listenAddr: config.ListenAddr,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	http.HandleFunc("/status", s.handleStatus)
	http.HandleFunc("/health", s.handleHealth)
	http.HandleFunc("/exec", s.handleExec)

	log.Printf("Starting HTTP server on %s", s.listenAddr)
	return http.ListenAndServe(s.listenAddr, nil)
}

// handleStatus handles GET /status requests
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status, err := s.monitor.GetStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleHealth handles GET /health requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		log.Printf("Failed to encode health response: %v", err)
	}
}

// ExecRequest represents a command execution request
type ExecRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
}

// ExecResponse represents a command execution response
type ExecResponse struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Error    string `json:"error,omitempty"`
}

// handleExec handles POST /exec requests to execute commands
func (s *Server) handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, "Command is required", http.StatusBadRequest)
		return
	}

	// Execute command
	cmd := exec.Command(req.Command, req.Args...)
	output, err := cmd.CombinedOutput()

	resp := ExecResponse{
		Output: string(output),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			resp.ExitCode = exitErr.ExitCode()
		} else {
			resp.ExitCode = -1
			resp.Error = err.Error()
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
