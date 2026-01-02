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

var noVerify bool = false

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Runs the controller.",
	Run: func(cmd *cobra.Command, args []string) {
		hostname, _ := os.Hostname()
		NodeID = hostname

		// Verify that the prerequisites are met
		if !noVerify {
			if err := k3s.VerifyK3sInstallation("server"); err != nil {
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

		discovery.RunControllerMode(NodeID, uint16(DefaultPort), func(ip, role string) { adoptNode(currentLocalIP(), ip, role) })
	},
}

func init() {
	controllerCmd.PersistentFlags().BoolVar(&noVerify, "no-verify", false, "Skip K3s installation verification")
	rootCmd.AddCommand(controllerCmd)
}
