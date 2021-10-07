/*
Copyright 2019 Red Hat, Inc.
Copyright 2021 DensityOps All rights reserved.
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
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorilla/handlers"
	"github.com/sevlyar/go-daemon"

	"github.com/containers/gvisor-tap-vsock/pkg/types"
	"github.com/containers/gvisor-tap-vsock/pkg/virtualnetwork"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// daemonCmd represents the daemon command
var daemonCmd = &cobra.Command{
	Use:   "daemon <staert|stop>",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	ValidArgs: []string{"start", "stop"},
	Args:      cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		action := args[0]
		os.MkdirAll(filepath.Join(machineDir, machineName), os.ModePerm)
		cntxt := &daemon.Context{
			PidFileName: filepath.Join(machineDir, machineName, "daemon.pid"),
			PidFilePerm: 0644,
			LogFileName: filepath.Join(machineDir, machineName, "daemon.log"),
			LogFilePerm: 0640,
			WorkDir:     filepath.Join(machineDir, machineName),
			Umask:       027,
			Args:        nil,
		}
		switch action {
		case "start":
			fmt.Fprintln(cmd.OutOrStdout(), "Starting daemon")
			config := newVirtualNetworkConfig()
			d, err := cntxt.Reborn()
			if err != nil {
				return fmt.Errorf("unable to start daemon: %s", err)
			}
			if d != nil {
				return nil
			}
			defer cntxt.Release()

			return runDaemon(cmd.Context(), config)
		case "stop":
			fmt.Fprintln(cmd.OutOrStdout(), "Stopping daemon")
			d, err := cntxt.Search()
			if err != nil {
				return err
			}
			d.Kill()
			return nil
		default:
			return fmt.Errorf("%s is not a dameon action", action)

		}
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}

func newVirtualNetworkConfig() *types.Configuration {
	return &types.Configuration{
		Debug:             false, // never log packets
		CaptureFile:       os.Getenv("MCT_DAEMON_PCAP_FILE"),
		MTU:               1500, // Large packets slightly improve the performance. Less small packets.
		Subnet:            "192.168.127.0/24",
		GatewayIP:         "192.168.127.1",
		GatewayMacAddress: "5a:94:ef:e4:0c:dd",
		DHCPStaticLeases: map[string]string{
			"192.168.127.2": "5a:94:ef:e4:0c:ee",
		},
		DNS: []types.Zone{
			{
				Name:      "mactainer.containers.",
				DefaultIP: net.ParseIP("192.168.127.2"),
			},
			{
				Name: "mct.mactainer.",
				Records: []types.Record{
					{
						Name: "gateway",
						IP:   net.ParseIP("192.168.127.1"),
					},
					{
						Name: "api",
						IP:   net.ParseIP("192.168.127.2"),
					},
					{
						Name: "api-int",
						IP:   net.ParseIP("192.168.127.2"),
					},
					{
						Name: "host",
						IP:   net.ParseIP("192.168.127.254"),
					},
				},
			},
		},
		Forwards: map[string]string{
			fmt.Sprintf(":%d", sshPort): "192.168.127.2:22",
		},
		NAT: map[string]string{
			"192.168.127.254": "127.0.0.1",
		},
		GatewayVirtualIPs: []string{"192.168.127.254"},
		VpnKitUUIDMacAddresses: map[string]string{
			"c3d68012-0208-11ea-9fd7-f2189899ab08": "5a:94:ef:e4:0c:ee",
		},
		Protocol: types.HyperKitProtocol,
	}
}

func runDaemon(ctx context.Context, config *types.Configuration) error {
	vsockListener, err := vsockListener()
	if err != nil {
		return err
	}
	virtualNetwork, err := virtualnetwork.New(config)
	if err != nil {
		return err
	}

	listener, err := httpListener()
	if err != nil {
		return err
	}

	errCh := make(chan error)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/network/", http.StripPrefix("/network", virtualNetwork.Mux()))
		if err := http.Serve(listener, handlers.LoggingHandler(os.Stderr, mux)); err != nil {
			errCh <- errors.Wrap(err, "api http.Serve failed")
		}
	}()

	ln, err := virtualNetwork.Listen("tcp", fmt.Sprintf("%s:8080", "192.168.127.1"))
	if err != nil {
		return fmt.Errorf("could not bind tcp/8080 on 192.168.127.1: %s", err)
	}

	go func() {
		mux := ignitionMux()
		if err := http.Serve(ln, handlers.LoggingHandler(os.Stderr, mux)); err != nil {
			errCh <- errors.Wrap(err, "gateway http.Serve failed")
		}
	}()

	go func() {
		for {
			conn, err := vsockListener.Accept()
			if err != nil {
				fmt.Printf("vpnkit accept error: %s", err)
				continue
			}
			if err := virtualNetwork.AcceptVpnKit(conn); err != nil {
				fmt.Printf("vpnkit accept error: %s", err)
			}
		}
	}()

	startupDone()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	select {
	case <-c:
		return nil
	case err := <-errCh:
		return err
	}
}

func vsockListener() (net.Listener, error) {
	_ = os.Remove("tap.sock")
	ln, err := net.Listen("unix", tapSocketPath())
	if err != nil {
		return nil, fmt.Errorf("could not bind vsock: %s", err)
	}
	fmt.Printf("listening %s\n", tapSocketPath())
	return ln, nil
}

func httpListener() (net.Listener, error) {
	_ = os.Remove(httpSocketPath())
	ln, err := net.Listen("unix", httpSocketPath())
	if err != nil {
		return nil, fmt.Errorf("could not bind http socket: %s", err)
	}
	fmt.Printf("listening %s\n", httpSocketPath())

	return ln, nil
}

// This API is only exposed in the virtual network (only the VM can reach this).
// Any process inside the VM can reach it by connecting to curl gateway.mct.mactainer:8080/ignition
func ignitionMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/ignition", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		http.ServeFile(w, r, "/Users/mabe01/code/src/github.com/densityops/mactainer/_build/instances/mactainer/mactainer.ign")
	})
	return mux
}

func startupDone() {
}

func tapSocketPath() string {
	return filepath.Join(machineDir, machineName, tapSocket)
}

func httpSocketPath() string {
	return filepath.Join(machineDir, machineName, httpSocket)

}
