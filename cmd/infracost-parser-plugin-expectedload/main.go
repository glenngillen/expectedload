// infracost-parser-plugin-expectedload is an Infracost parser plugin that
// extracts `expected-load` declarations from source-code comments (Terraform,
// TypeScript/JavaScript, Python, Go, Java/Kotlin, Rust) using the
// github.com/glenngillen/expectedload library.
//
// The Infracost CLI spawns this binary and talks to it over gRPC via the
// HashiCorp go-plugin framework; it is not meant to be run by hand. Running it
// with --version prints the build version and exits.
package main

import (
	"context"
	"fmt"
	"os"

	goplugin "github.com/hashicorp/go-plugin"
	pluginpb "github.com/infracost/proto/gen/go/infracost/plugin"
	"google.golang.org/grpc"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

const maxMessageSize = 64 * 1024 * 1024

var (
	_ goplugin.Plugin     = (*parserPlugin)(nil)
	_ goplugin.GRPCPlugin = (*parserPlugin)(nil)
)

// parserPlugin wires the ParserService implementation into go-plugin.
type parserPlugin struct {
	goplugin.NetRPCUnsupportedPlugin
}

func (p *parserPlugin) GRPCServer(_ *goplugin.GRPCBroker, g *grpc.Server) error {
	s := &service{}
	pluginpb.RegisterPluginServiceServer(g, s)
	pluginpb.RegisterParserServiceServer(g, s)
	return nil
}

func (p *parserPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, _ *grpc.ClientConn) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-version" || arg == "version" {
			fmt.Printf("infracost-parser-plugin-expectedload %s\n", version)
			return
		}
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: goplugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "INFRACOST_PLUGIN",
			MagicCookieValue: "de8c7e96-497c-4168-80c4-fc875c8ce764",
		},
		Plugins: map[string]goplugin.Plugin{
			"plugin": new(parserPlugin),
		},
		GRPCServer: func(opts []grpc.ServerOption) *grpc.Server {
			opts = append(opts,
				grpc.MaxRecvMsgSize(maxMessageSize),
				grpc.MaxSendMsgSize(maxMessageSize),
			)
			return grpc.NewServer(opts...)
		},
	})
}
