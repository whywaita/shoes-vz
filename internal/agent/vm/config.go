package vm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// MonitorTCPPort is the TCP port used by runner-agent for HTTP communication
const MonitorTCPPort = 8080

// BundleConfig represents the VM bundle configuration
type BundleConfig struct {
	DiskPath            string `json:"disk_path"`
	AuxiliaryPath       string `json:"auxiliary_path"`
	HardwareModelPath   string `json:"hardware_model_path"`
	MachineIdentifier   string `json:"machine_identifier"`
	RuntimeMetadataPath string `json:"runtime_metadata_path"`
}

// RuntimeMetadata contains runtime information about the VM
type RuntimeMetadata struct {
	RunnerID  string `json:"runner_id"`
	IPAddress string `json:"ip_address"` // Guest IP address (set after VM starts)
	CreatedAt string `json:"created_at"`
	State     string `json:"state"`      // Current state: creating, running, stopped, error, etc.
	UpdatedAt string `json:"updated_at"` // Last update timestamp
}

// LoadBundleConfig loads the bundle configuration from a directory
func LoadBundleConfig(bundlePath string) (*BundleConfig, error) {
	return &BundleConfig{
		DiskPath:            filepath.Join(bundlePath, "Disk.img"),
		AuxiliaryPath:       filepath.Join(bundlePath, "AuxiliaryStorage"),
		HardwareModelPath:   filepath.Join(bundlePath, "HardwareModel.json"),
		MachineIdentifier:   filepath.Join(bundlePath, "MachineIdentifier"),
		RuntimeMetadataPath: filepath.Join(bundlePath, "RuntimeMetadata.json"),
	}, nil
}

// SaveRuntimeMetadata saves runtime metadata to a file
func SaveRuntimeMetadata(path string, metadata *RuntimeMetadata) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// LoadRuntimeMetadata loads runtime metadata from a file
func LoadRuntimeMetadata(path string) (*RuntimeMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata RuntimeMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}
