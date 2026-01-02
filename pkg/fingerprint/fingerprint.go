package fingerprint

import (
	_ "embed"
	"fmt"
)

type Fingerprint struct {
	System  SystemInfo             `json:"system"`
	Cpus    []CpuInfo              `json:"cpus"`
	Memory  MemoryInfo             `json:"memory"`
	Network []NetworkInterfaceInfo `json:"network"`
	Storage []StorageInfo          `json:"storage"`
}

func GetFingerprint() (Fingerprint, error) {
	systemInfo, err := GetSystemInfo()
	if err != nil {
		return Fingerprint{}, fmt.Errorf("failed to get system info: %v", err)
	}

	cpus, err := GetCpus()
	if err != nil {
		return Fingerprint{}, fmt.Errorf("failed to get cpus: %v", err)
	}

	memoryInfo, err := GetMemoryInfo()
	if err != nil {
		return Fingerprint{}, fmt.Errorf("failed to get memory info: %v", err)
	}

	networkInterfaces, err := GetNetworkInterfaces()
	if err != nil {
		return Fingerprint{}, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	storageDevices, err := GetStorageDevices()
	if err != nil {
		return Fingerprint{}, fmt.Errorf("failed to get storage devices: %v", err)
	}

	fingerprint := Fingerprint{
		System:  systemInfo,
		Cpus:    cpus,
		Memory:  memoryInfo,
		Network: networkInterfaces,
		Storage: storageDevices,
	}

	return fingerprint, nil
}
