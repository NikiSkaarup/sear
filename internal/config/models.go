package config

// Config represents the main configuration structure
type Config struct {
	DefaultProfile string             `yaml:"default_profile"`
	Profiles       map[string]Profile `yaml:"profiles"`
	Network        *NetworkConfig     `yaml:"network,omitempty"`
	SSH            *SSHConfig         `yaml:"ssh,omitempty"`
}

// Profile represents a VM profile configuration
type Profile struct {
	VM      VMConfig       `yaml:"vm"`
	Tools   []string       `yaml:"tools"`
	Network *NetworkConfig `yaml:"network,omitempty"`
}

// VMConfig represents Firecracker VM configuration
type VMConfig struct {
	VCPUs      int    `mapstructure:"vcpus" yaml:"vcpus"`
	MemoryMiB  int    `mapstructure:"memory_mib" yaml:"memory_mib"`
	RootFS     string `mapstructure:"rootfs" yaml:"rootfs"`
	Kernel     string `mapstructure:"kernel" yaml:"kernel"`
	KernelArgs string `mapstructure:"kernel_args" yaml:"kernel_args"`
}

// NetworkConfig represents network configuration
type NetworkConfig struct {
	TAPDevice     string `yaml:"tap_device,omitempty"`
	TAPIP         string `yaml:"tap_ip,omitempty"`
	GuestIP       string `yaml:"guest_ip,omitempty"`
	GatewayIP     string `yaml:"gateway_ip,omitempty"`
	HostInterface string `yaml:"host_interface,omitempty"`
	DNSServer     string `yaml:"dns_server,omitempty"`
}

// SSHConfig represents SSH configuration
type SSHConfig struct {
	KeyPath  string `yaml:"key_path,omitempty"`
	Username string `yaml:"username,omitempty"`
}
