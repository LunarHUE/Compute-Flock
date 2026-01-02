package cmd

import (
	"fmt"
	"net"
	"os"

	"github.com/lunarhue/libs-go/log"
	"github.com/lunarhue/metallic-flock/pkg/discovery"
	"github.com/lunarhue/metallic-flock/pkg/k3s"
	pb "github.com/lunarhue/metallic-flock/pkg/proto/adoption/v1"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Runs the agent.",
	Run: func(cmd *cobra.Command, args []string) {
		hostname, _ := os.Hostname()
		NodeID = hostname

		// Verify that the prerequisites are met
		if !noVerify {
			if err := k3s.VerifyK3sInstallation("agent"); err != nil {
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

		discovery.RunPendingMode(NodeID, uint16(DefaultPort))
	},
}

func init() {
	agentCmd.PersistentFlags().BoolVar(&noVerify, "no-verify", false, "Skip K3s installation verification")
	rootCmd.AddCommand(agentCmd)
}
