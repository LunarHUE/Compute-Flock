package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lunarhue/compute-flock/pkg/discovery"
	"github.com/lunarhue/compute-flock/pkg/system"
	"google.golang.org/grpc"

	zeroconf "github.com/lunarhue/compute-flock-zeroconf"
	pb "github.com/lunarhue/compute-flock/pkg/proto/adoption/v1"
)

type FlockMode string

const (
	ModePending    FlockMode = "PENDING"
	ModeController FlockMode = "CONTROLLER"
	ModeCompute    FlockMode = "COMPUTE"
)

// Global State
var (
	NodeID       string
	CurrentState FlockMode = ModePending
	Port                   = 9000
)

type server struct {
	pb.UnimplementedFlockServiceServer
}

func (s *server) Adopt(ctx context.Context, req *pb.AdoptRequest) (*pb.AdoptResponse, error) {
	log.Printf("Received ADOPT command. Role: %s, Controller: %s", req.Role, req.ControllerIp)

	conf := system.K3sConfig{
		Role:         req.Role,
		Token:        req.ClusterToken,
		ControllerIP: req.ControllerIp,
		ClusterInit:  (req.Role == "server" && req.ControllerIp == ""),
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

func main() {
	mode := flag.String("mode", "auto", "Force mode: controller, compute, auto")
	flag.Parse()

	hostname, _ := os.Hostname()
	NodeID = hostname

	// Start GRPC Server (Listens on all modes)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", Port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFlockServiceServer(s, &server{})
	go s.Serve(lis)

	// STATE MACHINE
	switch *mode {
	case "controller":
		runControllerMode()
	case "compute":
		runComputeMode()
	case "auto":
		runPendingMode()
	}
}

// ---------------------------------------------------------
// Mode: COMPUTE (The "Worker" State)
// ---------------------------------------------------------
func runComputeMode() {
	log.Println("State: COMPUTE. Connecting to Cluster...")

	// Survivability Loop
	for {
		// Try to find controller
		controllerIP := discovery.ScanForControllers(5 * time.Second)

		if controllerIP == "" {
			log.Println("⚠️ Lost Controller! Scanning...")
			// Logic: If lost for too long, maybe revert to PENDING?
			// For now, just keep looking.
		} else {
			log.Printf("✅ Connected to Controller at %s", controllerIP)
			// Send Heartbeat via gRPC here
		}

		time.Sleep(10 * time.Second)
	}
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

func adoptNode(ip string, role string) {
	conn, _ := grpc.Dial(fmt.Sprintf("%s:%d", ip, Port), grpc.WithInsecure())
	defer conn.Close()

	client := pb.NewFlockServiceClient(conn)
	_, err := client.Adopt(context.Background(), &pb.AdoptRequest{
		ClusterToken: "my-secret-token",
		ControllerIp: "192.168.1.10", // Self IP
		Role:         role,
	})

	if err != nil {
		log.Printf("Failed to adopt %s: %v", ip, err)
	} else {
		log.Printf("Successfully sent adoption command to %s", ip)
	}
}
