/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/densityops/machine/drivers/hyperkit"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type StartConfig struct {

	// Hypervisor
	Memory   int // Memory size in MiB
	CPUs     int
	DiskSize int // Disk size in GiB

	// Nameserver
	NameServer string
}

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "Starting machine\n")
		if err := os.MkdirAll(filepath.Join(mctHome, "machines", "mct"), os.ModePerm); err != nil {
			cobra.CheckErr(err)
		}
		logrus.SetLevel(logrus.DebugLevel)
		driver := hyperkit.NewDriver("mct", mctHome)
		driver.HyperKitPath = filepath.Join(mctHome, "bin", "hyperkit")
		driver.QcowToolPath = filepath.Join(mctHome, "bin", "qcow-tool")
		driver.UUID = "c3d68012-0208-11ea-9fd7-f2189899ab08"
		// TODO: read from bundle
		driver.Cmdline = "BOOT_IMAGE=(hd0,gpt3)/ostree/fedora-coreos-5dce8a8faac406dc3baf6e1e6ece5946780cc3bd5de6fb78075526ec183a93f6/vmlinuz-5.13.16-200.fc34.x86_64 mitigations=auto,nosmt console=tty0 console=ttyS0,115200n8 ignition.platform.id=qemu ignition.config.url=http://192.168.127.1:8080/ignition ostree=/ostree/boot.1/fedora-coreos/5dce8a8faac406dc3baf6e1e6ece5946780cc3bd5de6fb78075526ec183a93f6/0 root=LABEL=root rw rootflags=prjquota"
		//driver.BootromPath = filepath.Join(mctHome, "bundles", "v1.0.0", "UEFI.fd")
		driver.VmlinuzPath = filepath.Join(mctHome, "bundles", "v1.0.0", "kernel")
		driver.InitrdPath = filepath.Join(mctHome, "bundles", "v1.0.0", "initrd.img")
		driver.DiskCapacity = 10737418240
		driver.ImageSourcePath = filepath.Join(mctHome, "bundles", "v1.0.0", "image.qcow2")
		driver.StorePath = filepath.Join(mctHome)
		driver.ImageFormat = "qcow2"
		driver.VMNet = false
		driver.VpnKitSock = tapSocketPath()
		driver.VpnKitUUID = "c3d68012-0208-11ea-9fd7-f2189899ab08"
		driver.VSockPorts = []string{"2376"}

		libMachineAPIClient, cleanup := createLibMachineClient()
		defer cleanup()
		driverJSON, _ := json.Marshal(driver)

		fmt.Fprintf(cmd.OutOrStdout(), "Creating host: %s\n", driver.MachineName)
		host, err := libMachineAPIClient.NewHost("hyperkit", binDir, driverJSON)
		if err != nil {
			return fmt.Errorf("could not create host: %s", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Checking if machine %s exists\n", host.Name)
		exists, err := libMachineAPIClient.Exists("mct")
		if err != nil {
			return fmt.Errorf("error checking if machine exists: %s", err)
		}

		if !exists {
			fmt.Fprintf(cmd.OutOrStdout(), "Creating machine: %s in %s\n", host.Name, libMachineAPIClient.MachinesDir)
			machineDir := filepath.Join(libMachineAPIClient.MachinesDir, "mct")
			os.MkdirAll(machineDir, os.ModePerm)
			// use ignition on first boot
			driver.Cmdline = "BOOT_IMAGE=(hd0,gpt3)/ostree/fedora-coreos-5dce8a8faac406dc3baf6e1e6ece5946780cc3bd5de6fb78075526ec183a93f6/vmlinuz-5.13.16-200.fc34.x86_64 mitigations=auto,nosmt console=tty0 console=ttyS0,115200n8 ignition.platform.id=qemu ignition.firstboot ignition.config.url=http://192.168.127.1:8080/ignition ostree=/ostree/boot.1/fedora-coreos/5dce8a8faac406dc3baf6e1e6ece5946780cc3bd5de6fb78075526ec183a93f6/0"
			driverJSON, _ := json.Marshal(driver)
			host.UpdateConfig(driverJSON)
			if err = host.Driver.Create(); err != nil {
				return fmt.Errorf("error creating host: %s", err)
			}
			if err = libMachineAPIClient.SetExists("mct"); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "could not set exists: %s\n", err)
			}
			if err = libMachineAPIClient.Save(host); err != nil {
				return fmt.Errorf("error saving host: %s", err)
			}
		} else {
			// save the host
			driverJSON, _ := json.Marshal(driver)
			host.UpdateConfig(driverJSON)
			if err = libMachineAPIClient.Save(host); err != nil {
				return fmt.Errorf("error saving host: %s", err)
			}
		}

		s, err := host.Driver.GetState()
		if err != nil {
			return fmt.Errorf("could not get state: %s", err)
		}

		if s.String() != "Running" {
			// https://stackoverflow.com/questions/39508086/golang-exec-background-process-and-get-its-pid Start and leave the daemon running
			// https://github.com/sevlyar/go-daemon/blob/master/examples/cmd/gd-simple/simple.go
			if err = host.Driver.Start(); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "could not start machine: %s\n", err)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Machine is already running: %s\n", host.Name)

		}

		fmt.Fprintf(cmd.OutOrStdout(), "Machine state: %s\n", s.String())

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// runCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// runCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
