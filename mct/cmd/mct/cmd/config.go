package cmd

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	sshPort     = 2022
	tapSocket   = "tap.sock"
	httpSocket  = "http.sock"
	bundlesDir  = filepath.Join(defaultMCTHome(), "bundles")
	machineDir  = filepath.Join(defaultMCTHome(), "machine")
	binDir      = filepath.Join(defaultMCTHome(), "bin")
	machineName = "mct"
	mctHome     = defaultMCTHome()
)

func defaultMCTHome() string {
	home, err := homedir.Dir()
	cobra.CheckErr(err)
	return filepath.Join(home, ".mct")
}

func createDefaultConfig() {
	viper.SetDefault("home", defaultMCTHome())
	viper.SetDefault("bundles.dir", bundlesDir)
	viper.SetDefault("network.ssh.port", sshPort)
	viper.SetDefault("bin", binDir)
	viper.SetDefault("machine.dir", machineDir)
	viper.SetDefault("machine.name", machineName)
	err := os.MkdirAll(defaultMCTHome(), os.ModePerm)
	cobra.CheckErr(err)
	if err := viper.SafeWriteConfig(); err != nil {
		cobra.CheckErr(err)
	}
}
