package debug

import (
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "debug",
	Short: "Commands to use for debugging purposes.",
}
