package client

import (
	"context"
	"fmt"

	myshoespb "github.com/whywaita/myshoes/api/proto.go"
	shoesvzpb "github.com/whywaita/shoes-vz/gen/go/shoes/vz/shoes/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Client implements the ShoesServer interface and acts as a bridge between
// myshoes and shoes-vz-server
type Client struct {
	myshoespb.UnimplementedShoesServer
	conn       *grpc.ClientConn
	vzClient   shoesvzpb.ShoesServiceClient
	serverAddr string
}

// NewClient creates a new Client instance
func NewClient(config *Config) (*Client, error) {
	conn, err := grpc.NewClient(
		config.ServerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	vzClient := shoesvzpb.NewShoesServiceClient(conn)

	return &Client{
		conn:       conn,
		vzClient:   vzClient,
		serverAddr: config.ServerAddr,
	}, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// AddInstance implements the ShoesServer interface
// It converts myshoes AddInstanceRequest to shoes-vz AddInstanceRequest,
// calls shoes-vz-server, and converts the response back
func (c *Client) AddInstance(ctx context.Context, req *myshoespb.AddInstanceRequest) (*myshoespb.AddInstanceResponse, error) {
	// Convert resource type from enum to string
	resourceType, err := ConvertResourceType(req.ResourceType)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid resource type: %v", err)
	}

	// Create shoes-vz request
	vzReq := &shoesvzpb.AddInstanceRequest{
		RunnerName:   req.RunnerName,
		SetupScript:  req.SetupScript,
		ResourceType: resourceType,
		Labels:       req.Labels,
	}

	// Call shoes-vz-server
	vzResp, err := c.vzClient.AddInstance(ctx, vzReq)
	if err != nil {
		return nil, err
	}

	// Convert response back to myshoes format
	resp := &myshoespb.AddInstanceResponse{
		CloudId:      vzResp.CloudId,
		ShoesType:    vzResp.ShoesType,
		IpAddress:    vzResp.IpAddress,
		ResourceType: req.ResourceType, // Keep the original resource type
	}

	return resp, nil
}

// DeleteInstance implements the ShoesServer interface
// It converts myshoes DeleteInstanceRequest to shoes-vz DeleteInstanceRequest
// and calls shoes-vz-server
func (c *Client) DeleteInstance(ctx context.Context, req *myshoespb.DeleteInstanceRequest) (*myshoespb.DeleteInstanceResponse, error) {
	// Create shoes-vz request
	vzReq := &shoesvzpb.DeleteInstanceRequest{
		CloudId: req.CloudId,
	}

	// Call shoes-vz-server
	_, err := c.vzClient.DeleteInstance(ctx, vzReq)
	if err != nil {
		return nil, err
	}

	// Return empty response
	return &myshoespb.DeleteInstanceResponse{}, nil
}
