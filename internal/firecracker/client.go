package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Client represents a Firecracker API client
type Client struct {
	socketPath string
	httpClient *http.Client
}

// NewClient creates a new Firecracker API client
func NewClient(socketPath string) (*Client, error) {
	// Verify socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Firecracker socket not found: %s", socketPath)
	}

	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}, nil
}

// request makes an HTTP request to the Firecracker API via Unix socket
func (c *Client) request(method, endpoint string, data interface{}) error {
	u := &url.URL{
		Scheme: "http",
		Host:   "localhost",
		Path:   endpoint,
	}

	var body []byte
	if data != nil {
		var err error
		body, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	req, err := http.NewRequest(method, u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Use Unix socket transport
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", c.socketPath)
		},
	}

	client := &http.Client{Transport: transport}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// ConfigureLogger sets up Firecracker logging
func (c *Client) ConfigureLogger(logPath, level string) error {
	logrus.Infof("Configuring Firecracker logger: %s", logPath)

	data := map[string]interface{}{
		"log_path":        logPath,
		"level":           level,
		"show_level":      true,
		"show_log_origin": true,
	}

	return c.request("PUT", "/logger", data)
}

// SetBootSource sets the kernel and boot arguments
func (c *Client) SetBootSource(kernelPath, bootArgs string) error {
	logrus.Infof("Setting boot source: %s", kernelPath)

	// Expand home directory
	kernelPath = expandPath(kernelPath)

	data := map[string]interface{}{
		"kernel_image_path": kernelPath,
		"boot_args":         bootArgs,
	}

	return c.request("PUT", "/boot-source", data)
}

// AttachRootfs attaches a drive to the VM
func (c *Client) AttachRootfs(driveID, pathOnHost string, isRoot, isReadOnly bool) error {
	logrus.Infof("Attaching rootfs: %s", pathOnHost)

	// Expand home directory
	pathOnHost = expandPath(pathOnHost)

	data := map[string]interface{}{
		"drive_id":       driveID,
		"path_on_host":   pathOnHost,
		"is_root_device": isRoot,
		"is_read_only":   isReadOnly,
	}

	return c.request("PUT", fmt.Sprintf("/drives/%s", driveID), data)
}

// AttachNetwork attaches a network interface
func (c *Client) AttachNetwork(interfaceID, guestMAC, hostDevName string) error {
	logrus.Infof("Attaching network interface: %s", hostDevName)

	data := map[string]interface{}{
		"iface_id":      interfaceID,
		"guest_mac":     guestMAC,
		"host_dev_name": hostDevName,
	}

	return c.request("PUT", fmt.Sprintf("/network-interfaces/%s", interfaceID), data)
}

// StartInstance starts the Firecracker instance
func (c *Client) StartInstance() error {
	logrus.Info("Starting Firecracker instance")

	data := map[string]interface{}{
		"action_type": "InstanceStart",
	}

	return c.request("PUT", "/actions", data)
}

// Helper function to expand home directory
func expandPath(path string) string {
	if len(path) > 1 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
