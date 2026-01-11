package vm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nikiskaarup/sear/internal/config"
	"github.com/nikiskaarup/sear/internal/firecracker"
	"github.com/nikiskaarup/sear/internal/network"
	"github.com/nikiskaarup/sear/internal/ssh"
	"github.com/sirupsen/logrus"
)

// VM represents a Firecracker microVM
type VM struct {
	profile    config.Profile
	fcClient   *firecracker.Client
	netManager *network.Manager
	sshClient  *SSHClient
}

// SSHClient wraps the SSH client for VM interaction
type SSHClient struct {
	client *ssh.Client
}

// ExecuteCommand executes a command in the VM
func (c *SSHClient) ExecuteCommand(cmd string) error {
	return c.client.ExecuteCommand(cmd)
}

// Shell starts an interactive shell in the VM
func (c *SSHClient) Shell() error {
	return c.client.Shell()
}

// NewVM creates a new VM instance
func NewVM(profile config.Profile) (*VM, error) {
	return &VM{
		profile: profile,
	}, nil
}

// Start starts the VM
func (v *VM) Start() error {
	logrus.Info("Starting VM...")

	// Get network configuration
	networkConfig := v.getEffectiveNetworkConfig()

	// Setup network
	v.netManager = network.NewManager(
		networkConfig.TAPDevice,
		networkConfig.TAPIP,
		networkConfig.GatewayIP,
	)
	if err := v.netManager.Setup(); err != nil {
		return fmt.Errorf("failed to setup network: %w", err)
	}

	// Create Firecracker client
	socketPath := os.Getenv("FIRECRACKER_API_SOCKET")
	if socketPath == "" {
		socketPath = "/tmp/firecracker.socket"
	}

	fcClient, err := firecracker.NewClient(socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to Firecracker: %w", err)
	}
	v.fcClient = fcClient

	// Configure logger
	if err := fcClient.ConfigureLogger("/tmp/sear-firecracker.log", "Debug"); err != nil {
		logrus.Warnf("Failed to configure logger: %v", err)
	}

	// Set boot source
	kernelArgs := "console=ttyS0 reboot=k panic=1"
	if v.profile.VM.KernelArgs != "" {
		kernelArgs = v.profile.VM.KernelArgs
	}

	if err := fcClient.SetBootSource(v.profile.VM.Kernel, kernelArgs); err != nil {
		return fmt.Errorf("failed to set boot source: %w", err)
	}

	// Attach rootfs
	if err := fcClient.AttachRootfs("rootfs", v.profile.VM.RootFS, true, false); err != nil {
		return fmt.Errorf("failed to attach rootfs: %w", err)
	}

	// Attach network
	mac := v.netManager.GetMACAddress()
	if err := fcClient.AttachNetwork("net1", mac, v.netManager.TAPDevice); err != nil {
		return fmt.Errorf("failed to attach network: %w", err)
	}

	// Start instance
	if err := fcClient.StartInstance(); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	logrus.Info("VM started successfully")
	return nil
}

// Stop stops the VM
func (v *VM) Stop() error {
	logrus.Info("Stopping VM...")

	// Cleanup network
	if v.netManager != nil {
		if err := v.netManager.Teardown(); err != nil {
			logrus.Warnf("Failed to teardown network: %v", err)
		}
	}

	return nil
}

// GetSSHClient returns an SSH client for the VM
func (v *VM) GetSSHClient() (*SSHClient, error) {
	networkConfig := v.getEffectiveNetworkConfig()

	// Get SSH key path from config
	sshKeyPath := "sear_key"
	configHome, err := os.UserHomeDir()
	if err == nil {
		sshKeyPath = filepath.Join(configHome, ".config", "sear", "sear_key")
	}

	sshClient := ssh.NewClient(
		networkConfig.GuestIP,
		22,
		"root",
		sshKeyPath,
	)

	return &SSHClient{client: sshClient}, nil
}

// MountDirectory mounts a directory into the VM using virtiofs
func (v *VM) MountDirectory(sshClient *SSHClient, hostPath string) error {
	logrus.Infof("Mounting directory: %s", hostPath)

	// Get absolute path
	absPath, err := filepath.Abs(hostPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Create mount point in VM
	mountPoint := "/host"
	if err := sshClient.ExecuteCommand(fmt.Sprintf("mkdir -p %s", mountPoint)); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Mount using virtiofs
	// Note: This requires virtiofsd to be available in the VM
	mountCmd := fmt.Sprintf("mount -t virtiofs -o allow_other,default_permissions sear_share %s", mountPoint)
	if err := sshClient.ExecuteCommand(mountCmd); err != nil {
		// Fallback to bind mount if virtiofs is not available
		logrus.Warnf("virtiofs mount failed, trying bind mount: %v", err)
		bindCmd := fmt.Sprintf("mount --bind %s %s", absPath, mountPoint)
		if err := sshClient.ExecuteCommand(bindCmd); err != nil {
			return fmt.Errorf("failed to mount directory: %w", err)
		}
	}

	return nil
}

// ExecuteCommand executes a command in the VM
func (v *VM) ExecuteCommand(cmd string) error {
	sshClient, err := v.GetSSHClient()
	if err != nil {
		return err
	}
	return sshClient.ExecuteCommand(cmd)
}

// getEffectiveNetworkConfig returns the effective network configuration
func (v *VM) getEffectiveNetworkConfig() *config.NetworkConfig {
	if v.profile.Network != nil {
		return v.profile.Network
	}
	return &config.NetworkConfig{
		TAPDevice: "tap0",
		TAPIP:     "172.16.0.1",
		GuestIP:   "172.16.0.2",
		GatewayIP: "172.16.0.1",
		DNSServer: "1.1.1.1",
	}
}
