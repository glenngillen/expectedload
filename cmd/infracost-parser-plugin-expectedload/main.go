// infracost-parser-plugin-expectedload is an Infracost parser plugin that
// extracts `expected-load` declarations from source-code comments (Terraform,
// TypeScript/JavaScript, Python, Go, Java/Kotlin, Rust) using the
// github.com/infracost/expectedload library.
//
// The Infracost CLI spawns this binary and talks to it over gRPC via the
// HashiCorp go-plugin framework; it is not meant to be run by hand. Running it
// with --version prints the build version and exits.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/go-plugin"
	"github.com/infracost/proto/gen/go/infracost/parser/api"
	"google.golang.org/grpc"
)

// version is injected at build time via -ldflags "-X main.version=...".
var version = "dev"

const maxMessageSize = 64 * 1024 * 1024

var (
	_ plugin.Plugin     = (*parserPlugin)(nil)
	_ plugin.GRPCPlugin = (*parserPlugin)(nil)
)

// parserPlugin wires the ParserService implementation into go-plugin.
type parserPlugin struct {
	plugin.NetRPCUnsupportedPlugin
}

func (p *parserPlugin) GRPCServer(_ *plugin.GRPCBroker, g *grpc.Server) error {
	api.RegisterParserServiceServer(g, &service{})
	return nil
}

func (p *parserPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, _ *grpc.ClientConn) (any, error) {
	return nil, fmt.Errorf("not implemented")
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-version" || arg == "version" {
			fmt.Printf("infracost-parser-plugin-expectedload %s\n", version)
			return
		}
	}

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   "INFRACOST_PARSER_PLUGIN_MAGIC_COOKIE",
			MagicCookieValue: "ac92b06c592f",
		},
		Plugins: map[string]plugin.Plugin{
			"parser": new(parserPlugin),
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
