package fingerprint

import (
	"fmt"
	"os"
	"runtime"
)

type SystemInfo struct {
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
}

func GetSystemInfo() (SystemInfo, error) {
	name, err := os.Hostname()
	if err != nil {
		return SystemInfo{}, fmt.Errorf("failed to get hostname: %v", err)
	}

	info := SystemInfo{
		Arch:     runtime.GOARCH,
		Hostname: name,
	}

	return info, nil
}
