package config

type MachineConfig struct {
	// Virtual machine configuration
	Name            string
	Memory          int
	CPUs            int
	DiskSize        int
	ImageSourcePath string
	ImageFormat     string
	SSHKeyPath      string

	// HyperKit specific configuration
	KernelCmdLine string
	Initramfs     string
	Kernel        string

	// Experimental features
	NetworkMode NetworkMode
}

type NetworkMode string

const (
	SystemNetworkingMode NetworkMode = "system"
	UserNetworkingMode   NetworkMode = "user"
)
