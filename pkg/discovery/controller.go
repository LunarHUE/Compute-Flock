package discovery

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	zeroconf "github.com/lunarhue/compute-flock-zeroconf"
	"github.com/lunarhue/libs-go/log"
)

func RunControllerMode(NodeID string, Port uint16, callback func(ip string, role string)) {
	log.Info("State: CONTROLLER. Managing Cluster...")

	// 1. Define Myself (The Controller Service)
	me := zeroconf.NewService(TypeController, NodeID, Port)

	// 2. Define the Callback for finding new nodes
	onNodeFound := func(e zeroconf.Event) {
		// Only react to "Added" events with valid IPs
		if e.Op == zeroconf.OpAdded && len(e.Addrs) > 0 {
			meta := parseMetadata(e.Text)

			log.Infof("------------------------------------------------")
			log.Infof("   CANDIDATE FOUND: %s\n", e.Name)
			log.Infof("   IP:   %s\n", e.Addrs[0])
			log.Infof("   OS:   %s (%s)\n", meta["os"], meta["distro"])
			log.Infof("   HW:   %s Cores / %s GB RAM / %s GB Disk\n", meta["cpu"], meta["mem"], meta["disk"])
			log.Infof("   MAC:  %s\n", meta["mac"])
			log.Infof("------------------------------------------------")

			log.Infof("Found new node: %s [%v]. Auto-adopting...", e.Name, e.Addrs)

			// Extract IP (prefer IPv4)
			ip := e.Addrs[0].String()
			// Note: You might want to iterate e.Addrs to find the IPv4 one specifically
			// if the network has both.

			go callback(ip, "agent")
		}
	}

	// 3. Start the Engine (Publish Myself + Browse for Others)
	client, err := zeroconf.New().
		Publish(me).                      // "I am the Controller"
		Browse(onNodeFound, TypePending). // "Look for Pending Nodes"
		Open()

	if err != nil {
		log.Panicf("Failed to start zeroconf: %v", err)
	}
	defer client.Close()

	log.Info("Controller Beacon Active & Scanning...")
	// 4. Block forever (or until signal)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Info("Shutting down controller...")
}

func parseMetadata(txtRecords []string) map[string]string {
	data := make(map[string]string)
	for _, record := range txtRecords {
		// Split only on the first "="
		if key, val, found := strings.Cut(record, "="); found {
			data[key] = val
		}
	}
	return data
}
