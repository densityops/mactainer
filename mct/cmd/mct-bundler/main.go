package main

import (
	"fmt"
	"os"

	"github.com/densityops/mactainer/mct/pkg/bundle"
	"github.com/densityops/mactainer/mct/pkg/bundler"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

// Here is a real implementation of Unbudler
type MCTUnbundler struct {
	logger hclog.Logger
}

func (g *MCTUnbundler) Unbundle(path string) string {
	fmt.Printf("Unbundling mct-bundle-%s\n", bundle.Version)
	err := bundle.UnbundleBin(path)
	if err != nil {
		return fmt.Sprintf("error unbundling binaries: %s", err.Error())
	}
	err = bundle.UnbundleMachine(path)
	if err != nil {
		return fmt.Sprintf("error unbundling machine: %s", err.Error())
	}
	return fmt.Sprintf("installed at: %s/bundles/%s", path, bundle.Version)
}

// handshakeConfigs are used to just do a basic handshake between
// a plugin and host. If the handshake fails, a user friendly error is shown.
// This prevents users from executing bad plugins or executing a plugin
// directory. It is a UX feature, not a security feature.
var handshakeConfig = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_PLUGIN",
	MagicCookieValue: "mct-unbundler",
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Level:      hclog.Trace,
		Output:     os.Stderr,
		JSONFormat: true,
	})

	unbundler := &MCTUnbundler{
		logger: logger,
	}
	// pluginMap is the map of plugins we can dispense.
	var pluginMap = map[string]plugin.Plugin{
		"unbundler": &bundler.UnbundlerPlugin{Impl: unbundler},
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: handshakeConfig,
		Plugins:         pluginMap,
	})
}
