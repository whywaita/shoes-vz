package main

import (
	"context"
	"log"

	"github.com/hashicorp/go-plugin"
	myshoespb "github.com/whywaita/myshoes/api/proto.go"
	"github.com/whywaita/shoes-vz/internal/client"
	"google.golang.org/grpc"
)

// Handshake is a common handshake that is shared between the plugin and host.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "SHOES_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "are_you_a_shoes?",
}

// ShoesGRPCPlugin is the implementation of plugin.GRPCPlugin
type ShoesGRPCPlugin struct {
	plugin.Plugin
	Impl myshoespb.ShoesServer
}

// GRPCServer registers the gRPC server
func (p *ShoesGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	myshoespb.RegisterShoesServer(s, p.Impl)
	return nil
}

// GRPCClient returns the gRPC client
func (p *ShoesGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return myshoespb.NewShoesClient(c), nil
}

func main() {
	// Load configuration from environment variables
	config, err := client.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create client
	shoesClient, err := client.NewClient(config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if err := shoesClient.Close(); err != nil {
			log.Printf("Failed to close client: %v", err)
		}
	}()

	// Serve the plugin
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			"shoes_grpc": &ShoesGRPCPlugin{Impl: shoesClient},
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
