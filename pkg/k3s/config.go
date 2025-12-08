package k3s

type K3sConfig struct {
	Token  string `yaml:"token,omitempty"`
	Server string `yaml:"server,omitempty"` // URL of the controller to join
	// Add other flags as needed
	TLSSan []string `yaml:"tls-san,omitempty"`
}
