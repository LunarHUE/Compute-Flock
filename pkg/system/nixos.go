package system

import (
	_ "embed" // Required for the go:embed directive
	"fmt"
	"os"
	"os/exec"
	"text/template" // Changed from html/template to prevent escaping special characters
)

//go:embed nixos.nix.tmpl
var NixosTemplate string

const k3sConfigPath = "/etc/nixos/imports/k3s-generated.nix"

type K3sConfig struct {
	Role         string // "server" or "agent"
	Token        string
	ControllerIP string
	ClusterInit  bool
}

func WriteAndRebuild(conf K3sConfig) error {
	// 1. Render the template using the embedded variable
	t, err := template.New("k3s").Parse(NixosTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(k3sConfigPath)
	if err != nil {
		return fmt.Errorf("failed to open nix config: %v", err)
	}
	defer f.Close()

	if err := t.Execute(f, conf); err != nil {
		return err
	}

	// 2. Execute NixOS Rebuild
	// Note: This requires the binary to run as root!
	cmd := exec.Command("nixos-rebuild", "switch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Applying NixOS configuration... this may take a while.")
	return cmd.Run()
}
