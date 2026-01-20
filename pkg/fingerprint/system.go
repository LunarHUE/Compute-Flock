package fingerprint

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

type SystemInfo struct {
	Arch       string `json:"arch"`
	Hostname   string `json:"hostname"`
	TpmVersion string `json:"tpm_version"`
}

func GetSystemInfo() (SystemInfo, error) {
	name, err := os.Hostname()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("failed to get hostname: %v", err)
	}

	info := SystemInfo{
		Arch:       runtime.GOARCH,
		Hostname:   name,
		TpmVersion: getLinuxTPMVersion(),
	}

	return info, nil
}

func getLinuxTPMVersion() string {
	// 1. Try the modern Kernel Sysfs interface (Kernel 5.6+)
	// This is the cleanest way on NixOS. It usually contains just "1" or "2".
	// Path: /sys/class/tpm/tpm0/tpm_version_major
	content, err := os.ReadFile("/sys/class/tpm/tpm0/tpm_version_major")
	if err == nil {
		v := strings.TrimSpace(string(content))
		switch v {
		case "1":
			return "1.2"
		case "2":
			return "2.0"
		}
	}

	// 2. Check for TPM 2.0 Resource Manager
	// If the kernel has created 'tpmrm0', it is definitely a TPM 2.0 chip
	// handling multiple contexts.
	if _, err := os.Stat("/dev/tpmrm0"); err == nil {
		return "2.0"
	}

	// 3. Check for raw TPM device
	// If we see tpm0 but missed the checks above, it's ambiguous,
	// but often implies older hardware or drivers (likely 1.2).
	if _, err := os.Stat("/dev/tpm0"); err == nil {
		// Attempt to read the 'caps' file if version_major didn't exist
		// This is common in older kernels.
		caps, err := os.ReadFile("/sys/class/tpm/tpm0/caps")
		if err == nil {
			capsStr := string(caps)
			if strings.Contains(capsStr, "TCG version: 1.2") {
				return "1.2"
			}
		}
		return "Unknown (Device exists, likely 1.2)"
	}

	return "None"
}
