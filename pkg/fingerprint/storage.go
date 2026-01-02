package fingerprint

import (
	"fmt"

	"github.com/jaypipes/ghw"
)

type StorageInfo struct {
	DeviceName string `json:"device_name"`
	Model      string `json:"model"`
	SizeBytes  int64  `json:"size_bytes"`
}

func GetStorageDevices() ([]StorageInfo, error) {
	block, err := ghw.Block()
	if err != nil {
		return nil, fmt.Errorf("error getting storage info: %v", err)
	}

	var devices []StorageInfo
	for _, disk := range block.Disks {
		devices = append(devices, StorageInfo{
			DeviceName: disk.Name,
			Model:      disk.Model,
			SizeBytes:  int64(disk.SizeBytes),
		})
	}

	return devices, nil
}
