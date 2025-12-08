package discovery

import (
	"github.com/lunarhue/libs-go/log"
)

func RunPendingMode(NodeID string, Port uint16) {
	log.Info("State: PENDING. Broadcasting availability...")

	// 1. Advertise ourselves
	client, err := StartAgentBroadcast(NodeID, Port)
	if err != nil {
		log.Panicf("Failed to start broadcast: %v", err)
	}
	defer client.Close()

	// 2. Wait indefinitely
	select {}
}
