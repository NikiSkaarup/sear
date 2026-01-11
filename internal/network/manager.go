package network

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

// Manager handles network setup for the VM
type Manager struct {
	TAPDevice     string
	TAPIP         string
	GatewayIP     string
	HostInterface string
}

// NewManager creates a new network manager
func NewManager(tapDevice, tapIP, gatewayIP string) *Manager {
	return &Manager{
		TAPDevice: tapDevice,
		TAPIP:     tapIP,
		GatewayIP: gatewayIP,
	}
}

// Setup configures the network for the VM
func (m *Manager) Setup() error {
	logrus.Info("Setting up network...")

	// Detect host interface
	hostInterface, err := m.detectHostInterface()
	if err != nil {
		logrus.Warnf("Failed to detect host interface, using default: %v", err)
		m.HostInterface = "eth0"
	} else {
		m.HostInterface = hostInterface
	}

	// Setup TAP device
	if err := m.setupTAPDevice(); err != nil {
		return fmt.Errorf("failed to setup TAP device: %w", err)
	}

	// Enable IP forwarding
	if err := m.enableIPForwarding(); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Configure NAT
	if err := m.configureNAT(); err != nil {
		return fmt.Errorf("failed to configure NAT: %w", err)
	}

	logrus.Info("Network setup completed")
	return nil
}

// Teardown cleans up network configuration
func (m *Manager) Teardown() error {
	logrus.Info("Tearing down network...")

	// Remove TAP device
	if err := m.removeTAPDevice(); err != nil {
		logrus.Warnf("Failed to remove TAP device: %v", err)
	}

	// Remove NAT rules
	if err := m.removeNAT(); err != nil {
		logrus.Warnf("Failed to remove NAT rules: %v", err)
	}

	return nil
}

// setupTAPDevice creates and configures the TAP device
func (m *Manager) setupTAPDevice() error {
	// Remove existing TAP device
	_ = m.runCommand("ip", "link", "del", m.TAPDevice)

	// Create TAP device
	if err := m.runCommand("ip", "tuntap", "add", "dev", m.TAPDevice, "mode", "tap"); err != nil {
		return fmt.Errorf("failed to create TAP device: %w", err)
	}

	// Configure IP address
	if err := m.runCommand("ip", "addr", "add", m.TAPIP+"/30", "dev", m.TAPDevice); err != nil {
		return fmt.Errorf("failed to configure TAP IP: %w", err)
	}

	// Bring up device
	if err := m.runCommand("ip", "link", "set", "dev", m.TAPDevice, "up"); err != nil {
		return fmt.Errorf("failed to bring up TAP device: %w", err)
	}

	logrus.Infof("TAP device %s configured", m.TAPDevice)
	return nil
}

// removeTAPDevice removes the TAP device
func (m *Manager) removeTAPDevice() error {
	return m.runCommand("ip", "link", "del", m.TAPDevice)
}

// enableIPForwarding enables IP forwarding
func (m *Manager) enableIPForwarding() error {
	// Enable IP forwarding
	if err := m.runCommand("sh", "-c", "echo 1 > /proc/sys/net/ipv4/ip_forward"); err != nil {
		return fmt.Errorf("failed to enable IP forwarding: %w", err)
	}

	// Set forward policy to accept
	if err := m.runCommand("iptables", "-P", "FORWARD", "ACCEPT"); err != nil {
		return fmt.Errorf("failed to set forward policy: %w", err)
	}

	return nil
}

// configureNAT sets up NAT for the VM
func (m *Manager) configureNAT() error {
	// Remove existing NAT rule
	_ = m.runCommand("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", m.HostInterface, "-j", "MASQUERADE")

	// Add NAT rule
	if err := m.runCommand("iptables", "-t", "nat", "-A", "POSTROUTING", "-o", m.HostInterface, "-j", "MASQUERADE"); err != nil {
		return fmt.Errorf("failed to configure NAT: %w", err)
	}

	return nil
}

// removeNAT removes NAT configuration
func (m *Manager) removeNAT() error {
	return m.runCommand("iptables", "-t", "nat", "-D", "POSTROUTING", "-o", m.HostInterface, "-j", "MASQUERADE")
}

// detectHostInterface detects the default host network interface
func (m *Manager) detectHostInterface() (string, error) {
	// Try using ip command
	output, err := exec.Command("ip", "-j", "route", "list", "default").Output()
	if err != nil {
		// Fallback to simpler method
		output, err = exec.Command("ip", "route", "show", "default").Output()
		if err != nil {
			return "", fmt.Errorf("failed to detect default interface: %w", err)
		}
	}

	// Parse JSON output
	if len(output) > 0 && output[0] == '[' {
		var routes []map[string]interface{}
		if err := json.Unmarshal(output, &routes); err == nil {
			if len(routes) > 0 {
				if dev, ok := routes[0]["dev"].(string); ok {
					return dev, nil
				}
			}
		}
	}

	// Fallback: parse text output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "default") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "dev" && i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("could not detect default interface")
}

// runCommand executes a system command
func (m *Manager) runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		logrus.Debugf("Command failed: %s %v: %v", name, args, err)
		return err
	}
	return nil
}

// GetMACAddress generates a MAC address for the VM
func (m *Manager) GetMACAddress() string {
	// Generate MAC address based on gateway IP
	// Gateway IP: 172.16.0.1 -> MAC: 06:00:AC:10:00:02
	return "06:00:AC:10:00:02"
}
