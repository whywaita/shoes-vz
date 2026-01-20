package client

import (
	"fmt"
	"os"
)

const (
	// EnvShoesVzServerAddr is the environment variable name for shoes-vz-server address
	EnvShoesVzServerAddr = "SHOESVZ_SERVER_ADDR"
)

// Config holds the configuration for the shoesvz-client
type Config struct {
	ServerAddr string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	serverAddr := os.Getenv(EnvShoesVzServerAddr)
	if serverAddr == "" {
		return nil, fmt.Errorf("%s environment variable is required", EnvShoesVzServerAddr)
	}

	return &Config{
		ServerAddr: serverAddr,
	}, nil
}
