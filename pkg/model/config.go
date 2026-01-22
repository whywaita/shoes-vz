package model

import "time"

// ServerConfig contains configuration for shoes-vz-server
type ServerConfig struct {
	GRPCAddr     string
	MetricsAddr  string
	SyncInterval time.Duration
	AgentTimeout time.Duration
}

// AgentConfig contains configuration for shoes-vz-agent
type AgentConfig struct {
	ServerAddr     string
	Hostname       string
	MaxRunners     uint32
	TemplatePath   string
	RunnersPath    string
	SSHKeyPath     string
	SyncInterval   time.Duration
	EnableGraphics bool
}

// MonitorConfig contains configuration for shoes-vz-runner-agent
type MonitorConfig struct {
	ListenAddr   string // TCP listen address
	RunnerPath   string
	PollInterval time.Duration
}
