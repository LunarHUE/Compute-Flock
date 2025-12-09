package k3s

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lunarhue/libs-go/log"
)

func VerifyK3sInstallation() error {
	if os.Geteuid() != 0 {
		log.Warnf("Not running as root. Firewall checks might fail.")
		// return fmt.Errorf("k3s verification requires root privileges")
	}

	if err := checkServiceActive("k3s"); err != nil {
		log.Panicf("[FAIL] K3s service check failed: %v", err)
	}
	log.Info("[OK] K3s service is active.")

	if err := checkDistribution(); err != nil {
		log.Panicf("[FAIL] Distribution check failed: %v", err)
	}
	log.Info("[OK] Supported Linux distribution detected.")

	if err := checkProcessArgs("k3s", "server"); err != nil {
		log.Panicf("[FAIL] K3s process argument check failed: %v", err)
	}
	log.Info("[OK] K3s process is running with expected arguments.")

	if err := checkFirewallPort("6443", "tcp"); err != nil {
		log.Panicf("[FAIL] Firewall port check failed: %v", err)
	}
	log.Info("[OK] Required firewall ports are open.")

	return nil
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

func checkServiceActive(serviceName string) error {
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("service check failed: %v", err)
	}

	status := strings.TrimSpace(string(output))
	if status != "active" {
		return fmt.Errorf("service is not active. Status returned: %s", status)
	}

	return nil
}

func checkProcessArgs(processName, expectedArg string) error {
	// pgrep -a lists the full command line
	cmd := exec.Command("pgrep", "-a", processName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("process '%s' is not running", processName)
	}

	outStr := string(output)
	if !strings.Contains(outStr, expectedArg) {
		return fmt.Errorf("process '%s' found, but argument '%s' is missing", processName, expectedArg)
	}
	return nil
}

func checkFirewallPort(port, protocol string) error {
	// NixOS puts user-defined allowedPorts in the 'nixos-fw' chain.
	// We use -n to avoid DNS lookups (speed) and grep for the port.
	cmd := exec.Command("iptables", "-L", "nixos-fw", "-n")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("could not inspect iptables (are you root?): %v", err)
	}

	// Output format typically includes "dpt:6443" or similar
	searchString := fmt.Sprintf("dpt:%s", port)

	// Also ensure protocol matches (e.g., checking if the line that has the port also has the protocol)
	lines := strings.Split(string(output), "\n")
	found := false

	for _, line := range lines {
		// We verify the line contains the port AND the protocol to avoid false positives
		if strings.Contains(line, searchString) && strings.Contains(strings.ToLower(line), strings.ToLower(protocol)) {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("port %s/%s not found in 'nixos-fw' chain", port, protocol)
	}
	return nil
}
