package provider

import (
	"context"

	"mindmap/internal/model"
)

type DiscoveryProvider interface {
	ListDedicatedInterconnects(ctx context.Context, project string) ([]model.DedicatedInterconnect, error)
	ListVLANAttachments(ctx context.Context, project string) ([]model.VLANAttachment, error)
	ListCloudRouters(ctx context.Context, project string) ([]model.CloudRouter, error)
	GetCloudRouterStatus(ctx context.Context, project, region, router string) (model.RouterStatus, error)
}
