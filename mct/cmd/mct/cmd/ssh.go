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
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/spf13/cobra"
)

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "ssh to mct",
	Long:  `ssh`,
	Run: func(cmd *cobra.Command, args []string) {
		libMachineAPIClient, cleanup := createLibMachineClient()
		defer cleanup()

		host, err := libMachineAPIClient.Load("mct")
		if err != nil {
			cobra.CheckErr(err)
		}

		state, err := host.Driver.GetState()
		if err != nil {
			cobra.CheckErr(err)
		}
		if state.String() != stateRunning {
			cobra.CheckErr(fmt.Errorf("cannot ssh to a machine that is not running (%s)", state.String()))
		}

		privKey, err := os.CreateTemp("", "mct-ssh-*")
		if err != nil {
			cobra.CheckErr(err)
		}
		defer privKey.Close()
		defer os.Remove(privKey.Name())
		if _, err := privKey.Write([]byte(host.PrivateKey)); err != nil {
			cobra.CheckErr(err)
		}

		os.Chmod(privKey.Name(), 0400)
		sshDestination := "core@localhost"
		port := strconv.Itoa(2022)

		argsSSH := []string{"-i", privKey.Name(), "-p", port, sshDestination, "-o", "UserKnownHostsFile /dev/null", "-o", "StrictHostKeyChecking no"}

		fmt.Printf("Connecting to vm %s. To close connection, use `~.` or `exit`\n", "mct")
		cmdSSH := exec.Command("ssh", argsSSH...)

		cmdSSH.Stdout = os.Stdout
		cmdSSH.Stderr = os.Stderr
		cmdSSH.Stdin = os.Stdin
		cmdSSH.Run()
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// sshCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// sshCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
