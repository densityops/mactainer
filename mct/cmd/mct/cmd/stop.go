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

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var removeMachine bool

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "Stopping machine\n")
		logrus.SetLevel(logrus.DebugLevel)
		libMachineAPIClient, cleanup := createLibMachineClient()
		defer cleanup()
		exists, err := libMachineAPIClient.Exists(machineName)
		if err != nil {
			return fmt.Errorf("could find the machine")
		}
		if exists {
			host, err := libMachineAPIClient.Load(machineName)
			if err != nil {
				return fmt.Errorf("error loading host: %s", err)
			}
			state, err := host.Driver.GetState()
			if err != nil {
				return fmt.Errorf("could not get state from the machine: %s", err)
			}
			if state.String() != "Running" {
				cobra.CheckErr(fmt.Errorf("cannot stop a machine in state: %s", state.String()))
			}
			if err := host.Stop(); err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %s\n", err)
				fmt.Fprintf(cmd.OutOrStdout(), "Stopping machine failed, killing it\n")
				if err = host.Kill(); err != nil {
					cobra.CheckErr(err)
				}
			}
			// stop daemon process

			// remove machine
			if removeMachine {
				if err := waitForMachineState(host, stateStopped, 15); err != nil {
					cobra.CheckErr(err)
				}
				if err := libMachineAPIClient.Remove(machineName); err != nil {
					cobra.CheckErr(err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "removed machine")
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stopCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	stopCmd.Flags().BoolVarP(&removeMachine, "remove", "r", false, "Remove the machine")
}
