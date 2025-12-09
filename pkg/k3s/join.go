package k3s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	// Standard K3s config location
	DefaultConfigPath = "/etc/rancher/k3s/config.yaml"
	// Location where K3s stores node identity and creds
	AgentDataDir = "/var/lib/rancher/k3s/agent"
)

func StartAgent(ctx context.Context, serverIP string, token string) error {
	binPath, err := exec.LookPath("k3s")
	if err != nil {
		return fmt.Errorf("k3s binary not found in PATH: %w", err)
	}

	// 2. Construct the Server URL
	// K3s defaults to port 6443 for the API server
	serverURL := fmt.Sprintf("https://%s:6443", serverIP)

	// 3. Prepare arguments
	// Command: k3s agent --server https://<IP>:6443 --token <TOKEN>
	args := []string{
		"agent",
		"--server", serverURL,
		"--token", token,
	}

	// 4. Create Command with Context
	// If ctx is cancelled, Go will send a kill signal to the process
	cmd := exec.CommandContext(ctx, binPath, args...)

	// 5. Pipe Output
	// Useful to see K3s logs in your main application's stdout
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Starting K3s Agent (Server: %s)...\n", serverURL)

	// 6. Run and Wait
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	// This blocks until the process exits
	return cmd.Wait()
}

func writeK3sConfig(ip, token string) error {
	// 1. Ensure the directory exists
	// On NixOS, /etc/rancher usually doesn't exist by default unless created by the service.
	configDir := filepath.Dir(DefaultConfigPath) // /etc/rancher/k3s
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	// 2. Generate the YAML content
	// We include 'node-name' to ensure hostname consistency if NixOS changes hostname logic
	hostname, _ := os.Hostname()
	configContent := fmt.Sprintf(`server: https://%s:6443
token: %s
node-name: %s
`, ip, token, hostname)

	// 3. Write the file
	if err := os.WriteFile(DefaultConfigPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
