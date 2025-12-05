package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lunarhue/compute-flock/pkg/discovery"
	"github.com/lunarhue/compute-flock/pkg/system"

	zeroconf "github.com/lunarhue/compute-flock-zeroconf"
)

// Global State
var (
	NodeID       string
	CurrentState = "PENDING" // PENDING, CONTROLLER, COMPUTE
	Port         = 9000
)

type server struct {
	pb.UnimplementedHiveServiceServer
}

// Handle Adoption Request
func (s *server) Adopt(ctx context.Context, req *pb.AdoptRequest) (*pb.AdoptResponse, error) {
	log.Printf("Received ADOPT command. Role: %s, Controller: %s", req.Role, req.Controller_ip)

	// 1. Write Config
	conf := system.K3sConfig{
		Role:         req.Role,
		Token:        req.Cluster_token,
		ControllerIP: req.Controller_ip,
		ClusterInit:  (req.Role == "server" && req.Controller_ip == ""), // simplified logic
	}

	// 2. Trigger NixOS Rebuild (Blocking operation)
	// In production, do this in a goroutine and return "Accepted" immediately.
	go func() {
		if err := system.WriteAndRebuild(conf); err != nil {
			log.Printf("Rebuild failed: %v", err)
			return
		}
		log.Println("Rebuild complete. Restarting...")
		os.Exit(0) // Restart to load new config
	}()

	return &pb.AdoptResponse{Success: true, Message: "Adoption started"}, nil
}

func (s *server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	return &pb.HeartbeatResponse{Reconfigure: false}, nil
}

// ---------------------------------------------------------
// Mode: CONTROLLER (The "Boss" State)
// ---------------------------------------------------------
func runControllerMode() {
	log.Println("State: CONTROLLER. Managing Cluster...")

	// 1. Define Myself (The Controller Service)
	me := zeroconf.NewService(discovery.TypeController, NodeID, uint16(Port))

	// 2. Define the Callback for finding new nodes
	onNodeFound := func(e zeroconf.Event) {
		// Only react to "Added" events with valid IPs
		if e.Op == zeroconf.OpAdded && len(e.Addrs) > 0 {
			log.Printf("👀 Found new node: %s [%v]. Auto-adopting...", e.Name, e.Addrs)

			// Extract IP (prefer IPv4)
			ip := e.Addrs[0].String()
			// Note: You might want to iterate e.Addrs to find the IPv4 one specifically
			// if the network has both.

			go adoptNode(ip, "agent")
		}
	}

	// 3. Start the Engine (Publish Myself + Browse for Others)
	client, err := zeroconf.New().
		Publish(me).                                // "I am the Controller"
		Browse(onNodeFound, discovery.TypePending). // "Look for Pending Nodes"
		Open()

	if err != nil {
		log.Fatalf("Failed to start zeroconf: %v", err)
	}
	defer client.Close()

	log.Println("✅ Controller Beacon Active & Scanning...")

	// 4. Block forever (or until signal)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down controller...")
}

// ---------------------------------------------------------
// Mode: PENDING (The "Unboxed" State)
// ---------------------------------------------------------
func runPendingMode() {
	log.Println("State: PENDING. Broadcasting availability...")

	// 1. Advertise ourselves
	client, err := discovery.StartAgentBroadcast(NodeID, Port)
	if err != nil {
		log.Fatalf("Failed to start broadcast: %v", err)
	}
	defer client.Close()

	// 2. Wait indefinitely
	select {}
}

// ... [runComputeMode and adoptNode remain the same] ...
