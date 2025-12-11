package k3s

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lunarhue/libs-go/log"
)

func VerifyK3sInstallation(mode string) error {
	if os.Geteuid() != 0 {
		log.Warnf("Not running as root. Firewall checks might fail.")
		// return fmt.Errorf("k3s verification requires root privileges")
	}

	if err := checkDistribution(); err != nil {
		log.Panicf("[FAIL] Distribution check failed: %v", err)
	}
	log.Info("[OK] Supported Linux distribution detected.")

	if err := checkCommandExists("k3s"); err != nil {
		log.Panicf("[FAIL] K3s binary check failed: %v", err)
	}
	log.Info("[OK] K3s binary is present in PATH.")

	if mode == "server" {
		// In your new setup, the service should exist but might be stopped.
		// We check if it is 'loaded' rather than 'active'.
		err := checkServiceLoaded("k3s.service")
		if err != nil {
			log.Panicf("[FAIL] K3s service definition check failed: %v", err)
		}
		log.Info("[OK] K3s systemd unit is loaded and ready.")
	}

	if mode == "agent" {
		// Agents in your setup do NOT have a systemd service created by NixOS.
		// We verify that NO conflicting service exists.
		if err := checkServiceLoaded("k3s.service"); err == nil {
			log.Warnf("[WARNING] A 'k3s.service' was found but this node is an AGENT. This might cause conflicts if the service auto-starts.")
		}
	}

	if err := checkFirewallPort("6443", "tcp"); err != nil {
		log.Panicf("[FAIL] Firewall port check failed: %v", err)
	}
	log.Info("[OK] Required firewall ports are open.")

	return nil
}

func checkServiceLoaded(serviceName string) error {
	// "show -p LoadState" returns "LoadState=loaded" if the unit file is valid
	cmd := exec.Command("systemctl", "show", "-p", "LoadState", "--value", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query systemd: %v", err)
	}

	state := strings.TrimSpace(string(output))
	if state != "loaded" {
		return fmt.Errorf("unit %s is not loaded (state: %s). Is the NixOS config applied?", serviceName, state)
	}
	return nil
}

func checkFirewallPort(port, protocol string) error {
	cmd := exec.Command("iptables", "-L", "nixos-fw", "-n")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not inspect iptables: %v", err)
	}

	searchString := fmt.Sprintf("dpt:%s", port)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, searchString) && strings.Contains(strings.ToLower(line), strings.ToLower(protocol)) {
			return nil
		}
	}

	return fmt.Errorf("port %s/%s not found in 'nixos-fw' chain", port, protocol)
}

func checkDistribution() error {
	cmd := exec.Command("uname", "-a")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get system information: %v", err)
	}

	outStr := string(output)
	if !strings.Contains(outStr, "NixOS") {
		return fmt.Errorf("unsupported distribution: %s", outStr)
	}

	return nil
}

func checkCommandExists(cmdName string) error {
	_, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("command '%s' not found in PATH", cmdName)
	}
	return nil
}
