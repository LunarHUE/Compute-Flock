package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/lunarhue/libs-go/log"

	"github.com/lunarhue/compute-flock/pkg/discovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

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
	log.Infof("Received ADOPT command. Role: %s, Controller: %s", req.Role, req.ControllerIp)

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
		log.Panicf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFlockServiceServer(s, &server{})
	go s.Serve(lis)

	// STATE MACHINE
	switch *mode {
	case "controller":
		discovery.RunControllerMode(NodeID, uint16(Port), func(ip, role string) { adoptNode(currentLocalIP(), ip, role) })
	case "compute":
		discovery.RunComputeMode()
	case "auto":
		discovery.RunPendingMode(NodeID, uint16(Port))
	}
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
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", computeIp, Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("Failed to connect to %s: %v", computeIp, err)
		return
	}

	defer conn.Close()

	client := pb.NewFlockServiceClient(conn)
	_, err = client.Adopt(context.Background(), &pb.AdoptRequest{
		ClusterToken: "my-secret-token",
		ControllerIp: controllerIp,
		Role:         role,
	})

	if err != nil {
		log.Errorf("Failed to adopt %s: %v", computeIp, err)
	} else {
		log.Infof("Successfully sent adoption command to %s", computeIp)
	}
}
