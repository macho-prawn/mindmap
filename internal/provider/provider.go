package provider

import (
	"context"

	"netmap/internal/model"
)

type DiscoveryProvider interface {
	ListDedicatedInterconnects(ctx context.Context, project string) ([]model.DedicatedInterconnect, error)
	ListVLANAttachments(ctx context.Context, project string) ([]model.VLANAttachment, error)
	ListVPNGateways(ctx context.Context, project string) ([]model.VPNGateway, error)
	ListTargetVPNGateways(ctx context.Context, project string) ([]model.VPNGateway, error)
	ListVPNTunnels(ctx context.Context, project string) ([]model.VPNTunnel, error)
	ListCloudRouters(ctx context.Context, project string) ([]model.CloudRouter, error)
	GetCloudRouterStatus(ctx context.Context, project, region, router string) (model.RouterStatus, error)
}
