package fingerprint

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed mac-vendors-export.json
var MacVendorsJson string

// Internal cache to store the parsed vendor list.
// We use a map for O(1) fast lookups instead of iterating the slice.
var (
	vendorCache map[string]string
	loadOnce    sync.Once
	loadErr     error
)

// macVendorEntry matches the structure of the objects inside your JSON file.
type macVendorEntry struct {
	MacPrefix  string `json:"macPrefix"`
	VendorName string `json:"vendorName"`
}

// GetVendor returns the vendor name for a given MAC address.
// It parses the embedded JSON database only once (lazily).
func GetVendor(macAddress string) (string, error) {
	// 1. Lazy load the database on the first call only
	loadOnce.Do(func() {
		vendorCache = make(map[string]string)
		var vendors []macVendorEntry

		// Unmarshal the embedded JSON string
		if err := json.Unmarshal([]byte(MacVendorsJson), &vendors); err != nil {
			loadErr = fmt.Errorf("failed to parse embedded vendor database: %w", err)
			return
		}

		// Populate the map for fast lookup
		for _, v := range vendors {
			vendorCache[v.MacPrefix] = v.VendorName
		}
	})

	// Check if loading failed
	if loadErr != nil {
		return "", loadErr
	}

	// 2. Normalize the input MAC address
	// Convert to Uppercase and replace dashes/dots with colons to match JSON format
	// Example: "00-00-5e-..." -> "00:00:5E:..."
	r := strings.NewReplacer("-", ":", ".", ":")
	normalizedMac := r.Replace(strings.ToUpper(macAddress))

	//

	// 3. Extract the OUI (First 3 bytes / 8 characters)
	// The standard OUI format in your JSON is "XX:XX:XX" (8 chars)
	if len(normalizedMac) < 8 {
		return "", fmt.Errorf("invalid MAC address format: too short")
	}

	prefix := normalizedMac[0:8]

	// 4. Lookup in the map
	if name, found := vendorCache[prefix]; found {
		return name, nil
	}

	return "", fmt.Errorf("vendor not found for prefix %s", prefix)
}
