package discovery

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lunarhue/libs-go/log"
	zeroconf "github.com/lunarhue/metallic-flock-zeroconf"
	"github.com/lunarhue/metallic-flock/pkg/k3s"
)

func RunControllerMode(NodeID string, Port uint16, callback func(ip string, role string)) {
	log.Info("State: CONTROLLER. Managing Cluster...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Infof("Starting K3s Server...")
	if err := k3s.StartK3sServer(ctx); err != nil {
		log.Panicf("Error: %v", err)
	}
	log.Infof("K3s Server started successfully.")

	me := zeroconf.NewService(TypeController, NodeID, Port)

	onNodeFound := func(e zeroconf.Event) {
		log.Infof("[DISCOVERY] Saw Service: %s | Operation: %v", e.Name, e.Op)

		if e.Op == zeroconf.OpAdded && len(e.Addrs) > 0 {
			meta := parseMetadata(e.Text)

			log.Infof("------------------------------------------------")
			log.Infof("   CANDIDATE FOUND: %s", e.Name)
			log.Infof("   IP:   %s", e.Addrs[0])
			log.Infof("   OS:   %s (%s)", meta["os"], meta["distro"])
			log.Infof("   HW:   %s Threads / %s GB RAM / %s GB Disk", meta["cpu"], meta["mem"], meta["disk"])
			log.Infof("   MAC:  %s", meta["mac"])
			log.Infof("------------------------------------------------")

			log.Infof("Found new node: %s [%v]. Auto-adopting...", e.Name, e.Addrs)

			ip := e.Addrs[0].String()

			go callback(ip, "agent")
		}
	}

	client, err := zeroconf.New().
		Publish(me).
		Browse(onNodeFound, TypePending).
		Open()

	if err != nil {
		log.Panicf("Failed to start zeroconf: %v", err)
	}
	defer client.Close()

	log.Info("Controller Beacon Active & Scanning...")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Info("Shutting down controller...")
}

func parseMetadata(txtRecords []string) map[string]string {
	data := make(map[string]string)
	for _, record := range txtRecords {
		if key, val, found := strings.Cut(record, "="); found {
			data[key] = val
		}
	}
	return data
}
