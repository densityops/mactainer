package bundle

import "gopkg.in/yaml.v2"

var Version = "0.0.0-dev"

type BundleVersion struct {
	Bundle Bundle
}

type Bundle struct {
	Version     string
	CommandLine string   `yaml:"commandline"`
	HyperKit    HyperKit `yaml:"hyperkit,omitempty"`
}

type HyperKit struct {
	Image  string
	Kernel string
	Initrd string
}

func NewVersionFromBundle() (*BundleVersion, error) {
	fs := FS
	b, err := fs.ReadFile(Machine["machine.yaml"])
	if err != nil {
		return nil, err
	}
	v := &BundleVersion{}
	if err = yaml.Unmarshal(b, v); err != nil {
		return nil, err
	}
	v.Bundle.Version = Version
	return v, nil
}

func (b *BundleVersion) String() string {
	return b.Bundle.Version
}
