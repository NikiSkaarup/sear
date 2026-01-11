package cmd

import (
	"fmt"
	"os"

	"github.com/nikiskaarup/sear/internal/config"
	"github.com/nikiskaarup/sear/internal/vm"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [profile-name]",
	Short: "Run a microVM with the specified profile",
	Long: `Spawn a Firecracker microVM with the given profile and provide
an interactive shell with the current working directory mounted.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		profileName := args[0]
		return runProfile(profileName)
	},
}

func runProfile(profileName string) error {
	logrus.Infof("Starting profile: %s", profileName)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate profile exists
	profile, exists := cfg.Profiles[profileName]
	if !exists {
		return fmt.Errorf("profile '%s' not found. Available profiles: %v", profileName, getProfileNames(cfg))
	}

	// Create and start VM
	vmInstance, err := vm.NewVM(profile)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	// Start the VM
	if err := vmInstance.Start(); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Ensure cleanup on exit
	defer func() {
		if err := vmInstance.Stop(); err != nil {
			logrus.Errorf("Error stopping VM: %v", err)
		}
	}()

	logrus.Info("VM started successfully")

	// Get SSH client for the VM
	sshClient, err := vmInstance.GetSSHClient()
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	// Configure guest networking
	if err := configureGuestNetworking(sshClient, profile); err != nil {
		logrus.Warnf("Failed to configure guest networking: %v", err)
	}

	// Run tool commands
	if err := runToolCommands(sshClient, profile.Tools); err != nil {
		logrus.Warnf("Some tool commands failed: %v", err)
	}

	// Mount current directory
	cwd, err := os.Getwd()
	if err != nil {
		logrus.Warnf("Failed to get current directory: %v", err)
	} else {
		if err := vmInstance.MountDirectory(sshClient, cwd); err != nil {
			logrus.Warnf("Failed to mount current directory: %v", err)
		} else {
			logrus.Infof("Mounted current directory: %s", cwd)
		}
	}

	// Start interactive shell
	logrus.Info("Starting interactive shell...")
	return sshClient.Shell()
}

func configureGuestNetworking(sshClient *vm.SSHClient, profile config.Profile) error {
	logrus.Info("Configuring guest networking...")

	// Setup DNS (use 1.1.1.1 as default)
	dnsServer := "1.1.1.1"
	if profile.Network != nil && profile.Network.DNSServer != "" {
		dnsServer = profile.Network.DNSServer
	}

	commands := []string{
		// Setup DNS
		fmt.Sprintf("echo 'nameserver %s' > /etc/resolv.conf", dnsServer),
		// Setup default route
		"ip route add default via 172.16.0.1 dev eth0 2>/dev/null || true",
	}

	for _, cmd := range commands {
		if err := sshClient.ExecuteCommand(cmd); err != nil {
			logrus.Warnf("Command failed: %s: %v", cmd, err)
		}
	}

	return nil
}

func runToolCommands(sshClient *vm.SSHClient, tools []string) error {
	if len(tools) == 0 {
		logrus.Info("No tools to run")
		return nil
	}

	logrus.Infof("Running %d tool commands...", len(tools))

	for i, toolCmd := range tools {
		logrus.Infof("Running tool %d/%d: %s", i+1, len(tools), toolCmd)
		if err := sshClient.ExecuteCommand(toolCmd); err != nil {
			logrus.Warnf("Tool command failed: %v", err)
		}
	}

	return nil
}

func getProfileNames(cfg *config.Config) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	return names
}
