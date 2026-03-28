package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	compute "google.golang.org/api/compute/v1"

	"netmap/internal/model"
)

type ComputeProvider struct {
	service *compute.Service
}

func NewComputeProvider(ctx context.Context) (*ComputeProvider, error) {
	service, err := compute.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create compute service: %w", err)
	}
	return &ComputeProvider{service: service}, nil
}

func (p *ComputeProvider) ListDedicatedInterconnects(ctx context.Context, project string) ([]model.DedicatedInterconnect, error) {
	var items []model.DedicatedInterconnect
	call := p.service.Interconnects.List(project).Context(ctx)
	if err := call.Pages(ctx, func(page *compute.InterconnectList) error {
		for _, interconnect := range page.Items {
			items = append(items, model.DedicatedInterconnect{
				Name:          interconnect.Name,
				State:         firstNonEmpty(interconnect.OperationalStatus, interconnect.State, "unknown"),
				MacsecEnabled: interconnect.MacsecEnabled,
				MacsecKeyName: selectActiveMacsecKeyName(time.Now().UTC(), interconnect.Macsec),
			})
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list dedicated interconnects for source project %q: %w", project, err)
	}
	return items, nil
}

func (p *ComputeProvider) ListVLANAttachments(ctx context.Context, project string) ([]model.VLANAttachment, error) {
	var items []model.VLANAttachment
	call := p.service.InterconnectAttachments.AggregatedList(project).Context(ctx)
	if err := call.Pages(ctx, func(page *compute.InterconnectAttachmentAggregatedList) error {
		for _, scoped := range page.Items {
			for _, attachment := range scoped.InterconnectAttachments {
				items = append(items, model.VLANAttachment{
					Name:         attachment.Name,
					Region:       basename(attachment.Region),
					State:        firstNonEmpty(attachment.OperationalStatus, attachment.State, "unknown"),
					Interconnect: basename(attachment.Interconnect),
					Router:       basename(attachment.Router),
					VLANID:       formatVLANID(attachment.VlanTag8021q),
				})
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list vlan attachments for destination project %q: %w", project, err)
	}
	return items, nil
}

func (p *ComputeProvider) ListCloudRouters(ctx context.Context, project string) ([]model.CloudRouter, error) {
	var items []model.CloudRouter
	call := p.service.Routers.AggregatedList(project).Context(ctx)
	if err := call.Pages(ctx, func(page *compute.RouterAggregatedList) error {
		for _, scoped := range page.Items {
			for _, router := range scoped.Routers {
				current := model.CloudRouter{
					Name:   router.Name,
					Region: basename(router.Region),
					ASN:    formatASN(router.Bgp),
				}
				for _, iface := range router.Interfaces {
					if iface == nil {
						continue
					}
					current.Interfaces = append(current.Interfaces, model.RouterInterface{
						Name:                     iface.Name,
						LinkedInterconnectAttach: basename(iface.LinkedInterconnectAttachment),
						IPRange:                  iface.IpRange,
					})
				}
				for _, peer := range router.BgpPeers {
					if peer == nil {
						continue
					}
					current.BGPPeers = append(current.BGPPeers, model.BGPPeer{
						Name:         peer.Name,
						Interface:    peer.InterfaceName,
						LocalIP:      peer.IpAddress,
						RemoteIP:     peer.PeerIpAddress,
						SessionState: "",
					})
				}
				items = append(items, current)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list cloud routers for destination project %q: %w", project, err)
	}
	return items, nil
}

func (p *ComputeProvider) GetCloudRouterStatus(ctx context.Context, project, region, router string) (model.RouterStatus, error) {
	response, err := p.service.Routers.GetRouterStatus(project, region, router).Context(ctx).Do()
	if err != nil {
		return model.RouterStatus{}, fmt.Errorf("get status for router %q in region %q: %w", router, region, err)
	}

	status := model.RouterStatus{
		RouterName: router,
		Region:     region,
	}
	if response.Result == nil {
		return status, nil
	}
	for _, peer := range response.Result.BgpPeerStatus {
		if peer == nil {
			continue
		}
		status.Peers = append(status.Peers, model.BGPPeerStatus{
			Name:         peer.Name,
			LocalIP:      peer.IpAddress,
			RemoteIP:     peer.PeerIpAddress,
			SessionState: firstNonEmpty(peer.Status, peer.State),
		})
	}
	return status, nil
}

func basename(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(strings.TrimRight(value, "/"), "/")
	return parts[len(parts)-1]
}

func formatVLANID(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func formatASN(bgp *compute.RouterBgp) string {
	if bgp == nil || bgp.Asn <= 0 {
		return ""
	}
	return strconv.FormatInt(bgp.Asn, 10)
}

func selectActiveMacsecKeyName(now time.Time, macsec *compute.InterconnectMacsec) string {
	if macsec == nil || len(macsec.PreSharedKeys) == 0 {
		return ""
	}

	var activeName string
	var activeStart time.Time
	var haveActive bool
	var latestName string
	var latestStart time.Time
	var haveLatest bool

	for _, key := range macsec.PreSharedKeys {
		if key == nil || strings.TrimSpace(key.Name) == "" {
			continue
		}

		startTime, ok := parseRFC3339(key.StartTime)
		if !haveLatest || compareStartTime(startTime, ok, latestStart, haveLatest) > 0 {
			latestName = key.Name
			latestStart = startTime
			haveLatest = ok || !haveLatest
		}

		if ok && startTime.After(now) {
			continue
		}
		if !haveActive || compareStartTime(startTime, ok, activeStart, haveActive) > 0 {
			activeName = key.Name
			activeStart = startTime
			haveActive = ok || !haveActive
		}
	}

	if strings.TrimSpace(activeName) != "" {
		return activeName
	}
	return latestName
}

func parseRFC3339(value string) (time.Time, bool) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func compareStartTime(current time.Time, currentOK bool, best time.Time, bestOK bool) int {
	switch {
	case currentOK && !bestOK:
		return 1
	case !currentOK && bestOK:
		return -1
	case !currentOK && !bestOK:
		return 0
	case current.After(best):
		return 1
	case current.Before(best):
		return -1
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
