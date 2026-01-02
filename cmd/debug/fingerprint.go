package debug

import (
	"encoding/base64"
	"encoding/json"

	"github.com/lunarhue/libs-go/log"
	"github.com/lunarhue/metallic-flock/pkg/fingerprint"
	"github.com/spf13/cobra"
)

var fingerprintCmd = &cobra.Command{
	Use:   "fingerprint",
	Short: "Generates a fingerprint.",
	Long:  `Generates a fingerprint off of hardware specs and logs the JSON and Base64 representation.`,
	Run: func(cmd *cobra.Command, args []string) {
		fingerprint, err := fingerprint.GetFingerprint()
		if err != nil {
			log.Panicf("Fingerprint failed: %v", err)
		}

		jsonResult, err := json.Marshal(fingerprint)
		if err != nil {
			log.Panicf("Failed to marshal json: %v", err)
		}

		base64Result := base64.StdEncoding.EncodeToString(jsonResult)

		log.Infof("Fingerprint JSON: %s", jsonResult)
		log.Infof("Fingerprint Base64: %s", base64Result)
	},
}

func init() {
	RootCmd.AddCommand(fingerprintCmd)
}
