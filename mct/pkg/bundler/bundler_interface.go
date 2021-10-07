package bundler

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

// Greeter is the interface that we're exposing as a plugin.
type Unbundler interface {
	Unbundle(path string) string
}

// Here is an implementation that talks over RPC
type UnbundlerRPC struct{ client *rpc.Client }

func (g *UnbundlerRPC) Unbundle(path string) string {
	var resp string
	err := g.client.Call("Plugin.Unbundle", path, &resp)
	if err != nil {
		panic(err)
	}
	return resp
}

// Here is the RPC server that GreeterRPC talks to, conforming to
// the requirements of net/rpc
type UnbundlerRPCServer struct {
	// This is the real implementation
	Impl Unbundler
}

func (s *UnbundlerRPCServer) Unbundle(path string, resp *string) error {
	*resp = s.Impl.Unbundle(path)
	return nil
}

// This is the implementation of plugin.Plugin so we can serve/consume this
//
// This has two methods: Server must return an RPC server for this plugin
// type. We construct a GreeterRPCServer for this.
//
// Client must return an implementation of our interface that communicates
// over an RPC client. We return GreeterRPC for this.
//
// Ignore MuxBroker. That is used to create more multiplexed streams on our
// plugin connection and is a more advanced use case.
type UnbundlerPlugin struct {
	// Impl Injection
	Impl Unbundler
}

func (p *UnbundlerPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &UnbundlerRPCServer{Impl: p.Impl}, nil
}

func (UnbundlerPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &UnbundlerRPC{client: c}, nil
}
