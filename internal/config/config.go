package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Config file locations
	configHome := getConfigHome()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configHome)
	v.AddConfigPath(".")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found. Searched in: %s/config.yaml", configHome)
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	// Apply environment variable overrides
	applyEnvOverrides(&cfg)

	logrus.Debugf("Loaded configuration with %d profiles", len(cfg.Profiles))
	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Network defaults
	v.SetDefault("network.tap_device", "tap0")
	v.SetDefault("network.tap_ip", "172.16.0.1")
	v.SetDefault("network.guest_ip", "172.16.0.2")
	v.SetDefault("network.gateway_ip", "172.16.0.1")
	v.SetDefault("network.dns_server", "1.1.1.1")

	// SSH defaults
	v.SetDefault("ssh.key_path", "sear_key")
	v.SetDefault("ssh.username", "root")
}

func getConfigHome() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return xdg
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "$HOME/.config"
	}
	return filepath.Join(home, ".config", "sear")
}

func applyEnvOverrides(cfg *Config) {
	// Override from environment if set
	if tapDevice := os.Getenv("SEAR_TAP_DEVICE"); tapDevice != "" {
		if cfg.Network == nil {
			cfg.Network = &NetworkConfig{}
		}
		cfg.Network.TAPDevice = tapDevice
	}

	if sshKey := os.Getenv("SEAR_SSH_KEY"); sshKey != "" {
		if cfg.SSH == nil {
			cfg.SSH = &SSHConfig{}
		}
		cfg.SSH.KeyPath = sshKey
	}
}
