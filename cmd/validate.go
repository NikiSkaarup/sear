package cmd

import (
	"fmt"

	"github.com/nikiskaarup/sear/internal/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate-config",
	Short: "Validate the configuration file",
	Long:  "Check the configuration file for syntax errors and missing required fields.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return validateConfig()
	},
}

func validateConfig() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration file has errors: %w", err)
	}

	// Validate each profile
	if len(cfg.Profiles) == 0 {
		return fmt.Errorf("no profiles defined in configuration")
	}

	errors := make([]string, 0)

	for name, profile := range cfg.Profiles {
		if err := validateProfile(name, profile); err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("validation failed:\n%s", joinErrors(errors))
	}

	fmt.Println("✓ Configuration is valid")
	fmt.Printf("✓ Default profile: %s\n", cfg.DefaultProfile)
	fmt.Printf("✓ Total profiles: %d\n", len(cfg.Profiles))

	for name := range cfg.Profiles {
		fmt.Printf("  - %s\n", name)
	}

	return nil
}

func validateProfile(name string, profile config.Profile) error {
	// Check VM configuration
	if profile.VM.VCPUs <= 0 {
		return fmt.Errorf("profile '%s': VCPUs must be greater than 0", name)
	}
	if profile.VM.MemoryMiB <= 0 {
		return fmt.Errorf("profile '%s': MemoryMiB must be greater than 0", name)
	}

	return nil
}

func joinErrors(errors []string) string {
	result := ""
	for i, err := range errors {
		if i > 0 {
			result += "\n"
		}
		result += fmt.Sprintf("  • %s", err)
	}
	return result
}
