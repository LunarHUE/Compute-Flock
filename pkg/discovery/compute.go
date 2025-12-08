package discovery

import (
	"time"

	"github.com/lunarhue/libs-go/log"
)

func RunComputeMode() {
	log.Info("State: COMPUTE. Connecting to Cluster...")

	// Survivability Loop
	for {
		controllerIP := ScanForControllers(5 * time.Second)

		if controllerIP == "" {
			log.Info("Lost Controller! Scanning...")
		} else {
			log.Infof("Connected to Controller at %s", controllerIP)
		}

		time.Sleep(10 * time.Second)
	}
}
