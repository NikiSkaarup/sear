package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var verbose bool

var rootCmd = &cobra.Command{
	Use:   "sear",
	Short: "Firecracker microVM orchestration tool",
	Long: `sear is a CLI tool to spawn Firecracker microVMs with configured profiles.

Complete documentation is available at https://github.com/nikiskaarup/sear`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(validateCmd)
}

func initConfig() {
	// Set up logging
	setupLogging()

	// Viper configuration
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	configHome := getConfigHome()
	viper.AddConfigPath(configHome)
	viper.AddConfigPath(".")

	// Enable environment variable override
	viper.AutomaticEnv()

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			logrus.Fatalf("Error reading config file: %v", err)
		}
		logrus.Warnf("Config file not found, using defaults")
	}
}

func setupLogging() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func getConfigHome() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg
	}
	return "$HOME/.config"
}
