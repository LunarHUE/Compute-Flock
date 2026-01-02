package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/lunarhue/libs-go/log"

	"github.com/lunarhue/metallic-flock/cmd/debug"
	"github.com/lunarhue/metallic-flock/pkg/k3s"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/lunarhue/metallic-flock/pkg/proto/adoption/v1"
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

var rootCmd = &cobra.Command{
	Use:   "metallic",
	Short: "Used to create base k3s cluster.",
	Long:  `Sets up the controller and agent relationship and does cluster authentication.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(debug.RootCmd)
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

	adoptionToken, err := k3s.CreateJoinToken("metallic-flock-node", 1*time.Minute)
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
