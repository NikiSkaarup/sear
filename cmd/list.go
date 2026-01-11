package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nikiskaarup/sear/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list-profiles",
	Short: "List all available profiles",
	Long:  "Display all configured profiles with their details.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return listProfiles()
	},
}

func listProfiles() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured.")
		return nil
	}

	// Determine default profile
	defaultProfile := cfg.DefaultProfile
	if defaultProfile == "" {
		defaultProfile = "none"
	}

	fmt.Printf("Default profile: %s\n\n", defaultProfile)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "Profile\tVCPUs\tMemory (MiB)\tRootFS\tTools\n")
	fmt.Fprintf(w, "-------\t-----\t------------\t------\t-----\n")

	for name, profile := range cfg.Profiles {
		vcpus := "1"
		memory := "512"
		rootfs := "default"
		tools := "0"

		if profile.VM.VCPUs > 0 {
			vcpus = fmt.Sprintf("%d", profile.VM.VCPUs)
		}
		if profile.VM.MemoryMiB > 0 {
			memory = fmt.Sprintf("%d", profile.VM.MemoryMiB)
		}
		if profile.VM.RootFS != "" {
			rootfs = profile.VM.RootFS
		}
		if len(profile.Tools) > 0 {
			tools = fmt.Sprintf("%d", len(profile.Tools))
		}

		marker := ""
		if name == defaultProfile {
			marker = " *"
		}

		fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\n", name, marker, vcpus, memory, rootfs, tools)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	logrus.Debug("Profile list displayed successfully")
	return nil
}
