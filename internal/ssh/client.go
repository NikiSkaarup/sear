package ssh

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

// Client represents an SSH client for connecting to the VM
type Client struct {
	host       string
	port       int
	username   string
	privateKey string
}

// NewClient creates a new SSH client
func NewClient(host string, port int, username, privateKey string) *Client {
	return &Client{
		host:       host,
		port:       port,
		username:   username,
		privateKey: privateKey,
	}
}

// Connect establishes an SSH connection
func (c *Client) Connect() (*ssh.Client, error) {
	keyPath := c.expandPath(c.privateKey)

	// Read private key
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// SSH client configuration
	config := &ssh.ClientConfig{
		User: c.username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return client, nil
}

// ExecuteCommand executes a command on the VM
func (c *Client) ExecuteCommand(cmd string) error {
	client, err := c.Connect()
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Run command
	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return fmt.Errorf("command failed: %w, output: %s", err, output)
	}

	return nil
}

// Shell starts an interactive shell session
func (c *Client) Shell() error {
	client, err := c.Connect()
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Set up PTY
	if err := session.RequestPty("xterm", 80, 40, ssh.TerminalModes{
		ssh.ECHO: 1,
	}); err != nil {
		return fmt.Errorf("failed to request PTY: %w", err)
	}

	// Start interactive shell
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	return session.Shell()
}

// expandPath expands home directory in path
func (c *Client) expandPath(path string) string {
	if len(path) > 1 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}
