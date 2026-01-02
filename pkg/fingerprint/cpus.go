package fingerprint

import (
	"fmt"

	"github.com/jaypipes/ghw"
)

type CpuInfo struct {
	Vendor  string `json:"vendor"`
	Model   string `json:"model"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
}

func GetCpus() ([]CpuInfo, error) {
	cpu, err := ghw.CPU()
	if err != nil {
		return nil, fmt.Errorf("error getting cpu info: %v", err)
	}

	var cpus []CpuInfo
	for _, proc := range cpu.Processors {
		cpus = append(cpus, CpuInfo{
			Vendor:  proc.Vendor,
			Model:   proc.Model,
			Cores:   int(proc.NumCores),
			Threads: int(proc.NumThreads),
		})
	}

	return cpus, nil
}
