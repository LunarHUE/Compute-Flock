package k3s

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/lunarhue/libs-go/log"
)

func JoinCluster(serverURL string, token string) error {
	// 1. Stop the systemd service
	fmt.Println("Stopping k3s service...")
	stopCmd := exec.Command("systemctl", "stop", "k3s")
	stopCmd.Stdout = os.Stdout
	stopCmd.Stderr = os.Stderr
	// Ignore errors here in case the service is already stopped
	_ = stopCmd.Run()

	// 2. Locate k3s binary
	binPath, err := exec.LookPath("k3s")
	if err != nil {
		return fmt.Errorf("k3s binary not found in PATH: %w", err)
	}

	// 3. Prepare the Context with a Timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 4. Prepare the Agent Command
	args := []string{
		"agent",
		"--server", serverURL,
		"--token", token,
	}

	// CommandContext will kill the process when ctx expires (10 seconds)
	cmd := exec.CommandContext(ctx, binPath, args...)

	finishLog, err := log.LogCommand(cmd, "K3S-AGENT")
	if err != nil {
		return err
	}

	log.Infof("Starting temporary K3s Agent (10 second run) against %s...\n", serverURL)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start temporary agent: %w", err)
	}

	// 5. Wait for the timeout
	// cmd.Wait() will return an error when the process is killed by the context.
	err = cmd.Wait()

	finishLog()

	// Check why it stopped
	if ctx.Err() == context.DeadlineExceeded {
		log.Infof("\n10 seconds reached. Process killed successfully.")
	} else if err != nil {
		// If it crashed *before* the 10 seconds were up
		log.Warnf("\nWarning: Agent exited early or failed: %v\n", err)
	}

	// 6. Restart the systemd service
	log.Infof("Restarting k3s systemd service...")
	restartCmd := exec.Command("systemctl", "restart", "k3s")
	restartCmd.Stdout = os.Stdout
	restartCmd.Stderr = os.Stderr

	if err := restartCmd.Run(); err != nil {
		return fmt.Errorf("failed to restart k3s service: %w", err)
	}

	log.Infof("K3s service restarted successfully.")

	return nil
}
