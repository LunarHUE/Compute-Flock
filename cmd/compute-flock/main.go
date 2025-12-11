package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/lunarhue/libs-go/log"

	"github.com/lunarhue/compute-flock/pkg/discovery"
	"github.com/lunarhue/compute-flock/pkg/k3s"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/lunarhue/compute-flock/pkg/proto/adoption/v1"
)

type FlockMode string

const (
	ModePending    FlockMode = "PENDING"
	ModeController FlockMode = "CONTROLLER"
	ModeAgent      FlockMode = "AGENT"
)

// Global State
var (
	NodeID       string
	CurrentState FlockMode = ModePending
	DefaultPort            = 9000
)

type server struct {
	pb.UnimplementedFlockServiceServer
}

func (s *server) Adopt(ctx context.Context, req *pb.AdoptRequest) (*pb.AdoptResponse, error) {
	log.Infof("Received ADOPT command. Role: %s, Controller: %s", req.Role, req.ControllerIp)
	log.Infof("Cluster Token: %s", req.ClusterToken)

	k3s.StartAgent(fmt.Sprintf("https://%s:6443", req.ControllerIp), req.ClusterToken)

	return &pb.AdoptResponse{Success: true, Message: "Adoption started"}, nil
}

func (s *server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	return &pb.HeartbeatResponse{Reconfigure: false}, nil
}

func main() {
	mode := flag.String("mode", "auto", "Force mode: server, agent, auto")
	noVerify := flag.Bool("no-verify", false, "Skip K3s installation verification")
	flag.Parse()

	hostname, _ := os.Hostname()
	NodeID = hostname

	// Verify that the prerequisites are met
	if !*noVerify {
		if err := k3s.VerifyK3sInstallation(*mode); err != nil {
			log.Panicf("K3s verification failed: %v", err)
		}
	}

	// Start GRPC Server (Listens on all modes)
	apiPort := findOpenPort(DefaultPort)
	if apiPort == 0 {
		log.Panic("No open port found")
	} else if apiPort != DefaultPort {
		log.Warnf("Default port %d is in use. Using port %d instead.", DefaultPort, apiPort)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", apiPort))
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFlockServiceServer(s, &server{})
	go s.Serve(lis)

	// STATE MACHINE
	switch *mode {
	case "server":
		discovery.RunControllerMode(NodeID, uint16(DefaultPort), func(ip, role string) { adoptNode(currentLocalIP(), ip, role) })
	case "agent":
		discovery.RunPendingMode(NodeID, uint16(DefaultPort))
	}
}

func findOpenPort(defaultPort int) int {
	const maxPort = 32767 // Max positive value for int16

	for port := defaultPort; port <= maxPort; port++ {
		addr := net.JoinHostPort("", strconv.Itoa(port))
		listener, err := net.Listen("tcp", addr)

		// If err is nil, the port is available
		if err == nil {
			listener.Close()
			return port
		}
	}

	return 0
}

func currentLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String()
}

func adoptNode(controllerIp, computeIp string, role string) {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", computeIp, DefaultPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("Failed to connect to %s: %v", computeIp, err)
		return
	}

	defer conn.Close()

	adoptionToken, err := k3s.CreateJoinToken("compute-flock-node", 1*time.Minute)
	if err != nil {
		log.Errorf("Failed to create join token: %v", err)
		return
	}

	log.Infof("Generated join token for %s: %s", computeIp, adoptionToken)

	client := pb.NewFlockServiceClient(conn)
	_, err = client.Adopt(context.Background(), &pb.AdoptRequest{
		ClusterToken: adoptionToken,
		ControllerIp: controllerIp,
		Role:         role,
	})

	if err != nil {
		log.Errorf("Failed to adopt %s: %v", computeIp, err)
	} else {
		log.Infof("Successfully sent adoption command to %s", computeIp)
	}
}
