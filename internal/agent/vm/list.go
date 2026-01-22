package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VMListItem represents a VM in the list
type VMListItem struct {
	RunnerID   string
	BundlePath string
	IPAddress  string
	CreatedAt  string
	State      string
	UpdatedAt  string
}

// ListVMs lists all VM bundles in the runners directory
func ListVMs(runnersPath string) ([]VMListItem, error) {
	// Check if runners directory exists
	if _, err := os.Stat(runnersPath); os.IsNotExist(err) {
		return []VMListItem{}, nil
	}

	// Read directory entries
	entries, err := os.ReadDir(runnersPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read runners directory: %w", err)
	}

	var vms []VMListItem

	// Iterate through directories
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if it's a bundle directory (ends with .bundle)
		name := entry.Name()
		if !strings.HasSuffix(name, ".bundle") {
			continue
		}

		bundlePath := filepath.Join(runnersPath, name)

		// Try to load runtime metadata
		bundleConfig, err := LoadBundleConfig(bundlePath)
		if err != nil {
			// Skip bundles that can't be loaded
			continue
		}

		metadata, err := LoadRuntimeMetadata(bundleConfig.RuntimeMetadataPath)
		if err != nil {
			// Skip bundles without valid metadata
			continue
		}

		vms = append(vms, VMListItem{
			RunnerID:   metadata.RunnerID,
			BundlePath: bundlePath,
			IPAddress:  metadata.IPAddress,
			CreatedAt:  metadata.CreatedAt,
			State:      metadata.State,
			UpdatedAt:  metadata.UpdatedAt,
		})
	}

	return vms, nil
}
