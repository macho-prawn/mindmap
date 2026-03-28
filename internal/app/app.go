package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"mindmap/internal/config"
	"mindmap/internal/model"
	"mindmap/internal/provider"
	"mindmap/internal/render"
)

const (
	TypeInterconnect = "interconnect"
	TypeVPN          = "vpn"
)

type FileStore interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte) error
}

type RealFileStore struct{}

func (RealFileStore) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (RealFileStore) WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o644)
}

type App struct {
	files    FileStore
	provider provider.DiscoveryProvider
}

type Options struct {
	Type          string
	Org           string
	Workload      string
	Environment   string
	SourceProject string
	Format        string
	ConfigPath    string
}

func New(files FileStore, discovery provider.DiscoveryProvider) (*App, error) {
	if files == nil {
		return nil, errors.New("file store is required")
	}
	if discovery == nil {
		return nil, errors.New("provider is required")
	}
	return &App{files: files, provider: discovery}, nil
}

func (a *App) Run(ctx context.Context, args []string) error {
	opts, err := ParseOptions(args)
	if err != nil {
		return err
	}

	cfgData, err := a.files.ReadFile(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("read config %q: %w", opts.ConfigPath, err)
	}

	cfg, err := config.Parse(cfgData)
	if err != nil {
		return err
	}

	dstProject, err := cfg.Resolve(opts.Org, opts.Workload, opts.Environment)
	if err != nil {
		return err
	}

	if opts.Type == TypeVPN {
		return fmt.Errorf("vpn is not implemented yet for destination project %q", dstProject)
	}

	report, err := a.buildInterconnectReport(ctx, opts, dstProject)
	if err != nil {
		return err
	}

	outputFormat := opts.Format
	if outputFormat == "" {
		outputFormat = render.FormatMermaid
	}

	data, ext, err := render.Render(report, outputFormat)
	if err != nil {
		return err
	}

	outputPath := defaultOutputPath(outputFormat, opts.SourceProject, dstProject, ext)
	if err := a.files.WriteFile(outputPath, data); err != nil {
		return fmt.Errorf("write output %q: %w", outputPath, err)
	}

	fmt.Fprintf(io.Discard, "%s", outputPath)
	return nil
}

func ParseOptions(args []string) (Options, error) {
	fs := flag.NewFlagSet("mindmap", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var opts Options
	fs.StringVar(&opts.Type, "t", "", "resource type: interconnect or vpn")
	fs.StringVar(&opts.Org, "o", "", "org selector")
	fs.StringVar(&opts.Workload, "w", "", "workload selector")
	fs.StringVar(&opts.Environment, "e", "", "environment selector")
	fs.StringVar(&opts.SourceProject, "p", "", "source project for interconnect discovery")
	fs.StringVar(&opts.Format, "f", "", "optional output format: csv, tsv, json, tree")
	fs.StringVar(&opts.ConfigPath, "config", "config.yaml", "config path")

	if err := fs.Parse(args); err != nil {
		return Options{}, err
	}

	if strings.TrimSpace(opts.Type) == "" {
		return Options{}, fmt.Errorf("missing mandatory parameter -t")
	}
	if opts.Type != TypeInterconnect && opts.Type != TypeVPN {
		return Options{}, fmt.Errorf("invalid -t value %q, expected interconnect or vpn", opts.Type)
	}
	if strings.TrimSpace(opts.Org) == "" {
		return Options{}, fmt.Errorf("missing mandatory parameter -o")
	}
	if strings.TrimSpace(opts.Workload) == "" {
		return Options{}, fmt.Errorf("missing mandatory parameter -w")
	}
	if strings.TrimSpace(opts.Environment) == "" {
		return Options{}, fmt.Errorf("missing mandatory parameter -e")
	}
	if opts.Type == TypeInterconnect && strings.TrimSpace(opts.SourceProject) == "" {
		return Options{}, fmt.Errorf("missing mandatory parameter -p for -t interconnect")
	}
	if opts.Type == TypeVPN && strings.TrimSpace(opts.SourceProject) != "" {
		return Options{}, fmt.Errorf("-p must not be used with -t vpn")
	}

	switch opts.Format {
	case "", render.FormatCSV, render.FormatTSV, render.FormatJSON, render.FormatTree:
	default:
		return Options{}, fmt.Errorf("invalid -f value %q, expected csv, tsv, json, or tree", opts.Format)
	}

	return opts, nil
}

func (a *App) buildInterconnectReport(ctx context.Context, opts Options, dstProject string) (model.Report, error) {
	interconnects, err := a.provider.ListDedicatedInterconnects(ctx, opts.SourceProject)
	if err != nil {
		return model.Report{}, err
	}
	if len(interconnects) == 0 {
		return model.Report{}, fmt.Errorf("no dedicated interconnects found in source project %q", opts.SourceProject)
	}

	attachments, err := a.provider.ListVLANAttachments(ctx, dstProject)
	if err != nil {
		return model.Report{}, err
	}
	routers, err := a.provider.ListCloudRouters(ctx, dstProject)
	if err != nil {
		return model.Report{}, err
	}

	statusByRouter := make(map[string]model.RouterStatus, len(routers))
	for _, router := range routers {
		status, err := a.provider.GetCloudRouterStatus(ctx, dstProject, router.Region, router.Name)
		if err != nil {
			return model.Report{}, err
		}
		statusByRouter[routerKey(router.Region, router.Name)] = status
	}

	items := buildMappingItems(opts.SourceProject, dstProject, interconnects, attachments, routers, statusByRouter)
	sort.Slice(items, func(i, j int) bool {
		if items[i].SrcInterconnect != items[j].SrcInterconnect {
			return items[i].SrcInterconnect < items[j].SrcInterconnect
		}
		if items[i].Region != items[j].Region {
			return items[i].Region < items[j].Region
		}
		if items[i].Attachment != items[j].Attachment {
			return items[i].Attachment < items[j].Attachment
		}
		if items[i].Router != items[j].Router {
			return items[i].Router < items[j].Router
		}
		return items[i].Interface < items[j].Interface
	})

	return model.Report{
		Type:               opts.Type,
		SourceProject:      opts.SourceProject,
		DestinationProject: dstProject,
		Selectors: model.Selectors{
			Org:         opts.Org,
			Workload:    opts.Workload,
			Environment: opts.Environment,
		},
		Items: items,
	}, nil
}

func buildMappingItems(srcProject, dstProject string, interconnects []model.DedicatedInterconnect, attachments []model.VLANAttachment, routers []model.CloudRouter, statuses map[string]model.RouterStatus) []model.MappingItem {
	attachmentsByInterconnect := make(map[string][]model.VLANAttachment)
	for _, attachment := range attachments {
		attachmentsByInterconnect[attachment.Interconnect] = append(attachmentsByInterconnect[attachment.Interconnect], attachment)
	}

	routerByNameRegion := make(map[string]model.CloudRouter)
	for _, router := range routers {
		routerByNameRegion[routerKey(router.Region, router.Name)] = router
	}

	var items []model.MappingItem
	for _, interconnect := range interconnects {
		matches := attachmentsByInterconnect[interconnect.Name]
		if len(matches) == 0 {
			items = append(items, model.MappingItem{
				SrcProject:      srcProject,
				SrcInterconnect: interconnect.Name,
				SrcRegion:       "global",
				SrcState:        interconnect.State,
				DstProject:      dstProject,
				Mapped:          false,
			})
			continue
		}

		for _, attachment := range matches {
			router := routerByNameRegion[routerKey(attachment.Region, attachment.Router)]
			interfaces := interfacesForAttachment(router, attachment.Name)
			if len(interfaces) == 0 {
				items = append(items, baseItem(srcProject, dstProject, interconnect, attachment, router))
				continue
			}

			peersByInterface := peersForRouter(router, statuses[routerKey(router.Region, router.Name)])
			for _, iface := range interfaces {
				peers := peersByInterface[iface.Name]
				if len(peers) == 0 {
					item := baseItem(srcProject, dstProject, interconnect, attachment, router)
					item.Interface = iface.Name
					item.LocalIP = iface.IPRange
					item.BGPStatus = "unknown"
					items = append(items, item)
					continue
				}
				for _, peer := range peers {
					item := baseItem(srcProject, dstProject, interconnect, attachment, router)
					item.Interface = iface.Name
					item.BGPPeerName = peer.Name
					item.LocalIP = firstNonEmpty(peer.LocalIP, iface.IPRange)
					item.RemoteIP = peer.RemoteIP
					item.BGPStatus = firstNonEmpty(peer.SessionState, "unknown")
					items = append(items, item)
				}
			}
		}
	}
	return items
}

func baseItem(srcProject, dstProject string, interconnect model.DedicatedInterconnect, attachment model.VLANAttachment, router model.CloudRouter) model.MappingItem {
	return model.MappingItem{
		SrcProject:      srcProject,
		SrcInterconnect: interconnect.Name,
		SrcRegion:       "global",
		SrcState:        interconnect.State,
		DstProject:      dstProject,
		Region:          attachment.Region,
		Attachment:      attachment.Name,
		AttachmentState: attachment.State,
		Router:          router.Name,
		Mapped:          true,
	}
}

func interfacesForAttachment(router model.CloudRouter, attachment string) []model.RouterInterface {
	var result []model.RouterInterface
	for _, iface := range router.Interfaces {
		if iface.LinkedInterconnectAttach == attachment {
			result = append(result, iface)
		}
	}
	return result
}

func peersForRouter(router model.CloudRouter, status model.RouterStatus) map[string][]model.BGPPeer {
	statusByName := make(map[string]model.BGPPeerStatus, len(status.Peers))
	for _, peerStatus := range status.Peers {
		statusByName[peerStatus.Name] = peerStatus
	}

	result := make(map[string][]model.BGPPeer)
	for _, peer := range router.BGPPeers {
		if peer.Interface == "" {
			continue
		}
		merged := peer
		if status, ok := statusByName[peer.Name]; ok {
			merged.LocalIP = firstNonEmpty(status.LocalIP, merged.LocalIP)
			merged.RemoteIP = firstNonEmpty(status.RemoteIP, merged.RemoteIP)
			merged.SessionState = firstNonEmpty(status.SessionState, merged.SessionState)
		}
		result[peer.Interface] = append(result[peer.Interface], merged)
	}
	return result
}

func defaultOutputPath(format, srcProject, dstProject, ext string) string {
	base := fmt.Sprintf("mindmap-interconnect-%s-to-%s", srcProject, dstProject)
	if format == render.FormatJSON {
		return base + ".json"
	}
	if format == render.FormatTree {
		return base + ".tree.txt"
	}
	return base + "." + ext
}

func routerKey(region, name string) string {
	return region + "/" + name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
