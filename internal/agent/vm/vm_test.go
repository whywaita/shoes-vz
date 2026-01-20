package vm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/whywaita/shoes-vz/pkg/model"
)

func TestCloneFile(t *testing.T) {
	// Skip if not running on macOS
	if os.Getenv("GOOS") == "linux" || os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping macOS-specific test")
	}

	// Create a temporary test file
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	dstFile := filepath.Join(tmpDir, "dest.txt")

	testContent := []byte("test content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test APFS clone
	if err := cloneFile(srcFile, dstFile); err != nil {
		t.Fatalf("cloneFile() error = %v", err)
	}

	// Verify the cloned file exists and has the same content
	content, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read cloned file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("Cloned file content = %s, want %s", string(content), string(testContent))
	}
}

func TestBundleConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config, err := LoadBundleConfig(tmpDir)
	if err != nil {
		t.Fatalf("LoadBundleConfig() error = %v", err)
	}

	expectedDiskPath := filepath.Join(tmpDir, "Disk.img")
	if config.DiskPath != expectedDiskPath {
		t.Errorf("DiskPath = %s, want %s", config.DiskPath, expectedDiskPath)
	}
}

func TestRuntimeMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	metadataPath := filepath.Join(tmpDir, "metadata.json")

	// Test save
	metadata := &RuntimeMetadata{
		RunnerID:  "test-runner",
		IPAddress: "192.168.64.2",
		CreatedAt: "2024-01-01T00:00:00Z",
		State:     "running",
		UpdatedAt: "2024-01-01T00:00:00Z",
	}

	if err := SaveRuntimeMetadata(metadataPath, metadata); err != nil {
		t.Fatalf("SaveRuntimeMetadata() error = %v", err)
	}

	// Test load
	loaded, err := LoadRuntimeMetadata(metadataPath)
	if err != nil {
		t.Fatalf("LoadRuntimeMetadata() error = %v", err)
	}

	if loaded.RunnerID != metadata.RunnerID {
		t.Errorf("RunnerID = %s, want %s", loaded.RunnerID, metadata.RunnerID)
	}

	if loaded.IPAddress != metadata.IPAddress {
		t.Errorf("IPAddress = %s, want %s", loaded.IPAddress, metadata.IPAddress)
	}

	if loaded.State != metadata.State {
		t.Errorf("State = %s, want %s", loaded.State, metadata.State)
	}
}

// TestVMManager_Create tests VM creation (requires template)
func TestVMManager_Create(t *testing.T) {
	// Skip if template doesn't exist
	templatePath := os.Getenv("TEST_VM_TEMPLATE")
	if templatePath == "" {
		t.Skip("Skipping VM creation test: TEST_VM_TEMPLATE not set")
	}

	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		t.Skipf("Skipping VM creation test: template not found at %s", templatePath)
	}

	// Create temporary runners directory
	tmpDir := t.TempDir()

	config := &model.AgentConfig{
		TemplatePath: templatePath,
		RunnersPath:  tmpDir,
		SSHKeyPath:   "",
	}

	manager := NewManager(config, nil)

	// Test VM creation
	ctx := context.Background()
	vmInfo, err := manager.Create(ctx, "test-runner-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify bundle was created
	if _, err := os.Stat(vmInfo.BundlePath); os.IsNotExist(err) {
		t.Errorf("Bundle directory was not created: %s", vmInfo.BundlePath)
	}

	// Verify disk was cloned
	diskPath := filepath.Join(vmInfo.BundlePath, "Disk.img")
	if _, err := os.Stat(diskPath); os.IsNotExist(err) {
		t.Errorf("Disk.img was not cloned: %s", diskPath)
	}

	// Verify metadata was saved
	metadataPath := filepath.Join(vmInfo.BundlePath, "RuntimeMetadata.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		t.Errorf("RuntimeMetadata.json was not created: %s", metadataPath)
	}

	// Clean up
	if err := manager.Delete(ctx, "test-runner-1"); err != nil {
		t.Errorf("Delete() error = %v", err)
	}
}
