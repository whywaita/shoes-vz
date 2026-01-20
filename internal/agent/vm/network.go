package vm

import (
	"fmt"
	"path/filepath"
	"time"
)

// UpdateIPAddress updates the IP address in the runtime metadata
func (m *vzManager) UpdateIPAddress(runnerID, ipAddress string) error {
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	bundleConfig, err := LoadBundleConfig(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to load bundle config: %w", err)
	}

	metadata, err := LoadRuntimeMetadata(bundleConfig.RuntimeMetadataPath)
	if err != nil {
		return fmt.Errorf("failed to load runtime metadata: %w", err)
	}

	metadata.IPAddress = ipAddress
	metadata.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := SaveRuntimeMetadata(bundleConfig.RuntimeMetadataPath, metadata); err != nil {
		return fmt.Errorf("failed to save runtime metadata: %w", err)
	}

	return nil
}

// UpdateState updates the state in the runtime metadata
func (m *vzManager) UpdateState(runnerID, state string) error {
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	bundleConfig, err := LoadBundleConfig(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to load bundle config: %w", err)
	}

	metadata, err := LoadRuntimeMetadata(bundleConfig.RuntimeMetadataPath)
	if err != nil {
		return fmt.Errorf("failed to load runtime metadata: %w", err)
	}

	metadata.State = state
	metadata.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := SaveRuntimeMetadata(bundleConfig.RuntimeMetadataPath, metadata); err != nil {
		return fmt.Errorf("failed to save runtime metadata: %w", err)
	}

	return nil
}
