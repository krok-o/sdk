package main

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/krok-o/sdk/krok"
)

// Handshake is a common handshake that is shared by plugin and host.
var Handshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion: 1,
	MagicCookieKey:  "KROK_COMMANDS",
	// Never ever change this.
	MagicCookieValue: "26e39f04-4f5b-48e7-9c54-56b6e1f0c7cc",
}

// Command handles data passed down to this command from the hook server.
type Command interface {
	Execute(raw string) (string, bool, error)
}

// CommandGRPCPlugin is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type CommandGRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	plugin.Plugin
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
	Impl Command
}

// GRPCCommandClient is an implementation of Command that talks over RPC.
type GRPCCommandClient struct {
	client krok.CommandClient
}

// GRPCServer is the grpc server implementation which calls the
// protoc generated code to register it.
func (p *CommandGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	krok.RegisterCommandServer(s, &GRPCCommandServer{Impl: p.Impl})
	return nil
}

// GRPCClient is the grpc client that will talk to the GRPC Server
// and calls into the generated protoc code.
func (p *CommandGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCCommandClient{client: krok.NewCommandClient(c)}, nil
}

// Execute is the GRPC implementation of the Execute function for the
// Archive plugin definition. This will talk over GRPC.
func (m *GRPCCommandClient) Execute(raw string) (string, bool, error) {
	r, err := m.client.Execute(context.Background(), &krok.ExecuteRequest{
		Raw: raw,
	})
	if err != nil {
		return "", false, err
	}
	return r.Outcome, r.Success, nil
}

// GRPCCommandServer is the gRPC server that GRPCCommandClient talks to.
type GRPCCommandServer struct {
	// This is the real implementation
	Impl Command
}

// Execute is the execute function of the GRPCServer which will rely the information to the
// underlying implementation of this interface.
func (m *GRPCCommandServer) Execute(ctx context.Context, req *krok.ExecuteRequest) (*krok.ExecuteResponse, error) {
	out, res, err := m.Impl.Execute(req.Raw)
	return &krok.ExecuteResponse{
		Outcome: out,
		Success: res,
	}, err
}
