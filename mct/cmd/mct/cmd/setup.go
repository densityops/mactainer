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
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/densityops/mactainer/mct/pkg/bundler"
	"github.com/google/go-github/v39/github"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/mitchellh/go-homedir"
	progressbar "github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := hclog.New(&hclog.LoggerOptions{
			Name:   "plugin",
			Output: os.Stdout,
			Level:  hclog.Info,
		})

		bundle, err := getLatestBundle(cmd.Context())
		cobra.CheckErr(err)
		bundleTarget := fmt.Sprintf("%s/bundles/%s", mctHome, bundle.GetName())
		if !bundleExists(bundleTarget) {
			fmt.Fprintf(cmd.OutOrStdout(), "Downloading bundle %s (%s)\n", bundle.GetName(), bundle.GetBrowserDownloadURL())
			downloadBundle(bundle.GetBrowserDownloadURL(), bundleTarget)
		}

		// We're a host! Start by launching the plugin process.
		client := plugin.NewClient(&plugin.ClientConfig{
			HandshakeConfig: handshakeConfig,
			Plugins:         pluginMap,
			Cmd:             exec.Command(fmt.Sprintf("%s/bundles/mct-bundle-v1.0.0", mctHome)),
			Logger:          logger,
			SyncStdout:      cmd.OutOrStdout(),
		})
		defer client.Kill()

		// Connect via RPC
		rpcClient, err := client.Client()
		if err != nil {
			cobra.CheckErr(err)
		}

		// Request the plugin
		raw, err := rpcClient.Dispense("unbundler")
		if err != nil {
			cobra.CheckErr(err)
		}
		b := raw.(bundler.Unbundler)
		out := b.Unbundle(mctHome)
		fmt.Fprintf(cmd.OutOrStdout(), "Setup done: %s\n", out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// setupCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// setupCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	home, _ := homedir.Dir()
	setupCmd.Flags().StringVarP(&machineDir, "directory", "d", filepath.Join(home, ".mct", "machines"), "directory for machines")
	setupCmd.Flags().StringVarP(&machineName, "name", "n", "mct", "name of the machine")
}

var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "mct-unbundler",
}

// pluginMap is the map of plugins we can dispense.
var pluginMap = map[string]plugin.Plugin{
	"unbundler": &bundler.UnbundlerPlugin{},
}

func getLatestBundle(ctx context.Context) (*github.ReleaseAsset, error) {
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(ctx, "densityops", "mactainer")
	if err != nil {
		return nil, err
	}
	for _, asset := range release.Assets {
		if asset.GetLabel() == "bundle" {
			return asset, nil
		}
	}
	return nil, fmt.Errorf("unable to get bundle from latest release")
}

func downloadBundle(url string, dest string) error {
	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"Downloading",
	)
	_, err = io.Copy(io.MultiWriter(f, bar), resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func bundleExists(bundle string) bool {
	if _, err := os.Stat(bundle); os.IsNotExist(err) {
		return false
	}
	return true
}
