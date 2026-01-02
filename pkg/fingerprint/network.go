package fingerprint

import (
	"fmt"

	"github.com/jaypipes/ghw"
	"github.com/lunarhue/libs-go/log"
)

type NetworkInterfaceInfo struct {
	InterfaceName string `json:"interface_name"`
	MacAddress    string `json:"mac_address"`
	Vendor        string `json:"vendor"`
}

func GetNetworkInterfaces() ([]NetworkInterfaceInfo, error) {
	// We use ghw for hardware details (Vendor/Model)
	netInfo, err := ghw.Network()
	if err != nil {
		return nil, fmt.Errorf("error getting network info: %v", err)
	}

	var interfaces []NetworkInterfaceInfo

	for _, nic := range netInfo.NICs {
		if nic.IsVirtual {
			continue
		}

		vendor, err := GetVendor(nic.MACAddress)
		if err != nil {
			vendor = "Unknown"
			log.Warnf("Unable to get Vendor for MAC Address: %s, %v", nic.MACAddress, err)
		}

		interfaces = append(interfaces, NetworkInterfaceInfo{
			InterfaceName: nic.Name,
			MacAddress:    nic.MACAddress,
			Vendor:        vendor,
		})
	}

	return interfaces, nil
}
