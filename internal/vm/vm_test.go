//go:build integration
// +build integration

package vm_test

import (
	"os"
	"testing"
	"time"

	"github.com/nikiskaarup/sear/internal/config"
	"github.com/nikiskaarup/sear/internal/vm"
)

func TestVMSpawn(t *testing.T) {
	// Skip if Firecracker is not available
	socketPath := os.Getenv("FIRECRACKER_API_SOCKET")
	if socketPath == "" {
		socketPath = "/tmp/firecracker.socket"
	}

	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		t.Skip("Firecracker socket not found, skipping integration test")
	}

	// Create test profile
	profile := config.Profile{
		VM: config.VMConfig{
			VCPUs:     1,
			MemoryMiB: 512,
			RootFS:    os.Getenv("TEST_ROOTFS"),
			Kernel:    os.Getenv("TEST_KERNEL"),
		},
		Tools: []string{},
	}

	// Skip if rootfs or kernel not available
	if profile.VM.RootFS == "" || profile.VM.Kernel == "" {
		t.Skip("TEST_ROOTFS or TEST_KERNEL not set, skipping integration test")
	}

	// Create VM
	vmInstance, err := vm.NewVM(profile)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	// Start VM
	if err := vmInstance.Start(); err != nil {
		t.Fatalf("Failed to start VM: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if err := vmInstance.Stop(); err != nil {
			t.Logf("Error stopping VM: %v", err)
		}
	}()

	// Wait for VM to be ready
	time.Sleep(2 * time.Second)

	// Test SSH connection
	sshClient, err := vmInstance.GetSSHClient()
	if err != nil {
		t.Fatalf("Failed to get SSH client: %v", err)
	}

	// Execute test command
	if err := sshClient.ExecuteCommand("echo 'Hello from VM'"); err != nil {
		t.Fatalf("Failed to execute command: %v", err)
	}

	t.Log("VM test completed successfully")
}

func TestVMRunProfile(t *testing.T) {
	// Test with minimal profile
	profile := config.Profile{
		VM: config.VMConfig{
			VCPUs:     1,
			MemoryMiB: 512,
			RootFS:    os.Getenv("TEST_ROOTFS"),
			Kernel:    os.Getenv("TEST_KERNEL"),
		},
		Tools: []string{
			"echo 'Tool 1'",
			"echo 'Tool 2'",
		},
	}

	vmInstance, err := vm.NewVM(profile)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	if err := vmInstance.Start(); err != nil {
		t.Fatalf("Failed to start VM: %v", err)
	}

	defer vmInstance.Stop()

	time.Sleep(2 * time.Second)

	sshClient, err := vmInstance.GetSSHClient()
	if err != nil {
		t.Fatalf("Failed to get SSH client: %v", err)
	}

	// Test tool execution
	for i, tool := range profile.Tools {
		if err := sshClient.ExecuteCommand(tool); err != nil {
			t.Errorf("Tool %d failed: %v", i+1, err)
		}
	}

	t.Log("Profile test completed successfully")
}

func TestVMNetworkSetup(t *testing.T) {
	profile := config.Profile{
		VM: config.VMConfig{
			VCPUs:     1,
			MemoryMiB: 512,
			RootFS:    os.Getenv("TEST_ROOTFS"),
			Kernel:    os.Getenv("TEST_KERNEL"),
		},
		Network: &config.NetworkConfig{
			TAPDevice: "tap0",
			TAPIP:     "172.16.0.1",
			GuestIP:   "172.16.0.2",
			GatewayIP: "172.16.0.1",
			DNSServer: "1.1.1.1",
		},
	}

	vmInstance, err := vm.NewVM(profile)
	if err != nil {
		t.Fatalf("Failed to create VM: %v", err)
	}

	if err := vmInstance.Start(); err != nil {
		t.Fatalf("Failed to start VM: %v", err)
	}

	defer vmInstance.Stop()

	time.Sleep(2 * time.Second)

	sshClient, err := vmInstance.GetSSHClient()
	if err != nil {
		t.Fatalf("Failed to get SSH client: %v", err)
	}

	// Test DNS configuration
	if err := sshClient.ExecuteCommand("cat /etc/resolv.conf"); err != nil {
		t.Errorf("Failed to verify DNS configuration: %v", err)
	}

	t.Log("Network test completed successfully")
}
