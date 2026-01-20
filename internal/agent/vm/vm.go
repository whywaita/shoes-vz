package vm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Code-Hex/vz/v3"

	"github.com/whywaita/shoes-vz/internal/agent/ipnotify"
	"github.com/whywaita/shoes-vz/pkg/logging"
	"github.com/whywaita/shoes-vz/pkg/model"
)

// Manager manages VM lifecycle using Apple Virtualization Framework
type Manager interface {
	// Create creates a new VM by cloning the template
	Create(ctx context.Context, runnerID string) (*VMInfo, error)

	// Start starts the VM
	Start(ctx context.Context, runnerID string) error

	// Stop stops the VM
	Stop(ctx context.Context, runnerID string) error

	// Delete deletes the VM and its bundle
	Delete(ctx context.Context, runnerID string) error

	// WaitForSSH waits until SSH is ready on the VM
	WaitForSSH(ctx context.Context, runnerID string) error

	// RunSetupScript runs the setup script via SSH
	RunSetupScript(ctx context.Context, runnerID, script string) error

	// Exec executes a command on the VM via HTTP (using runner-agent)
	Exec(ctx context.Context, runnerID, command string, args []string) ([]byte, int, error)
}

// VMInfo contains information about a VM
type VMInfo struct {
	RunnerID   string
	BundlePath string
	IPAddress  string // Guest IP address for SSH connection
}

// vzManager implements Manager using Code-Hex/vz
type vzManager struct {
	templatePath   string
	runnersPath    string
	sshKeyPath     string
	ipNotifyServer *ipnotify.Server
	enableGraphics bool

	mu  sync.RWMutex
	vms map[string]*vz.VirtualMachine
}

// NewManager creates a new VM Manager
func NewManager(config *model.AgentConfig, ipNotifyServer *ipnotify.Server) Manager {
	return &vzManager{
		templatePath:   config.TemplatePath,
		runnersPath:    config.RunnersPath,
		sshKeyPath:     config.SSHKeyPath,
		ipNotifyServer: ipNotifyServer,
		enableGraphics: config.EnableGraphics,
		vms:            make(map[string]*vz.VirtualMachine),
	}
}

// Create creates a new VM by cloning the template
func (m *vzManager) Create(ctx context.Context, runnerID string) (*VMInfo, error) {
	// Create runner bundle directory
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	if err := os.MkdirAll(bundlePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create bundle directory: %w", err)
	}

	// Clone Disk.img
	diskSrc := filepath.Join(m.templatePath, "Disk.img")
	diskDst := filepath.Join(bundlePath, "Disk.img")
	if err := cloneFile(diskSrc, diskDst); err != nil {
		return nil, fmt.Errorf("failed to clone disk: %w", err)
	}

	// Clone AuxiliaryStorage
	auxSrc := filepath.Join(m.templatePath, "AuxiliaryStorage")
	auxDst := filepath.Join(bundlePath, "AuxiliaryStorage")
	if err := cloneFile(auxSrc, auxDst); err != nil {
		return nil, fmt.Errorf("failed to clone auxiliary storage: %w", err)
	}

	// Copy HardwareModel.json (required for macOS VMs)
	hwModelSrc := filepath.Join(m.templatePath, "HardwareModel.json")
	hwModelDst := filepath.Join(bundlePath, "HardwareModel.json")
	if _, err := os.Stat(hwModelSrc); os.IsNotExist(err) {
		return nil, fmt.Errorf("hardware model not found in template: %s", hwModelSrc)
	}
	if err := copyFile(hwModelSrc, hwModelDst); err != nil {
		return nil, fmt.Errorf("failed to copy hardware model: %w", err)
	}

	// Create a new Mac machine identifier
	machineIDPath := filepath.Join(bundlePath, "MachineIdentifier")
	machineIdentifier, err := vz.NewMacMachineIdentifier()
	if err != nil {
		return nil, fmt.Errorf("failed to create machine identifier: %w", err)
	}

	// Save the binary data representation
	dataRep := machineIdentifier.DataRepresentation()
	if err := os.WriteFile(machineIDPath, dataRep, 0644); err != nil {
		return nil, fmt.Errorf("failed to write machine identifier: %w", err)
	}

	// Save runtime metadata
	now := time.Now().Format(time.RFC3339)
	metadata := &RuntimeMetadata{
		RunnerID:  runnerID,
		IPAddress: "", // Will be set after VM starts and we get the IP
		CreatedAt: now,
		State:     "creating",
		UpdatedAt: now,
	}
	metadataPath := filepath.Join(bundlePath, "RuntimeMetadata.json")
	if err := SaveRuntimeMetadata(metadataPath, metadata); err != nil {
		return nil, fmt.Errorf("failed to save runtime metadata: %w", err)
	}

	return &VMInfo{
		RunnerID:   runnerID,
		BundlePath: bundlePath,
		IPAddress:  "", // Will be discovered after VM starts
	}, nil
}

// Start starts the VM
func (m *vzManager) Start(ctx context.Context, runnerID string) error {
	logger := logging.WithComponent("vm")

	// Update state to booting
	if err := m.UpdateState(runnerID, "booting"); err != nil {
		// Log but don't fail
		logger.Warn("Failed to update state to booting", "runner_id", runnerID, "error", err)
	}

	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))

	// Load bundle config
	bundleConfig, err := LoadBundleConfig(bundlePath)
	if err != nil {
		m.UpdateState(runnerID, "error")
		return fmt.Errorf("failed to load bundle config: %w", err)
	}

	// Create VM configuration
	vmConfig, err := m.createVMConfig(bundleConfig)
	if err != nil {
		return fmt.Errorf("failed to create VM config: %w", err)
	}

	// Validate configuration
	validated, err := vmConfig.Validate()
	if err != nil {
		return fmt.Errorf("VM config validation failed: %w", err)
	}
	if !validated {
		return fmt.Errorf("VM config validation returned false")
	}

	// Create and start VM
	vm, err := vz.NewVirtualMachine(vmConfig)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	// Store VM instance
	m.mu.Lock()
	m.vms[runnerID] = vm
	m.mu.Unlock()

	// Start VM
	logger.Info("Starting VM", "runner_id", runnerID, "graphics_enabled", m.enableGraphics)
	if err := vm.Start(); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Wait for VM to reach running state
	logger.Info("Waiting for VM to reach running state", "runner_id", runnerID)
	for i := 0; i < 60; i++ {
		state := vm.State()
		logger.Debug("VM state check", "runner_id", runnerID, "attempt", i+1, "state", state)
		if state == vz.VirtualMachineStateRunning {
			break
		}
		if state == vz.VirtualMachineStateError || state == vz.VirtualMachineStateStopped {
			logger.Error("VM failed to start", "runner_id", runnerID, "state", state)
			return fmt.Errorf("VM failed to start, state: %v", state)
		}
		time.Sleep(1 * time.Second)
	}

	if vm.State() != vz.VirtualMachineStateRunning {
		logger.Error("VM did not reach running state", "runner_id", runnerID, "state", vm.State())
		return fmt.Errorf("VM did not reach running state, current state: %v", vm.State())
	}

	logger.Info("VM is now running, waiting for IP notification", "runner_id", runnerID)

	// Wait for IP notification from runner-agent (2 minutes timeout)
	// The runner-agent will send its IOPlatformUUID (obtained from ioreg) as runner_id
	// The IP notify server will match it to this runner ID using FIFO queue
	ipAddress, err := m.ipNotifyServer.WaitForIP(ctx, runnerID, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("failed to receive IP notification: %w", err)
	}

	logger.Info("Guest IP received via notification", "runner_id", runnerID, "ip_address", ipAddress)

	// Update metadata with the discovered IP
	if err := m.UpdateIPAddress(runnerID, ipAddress); err != nil {
		return fmt.Errorf("failed to update IP address: %w", err)
	}

	// Update state to running
	if err := m.UpdateState(runnerID, "running"); err != nil {
		logger.Warn("Failed to update state to running", "runner_id", runnerID, "error", err)
	}

	return nil
}

// createVMConfig creates a VM configuration
func (m *vzManager) createVMConfig(bundleConfig *BundleConfig) (*vz.VirtualMachineConfiguration, error) {
	// Create boot loader
	bootLoader, err := vz.NewMacOSBootLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to create boot loader: %w", err)
	}

	// Create basic configuration
	// Parameters: bootLoader, CPUCount, MemorySize
	config, err := vz.NewVirtualMachineConfiguration(
		bootLoader,
		2,                // CPU count
		4*1024*1024*1024, // 4GB memory
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM config: %w", err)
	}

	// Create platform configuration
	// Load hardware model (required for macOS VMs)
	// Note: HardwareModel.json is a JSON file with a base64-encoded hardware model
	hardwareModel, err := LoadHardwareModel(bundleConfig.HardwareModelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load hardware model from %s: %w", bundleConfig.HardwareModelPath, err)
	}

	// Load machine identifier (required for macOS VMs)
	machineIDData, err := os.ReadFile(bundleConfig.MachineIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to read machine identifier from %s: %w", bundleConfig.MachineIdentifier, err)
	}

	machineIdentifier, err := vz.NewMacMachineIdentifierWithData(machineIDData)
	if err != nil {
		return nil, fmt.Errorf("failed to load machine identifier: %w", err)
	}

	// Load auxiliary storage
	auxiliaryStorage, err := vz.NewMacAuxiliaryStorage(bundleConfig.AuxiliaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create auxiliary storage: %w", err)
	}

	// Create Mac platform configuration with hardware model, machine identifier and auxiliary storage
	platform, err := vz.NewMacPlatformConfiguration(
		vz.WithMacHardwareModel(hardwareModel),
		vz.WithMacMachineIdentifier(machineIdentifier),
		vz.WithMacAuxiliaryStorage(auxiliaryStorage),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create platform: %w", err)
	}
	config.SetPlatformVirtualMachineConfiguration(platform)

	// Create storage device
	diskAttachment, err := vz.NewDiskImageStorageDeviceAttachment(
		bundleConfig.DiskPath,
		false, // read-only
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk attachment: %w", err)
	}

	storageConfig, err := vz.NewVirtioBlockDeviceConfiguration(diskAttachment)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage config: %w", err)
	}

	config.SetStorageDevicesVirtualMachineConfiguration([]vz.StorageDeviceConfiguration{storageConfig})

	// Create network device with NAT
	natAttachment, err := vz.NewNATNetworkDeviceAttachment()
	if err != nil {
		return nil, fmt.Errorf("failed to create NAT attachment: %w", err)
	}

	networkConfig, err := vz.NewVirtioNetworkDeviceConfiguration(natAttachment)
	if err != nil {
		return nil, fmt.Errorf("failed to create network config: %w", err)
	}

	config.SetNetworkDevicesVirtualMachineConfiguration([]*vz.VirtioNetworkDeviceConfiguration{networkConfig})

	// Add graphics device if graphics is enabled
	if m.enableGraphics {
		// Create Mac graphics device
		graphicsDevice, err := vz.NewMacGraphicsDeviceConfiguration()
		if err != nil {
			return nil, fmt.Errorf("failed to create graphics device: %w", err)
		}

		// Create display configuration (1024x768 at 72 PPI)
		display, err := vz.NewMacGraphicsDisplayConfiguration(1024, 768, 72)
		if err != nil {
			return nil, fmt.Errorf("failed to create display configuration: %w", err)
		}

		graphicsDevice.SetDisplays(display)
		config.SetGraphicsDevicesVirtualMachineConfiguration([]vz.GraphicsDeviceConfiguration{graphicsDevice})

		// Add keyboard for macOS VMs with graphics
		keyboardConfig, err := vz.NewMacKeyboardConfiguration()
		if err != nil {
			return nil, fmt.Errorf("failed to create keyboard configuration: %w", err)
		}
		config.SetKeyboardsVirtualMachineConfiguration([]vz.KeyboardConfiguration{keyboardConfig})

		// Add pointing device (trackpad)
		pointingDevice, err := vz.NewMacTrackpadConfiguration()
		if err != nil {
			return nil, fmt.Errorf("failed to create trackpad configuration: %w", err)
		}
		config.SetPointingDevicesVirtualMachineConfiguration([]vz.PointingDeviceConfiguration{pointingDevice})
	}

	return config, nil
}

// Stop stops the VM
func (m *vzManager) Stop(ctx context.Context, runnerID string) error {
	m.mu.RLock()
	vm, exists := m.vms[runnerID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("VM not found: %s", runnerID)
	}

	// Try graceful shutdown first
	if vm.CanRequestStop() {
		result, err := vm.RequestStop()
		if err == nil && result {
			// Wait for VM to stop
			for i := 0; i < 30; i++ {
				if vm.State() == vz.VirtualMachineStateStopped {
					m.mu.Lock()
					delete(m.vms, runnerID)
					m.mu.Unlock()
					m.UpdateState(runnerID, "stopped")
					return nil
				}
				time.Sleep(1 * time.Second)
			}
		}
	}

	// Force stop if graceful shutdown failed
	if err := vm.Stop(); err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	m.mu.Lock()
	delete(m.vms, runnerID)
	m.mu.Unlock()

	// Update state to stopped
	m.UpdateState(runnerID, "stopped")

	return nil
}

// Delete deletes the VM and its bundle
func (m *vzManager) Delete(ctx context.Context, runnerID string) error {
	logger := logging.WithComponent("vm")

	// Ensure VM is stopped
	if err := m.Stop(ctx, runnerID); err != nil {
		// Log error but continue with deletion
		logger.Warn("Failed to stop VM before deletion", "runner_id", runnerID, "error", err)
	}

	// Delete bundle directory
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	if err := os.RemoveAll(bundlePath); err != nil {
		return fmt.Errorf("failed to delete bundle: %w", err)
	}

	return nil
}

// WaitForSSH waits until SSH is ready on the VM
func (m *vzManager) WaitForSSH(ctx context.Context, runnerID string) error {
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	bundleConfig, err := LoadBundleConfig(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to load bundle config: %w", err)
	}

	metadata, err := LoadRuntimeMetadata(bundleConfig.RuntimeMetadataPath)
	if err != nil {
		return fmt.Errorf("failed to load runtime metadata: %w", err)
	}

	return waitForSSH(ctx, runnerID, metadata.IPAddress, m.sshKeyPath, 5*time.Minute)
}

// RunSetupScript runs the setup script via SSH
func (m *vzManager) RunSetupScript(ctx context.Context, runnerID, script string) error {
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	bundleConfig, err := LoadBundleConfig(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to load bundle config: %w", err)
	}

	metadata, err := LoadRuntimeMetadata(bundleConfig.RuntimeMetadataPath)
	if err != nil {
		return fmt.Errorf("failed to load runtime metadata: %w", err)
	}

	return runSSHScript(ctx, runnerID, metadata.IPAddress, m.sshKeyPath, script)
}

// Exec executes a command on the VM via HTTP using runner-agent
func (m *vzManager) Exec(ctx context.Context, runnerID, command string, args []string) ([]byte, int, error) {
	bundlePath := filepath.Join(m.runnersPath, fmt.Sprintf("%s.bundle", runnerID))
	bundleConfig, err := LoadBundleConfig(bundlePath)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to load bundle config: %w", err)
	}

	metadata, err := LoadRuntimeMetadata(bundleConfig.RuntimeMetadataPath)
	if err != nil {
		return nil, -1, fmt.Errorf("failed to load runtime metadata: %w", err)
	}

	if metadata.IPAddress == "" {
		return nil, -1, fmt.Errorf("VM IP address not yet discovered")
	}

	return execViaHTTP(ctx, metadata.IPAddress, MonitorTCPPort, command, args)
}
