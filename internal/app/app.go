package app

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"netmap/internal/config"
	"netmap/internal/model"
	"netmap/internal/provider"
	"netmap/internal/render"
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
	now      func() time.Time
	status   io.Writer
}

type Options struct {
	Type          string
	Org           string
	Workload      string
	Environment   string
	SourceProject string
	Format        string
	ConfigPath    string
	ShowHelp      bool
	Usage         string
}

func New(files FileStore, discovery provider.DiscoveryProvider) (*App, error) {
	if files == nil {
		return nil, errors.New("file store is required")
	}
	if discovery == nil {
		return nil, errors.New("provider is required")
	}
	return &App{
		files:    files,
		provider: discovery,
		now:      time.Now,
		status:   os.Stderr,
	}, nil
}

func (a *App) Run(ctx context.Context, args []string) error {
	opts, err := ParseOptions(args)
	if err != nil {
		return err
	}
	if opts.ShowHelp {
		fmt.Fprint(os.Stdout, opts.Usage)
		return nil
	}

	cfgData, err := a.files.ReadFile(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("read config %q: %w", opts.ConfigPath, err)
	}

	cfg, err := config.Parse(cfgData)
	if err != nil {
		return err
	}

	targets, err := cfg.ResolveTargets(opts.Org, opts.Workload, opts.Environment)
	if err != nil {
		return err
	}
	projectIDs := uniqueProjectIDs(targets)

	if opts.Type == TypeVPN {
		scope := "selected destination projects"
		if len(projectIDs) == 1 {
			scope = fmt.Sprintf("destination project %q", projectIDs[0])
		}
		return fmt.Errorf("vpn is not implemented yet for %s", scope)
	}

	a.statusf(
		"⏳ Running netmap for org=%s workload=%s environment=%s source_project=%s",
		opts.Org,
		statusValueOrAll(opts.Workload),
		statusValueOrAll(opts.Environment),
		opts.SourceProject,
	)

	report, err := a.buildInterconnectReport(ctx, opts, targets)
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

	timestamp := a.now().UTC().Format("20060102T150405Z")
	outputPath := defaultOutputPath(outputFormat, opts, report, ext, timestamp)
	if err := a.files.WriteFile(outputPath, data); err != nil {
		return fmt.Errorf("write output %q: %w", outputPath, err)
	}

	a.statusf("output file: %s", outputPath)
	return nil
}

func ParseOptions(args []string) (Options, error) {
	fs := flag.NewFlagSet("netmap", flag.ContinueOnError)
	var usage bytes.Buffer
	fs.SetOutput(&usage)

	var opts Options
	fs.StringVar(&opts.Type, "t", "", "resource type: interconnect or vpn")
	fs.StringVar(&opts.Org, "o", "", "org selector")
	fs.StringVar(&opts.Workload, "w", "", "workload selector")
	fs.StringVar(&opts.Environment, "e", "", "environment selector")
	fs.StringVar(&opts.SourceProject, "p", "", "source project for interconnect discovery")
	fs.StringVar(&opts.Format, "f", "", "optional output format: csv, tsv, json, tree")
	fs.StringVar(&opts.ConfigPath, "config", "config.yaml", "config path")
	fs.Usage = func() {
		fmt.Fprint(&usage, usageText())
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return Options{
				ShowHelp: true,
				Usage:    usage.String(),
			}, nil
		}
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

func usageText() string {
	return strings.TrimSpace(`
Usage:
  netmap -t interconnect -o <org> [-w <workload>] [-e <env>] -p <src-project> [-f <format>] [-config <path>]
  netmap -t vpn -o <org> [-w <workload>] [-e <env>] [-f <format>] [-config <path>]

Flags:
  -t        mandatory, accepts interconnect or vpn
  -o        mandatory, org lookup key from the YAML config
  -w        optional, workload selector; with -o and no -e, expands all environments in that workload
  -e        optional, environment selector; with -o and no -w, expands all workloads containing that environment
  -p        mandatory only for -t interconnect; source project containing dedicated interconnects
  -f        optional, output format override: csv, tsv, json, or tree
  -config   optional, defaults to config.yaml
  -h        optional, print usage

Selector Expansion:
  -o only        expands all workloads and environments under that org
  -o + -w        expands all environments under that workload
  -o + -e        expands all workloads containing that environment
  -o + -w + -e   resolves one exact workload/environment tuple

Output:
  Omit -f to write Mermaid output by default.
  Mermaid output file: netmap-interconnect-<src>-to-<dst>-<timestamp>.mmd
  CSV output file:     netmap-interconnect-<src>-to-<dst>-<timestamp>.csv
  TSV output file:     netmap-interconnect-<src>-to-<dst>-<timestamp>.tsv
  JSON output file:    netmap-interconnect-<src>-to-<dst>-<timestamp>.json
  Tree output file:    netmap-interconnect-<src>-to-<dst>-<timestamp>.tree.txt
  Org fanout output:   netmap-interconnect-<src>-to-<org>-all-<timestamp>.<ext>
  On success, the CLI prints: output file: <path>
  Mermaid output can be viewed in https://mermaid.live
`) + "\n"
}

func (a *App) buildInterconnectReport(ctx context.Context, opts Options, targets []config.ResolvedTarget) (model.Report, error) {
	interconnects, err := a.provider.ListDedicatedInterconnects(ctx, opts.SourceProject)
	if err != nil {
		return model.Report{}, err
	}
	if len(interconnects) == 0 {
		return model.Report{}, fmt.Errorf("no dedicated interconnects found in source project %q", opts.SourceProject)
	}

	var items []model.MappingItem
	itemsByProject := make(map[string][]model.MappingItem, len(targets))
	for _, target := range targets {
		projectItems, ok := itemsByProject[target.ProjectID]
		if !ok {
			attachments, err := a.provider.ListVLANAttachments(ctx, target.ProjectID)
			if err != nil {
				return model.Report{}, err
			}
			routers, err := a.provider.ListCloudRouters(ctx, target.ProjectID)
			if err != nil {
				return model.Report{}, err
			}

			statusByRouter := make(map[string]model.RouterStatus, len(routers))
			for _, router := range routers {
				status, err := a.provider.GetCloudRouterStatus(ctx, target.ProjectID, router.Region, router.Name)
				if err != nil {
					return model.Report{}, err
				}
				statusByRouter[routerKey(router.Region, router.Name)] = status
			}

			projectItems = buildMappingItems(opts.SourceProject, target.ProjectID, interconnects, attachments, routers, statusByRouter)
			itemsByProject[target.ProjectID] = projectItems
		}

		items = append(items, itemsForTarget(target, projectItems)...)

		a.statusf(
			"⏳ Completed org=%s workload=%s environment=%s project=%s",
			target.Org,
			target.Workload,
			target.Environment,
			target.ProjectID,
		)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Org != items[j].Org {
			return items[i].Org < items[j].Org
		}
		if items[i].Workload != items[j].Workload {
			return items[i].Workload < items[j].Workload
		}
		if items[i].Environment != items[j].Environment {
			return items[i].Environment < items[j].Environment
		}
		if items[i].SrcProject != items[j].SrcProject {
			return items[i].SrcProject < items[j].SrcProject
		}
		if items[i].SrcInterconnect != items[j].SrcInterconnect {
			return items[i].SrcInterconnect < items[j].SrcInterconnect
		}
		if items[i].DstProject != items[j].DstProject {
			return items[i].DstProject < items[j].DstProject
		}
		if items[i].DstRegion != items[j].DstRegion {
			return items[i].DstRegion < items[j].DstRegion
		}
		if items[i].DstVLANAttachment != items[j].DstVLANAttachment {
			return items[i].DstVLANAttachment < items[j].DstVLANAttachment
		}
		if items[i].DstCloudRouter != items[j].DstCloudRouter {
			return items[i].DstCloudRouter < items[j].DstCloudRouter
		}
		if items[i].DstCloudRouterInterface != items[j].DstCloudRouterInterface {
			return items[i].DstCloudRouterInterface < items[j].DstCloudRouterInterface
		}
		return items[i].RemoteBGPPeer < items[j].RemoteBGPPeer
	})

	destinationProject := ""
	projectIDs := uniqueProjectIDs(targets)
	if len(projectIDs) == 1 {
		destinationProject = projectIDs[0]
	}
	return model.Report{
		Type:               opts.Type,
		SourceProject:      opts.SourceProject,
		DestinationProject: destinationProject,
		Selectors: model.Selectors{
			Org:         opts.Org,
			Workload:    opts.Workload,
			Environment: opts.Environment,
		},
		Items: items,
	}, nil
}

func (a *App) statusf(format string, args ...any) {
	if a.status == nil {
		return
	}
	fmt.Fprintf(a.status, format+"\n", args...)
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
				SrcProject:       srcProject,
				SrcInterconnect:  interconnect.Name,
				SrcRegion:        "global",
				SrcState:         interconnect.State,
				SrcMacsecEnabled: interconnect.MacsecEnabled,
				SrcMacsecKeyName: interconnect.MacsecKeyName,
				DstProject:       dstProject,
				Mapped:           false,
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
					item.DstCloudRouterInterface = iface.Name
					item.DstCloudRouterInterfaceIP = iface.IPRange
					item.BGPPeeringStatus = "unknown"
					items = append(items, item)
					continue
				}
				for _, peer := range peers {
					item := baseItem(srcProject, dstProject, interconnect, attachment, router)
					item.DstCloudRouterInterface = iface.Name
					item.DstCloudRouterInterfaceIP = firstNonEmpty(peer.LocalIP, iface.IPRange)
					item.RemoteBGPPeer = peer.Name
					item.RemoteBGPPeerIP = peer.RemoteIP
					item.BGPPeeringStatus = firstNonEmpty(peer.SessionState, "unknown")
					items = append(items, item)
				}
			}
		}
	}
	return items
}

func baseItem(srcProject, dstProject string, interconnect model.DedicatedInterconnect, attachment model.VLANAttachment, router model.CloudRouter) model.MappingItem {
	return model.MappingItem{
		SrcProject:              srcProject,
		SrcInterconnect:         interconnect.Name,
		Mapped:                  true,
		SrcRegion:               "global",
		SrcState:                interconnect.State,
		SrcMacsecEnabled:        interconnect.MacsecEnabled,
		SrcMacsecKeyName:        interconnect.MacsecKeyName,
		DstProject:              dstProject,
		DstRegion:               attachment.Region,
		DstVLANAttachment:       attachment.Name,
		DstVLANAttachmentState:  attachment.State,
		DstVLANAttachmentVLANID: attachment.VLANID,
		DstCloudRouter:          router.Name,
		DstCloudRouterASN:       router.ASN,
	}
}

func itemsForTarget(target config.ResolvedTarget, base []model.MappingItem) []model.MappingItem {
	items := make([]model.MappingItem, 0, len(base))
	for _, item := range base {
		current := item
		current.Org = target.Org
		current.Workload = target.Workload
		current.Environment = target.Environment
		items = append(items, current)
	}
	return items
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

func defaultOutputPath(format string, opts Options, report model.Report, ext, timestamp string) string {
	target := report.DestinationProject
	if strings.TrimSpace(target) == "" {
		target = opts.Org + "-all"
	}
	base := fmt.Sprintf("netmap-interconnect-%s-to-%s-%s", opts.SourceProject, target, timestamp)
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

func statusValueOrAll(value string) string {
	if strings.TrimSpace(value) == "" {
		return "all"
	}
	return value
}

func uniqueProjectIDs(targets []config.ResolvedTarget) []string {
	var projectIDs []string
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if _, ok := seen[target.ProjectID]; ok {
			continue
		}
		seen[target.ProjectID] = struct{}{}
		projectIDs = append(projectIDs, target.ProjectID)
	}
	return projectIDs
}
