package discovery

import (
	"fmt"
	"log"
	"time"

	"github.com/lunarhue/libs-go/metadata"
	zeroconf "github.com/lunarhue/metallic-flock-zeroconf"
)

// Service Types
var (
	TypePending    = zeroconf.NewType("_hive-pending._tcp")
	TypeController = zeroconf.NewType("_hive-controller._tcp")
)

// StartAgentBroadcast announces "I am here" (Pending) AND listens for Controllers.
// Returns the client so you can Close() it later.
func StartAgentBroadcast(id string, port uint16) (*zeroconf.Client, error) {
	me := zeroconf.NewService(TypePending, id, port)
	sysinfo, err := metadata.GetSystemInfo()
	if err != nil {
		me.Text = []string{"version=1.0", "error=" + err.Error()}
		log.Printf("Failed to get system info: %v", err)
	} else {
		// Add useful metadata
		me.Text = []string{
			"version=1.0",
			"cpu=" + fmt.Sprintf("%d", sysinfo.CPUCores),
			"distro=" + sysinfo.Arch,
			"ip=" + sysinfo.MainIP,
			"mac=" + sysinfo.MainMAC,
			"os=" + sysinfo.OS,
			"disk=" + fmt.Sprintf("%f", sysinfo.TotalDiskGB),
			"mem=" + fmt.Sprintf("%f", sysinfo.TotalMemoryGB),
		}
	}

	client, err := zeroconf.New().
		Publish(me).
		Open()

	if err != nil {
		return nil, err
	}

	return client, nil
}

// ScanForControllers looks for a controller for a specific duration.
// It opens a temporary browser and closes it after the timeout.
func ScanForControllers(duration time.Duration) string {
	foundIP := make(chan string, 1)

	// Define the browse callback
	onEvent := func(e zeroconf.Event) {
		// e.Op represents the operation (Added, Removed, etc)
		// We only care if a service is Added (OpAdded is usually 0 or 1, assuming non-removal)
		// The README example checks e.Op, but for simple discovery we just check if we have IPs.

		if e.Op == zeroconf.OpAdded && len(e.Addrs) > 0 {
			// We found a candidate
			log.Printf("Discovered Controller: %s", e.Name)
			select {
			case foundIP <- e.Addrs[0].String():
			default:
			}
		}
	}

	// Start the browser
	client, err := zeroconf.New().
		Browse(onEvent, TypeController).
		Open()

	if err != nil {
		log.Printf("Failed to start scanner: %v", err)
		return ""
	}
	defer client.Close()

	// Wait for result or timeout
	select {
	case ip := <-foundIP:
		return ip
	case <-time.After(duration):
		return ""
	}
}
