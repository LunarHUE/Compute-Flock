package k3s

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/lunarhue/libs-go/log"
)

func StartAgent(serverURL string, token string) error {
	const unitName = "k3s-agent.service"

	binPath, err := exec.LookPath("k3s")
	if err != nil {
		return fmt.Errorf("k3s binary not found in PATH: %w", err)
	}

	log.Infof("Ensuring previous %s is stopped...", unitName)
	_ = exec.Command("systemctl", "stop", unitName).Run()

	// 3. Construct the systemd-run command
	// This creates a service named 'k3s-agent' that restarts automatically if it fails.
	// equivalent to: systemd-run --unit=k3s-agent -p Restart=always k3s agent ...
	cmd := exec.Command("systemd-run",
		"--unit="+unitName,
		"--description=K3s Agent (Transient)",
		"-p", "Restart=always", // Auto-restart if it crashes
		"-p", "RestartSec=10", // Wait 10s before restarting
		binPath, // Command to run
		"agent",
		"--server", serverURL,
		"--token", token,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Infof("Spawning K3s Agent via systemd-run against %s...", serverURL)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to spawn k3s-agent service: %w", err)
	}

	// 4. Verification (Optional but recommended)
	// Give systemd a moment to spin it up and check status
	time.Sleep(1 * time.Second)
	if err := checkServiceRunning(unitName); err != nil {
		return fmt.Errorf("agent service failed to start: %w", err)
	}

	log.Infof("SUCCESS: K3s Agent is running in background unit '%s'", unitName)
	return nil
}

func checkServiceRunning(unitName string) error {
	cmd := exec.Command("systemctl", "is-active", unitName)
	output, _ := cmd.Output()
	state := string(output)

	if len(state) > 0 {
		state = state[:len(state)-1] // trim newline
	}

	if state == "active" || state == "activating" {
		return nil
	}

	logs, _ := exec.Command("journalctl", "-u", unitName, "-n", "10", "--no-pager").CombinedOutput()
	return fmt.Errorf("service state is '%s'. Recent logs:\n%s", state, string(logs))
}
