package fingerprint

import (
	"fmt"

	"github.com/jaypipes/ghw"
)

type MemoryInfo struct {
	TotalBytes int64 `json:"total_bytes"`
	DimmCount  int   `json:"dimm_count"`
}

func GetMemoryInfo() (MemoryInfo, error) {
	mem, err := ghw.Memory()
	if err != nil {
		return MemoryInfo{}, fmt.Errorf("error getting memory info: %v", err)
	}

	return MemoryInfo{
		TotalBytes: mem.TotalPhysicalBytes,
		DimmCount:  len(mem.Modules), // Counts physical sticks of RAM
	}, nil
}
