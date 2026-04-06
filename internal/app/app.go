package app

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"netmap/internal/config"
	"netmap/internal/model"
	"netmap/internal/provider"
	"netmap/internal/render"
)

const (
	TypeInterconnect = "interconnect"
	TypeVPN          = "vpn"
)

var brailleSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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
	files       FileStore
	provider    provider.DiscoveryProvider
	now         func() time.Time
	status      io.Writer
	statusTable *statusTable
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

type ValidatedInput struct {
	Options Options
	Targets []config.ResolvedTarget
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
	input, err := Validate(a.files, args)
	if err != nil {
		return err
	}
	if input.Options.ShowHelp {
		fmt.Fprint(os.Stdout, input.Options.Usage)
		return nil
	}
	return a.RunValidated(ctx, input)
}

func (a *App) RunValidated(ctx context.Context, input ValidatedInput) error {
	opts := input.Options
	targets := input.Targets

	if opts.ShowHelp {
		fmt.Fprint(os.Stdout, opts.Usage)
		return nil
	}

	a.startStatusTable()
	defer a.stopStatusTable()

	var (
		report model.Report
		err    error
	)
	switch opts.Type {
	case TypeVPN:
		report, err = a.buildVPNReport(ctx, opts, targets)
	default:
		report, err = a.buildInterconnectReport(ctx, opts, targets)
	}
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

	a.finishStatusTable(outputPath)
	return nil
}

func Validate(files FileStore, args []string) (ValidatedInput, error) {
	if files == nil {
		return ValidatedInput{}, errors.New("file store is required")
	}

	opts, err := ParseOptions(args)
	if err != nil {
		return ValidatedInput{}, err
	}
	if opts.ShowHelp {
		return ValidatedInput{Options: opts}, nil
	}

	cfgData, err := files.ReadFile(opts.ConfigPath)
	if err != nil {
		return ValidatedInput{}, fmt.Errorf("read config %q: %w", opts.ConfigPath, err)
	}

	cfg, err := config.Parse(cfgData)
	if err != nil {
		return ValidatedInput{}, err
	}

	targets, err := cfg.ResolveTargets(opts.Org, opts.Workload, opts.Environment)
	if err != nil {
		return ValidatedInput{}, err
	}

	return ValidatedInput{
		Options: opts,
		Targets: targets,
	}, nil
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
	fs.StringVar(&opts.Format, "f", "", "optional output format: html, csv, tsv, json, tree")
	fs.StringVar(&opts.ConfigPath, "c", "config.yaml", "config path")
	fs.Usage = func() {
		fmt.Fprint(&usage, usageText())
	}

	if len(args) == 0 {
		fs.Usage()
		return Options{
			ShowHelp: true,
			Usage:    usage.String(),
		}, nil
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
	case "", render.FormatHTML, render.FormatCSV, render.FormatTSV, render.FormatJSON, render.FormatTree:
	default:
		return Options{}, fmt.Errorf("invalid -f value %q, expected html, csv, tsv, json, or tree", opts.Format)
	}

	return opts, nil
}

//go:embed usage.txt
var usageTextContent string

func usageText() string { return usageTextContent }

type vpnProjectData struct {
	Gateways      []model.VPNGateway
	Tunnels       []model.VPNTunnel
	Routers       []model.CloudRouter
	Statuses      map[string]model.RouterStatus
	GatewayByKey  map[string]model.VPNGateway
	GatewayByLink map[string]model.VPNGateway
	RouterByKey   map[string]model.CloudRouter
}

type vpnSourceInterfaceCandidate struct {
	RouterName    string
	RouterASN     string
	InterfaceName string
	InterfaceIP   string
	Peers         []model.BGPPeer
}

type vpnDestinationInterfaceCandidate struct {
	Region        string
	VPC           string
	GatewayName   string
	GatewayType   string
	TunnelName    string
	TunnelStatus  string
	RouterName    string
	RouterASN     string
	InterfaceName string
	InterfaceIP   string
	Peers         []model.BGPPeer
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
		label := taskLabel(target)
		startedAt := a.now()
		a.startTask(label)
		failTask := func(err error) (model.Report, error) {
			a.failTask(label, a.now().Sub(startedAt))
			return model.Report{}, err
		}

		projectItems, ok := itemsByProject[target.ProjectID]
		if !ok {
			attachments, err := a.provider.ListVLANAttachments(ctx, target.ProjectID)
			if err != nil {
				return failTask(err)
			}
			routers, err := a.provider.ListCloudRouters(ctx, target.ProjectID)
			if err != nil {
				return failTask(err)
			}

			statusByRouter := make(map[string]model.RouterStatus, len(routers))
			for _, router := range routers {
				status, err := a.provider.GetCloudRouterStatus(ctx, target.ProjectID, router.Region, router.Name)
				if err != nil {
					return failTask(err)
				}
				statusByRouter[routerKey(router.Region, router.Name)] = status
			}

			projectItems = buildMappingItems(opts.SourceProject, target.ProjectID, interconnects, attachments, routers, statusByRouter)
			itemsByProject[target.ProjectID] = projectItems
		}

		items = append(items, itemsForTarget(target, projectItems)...)
		a.completeTask(label, a.now().Sub(startedAt))
	}
	sortMappingItems(items)

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

func (a *App) buildVPNReport(ctx context.Context, opts Options, targets []config.ResolvedTarget) (model.Report, error) {
	var items []model.MappingItem
	var err error
	itemsBySourceProject := make(map[string][]model.MappingItem, len(targets))
	destinationProjects := make(map[string]struct{})
	destCache := make(map[string]vpnProjectData)

	for _, target := range targets {
		label := taskLabel(target)
		startedAt := a.now()
		a.startTask(label)
		failTask := func(err error) (model.Report, error) {
			a.failTask(label, a.now().Sub(startedAt))
			return model.Report{}, err
		}

		projectItems, ok := itemsBySourceProject[target.ProjectID]
		if !ok {
			projectItems, err = a.buildVPNProjectItems(ctx, target.ProjectID, destCache)
			if err != nil {
				return failTask(err)
			}
			itemsBySourceProject[target.ProjectID] = projectItems
		}
		for _, item := range projectItems {
			if strings.TrimSpace(item.DstProject) != "" {
				destinationProjects[item.DstProject] = struct{}{}
			}
		}

		items = append(items, itemsForTarget(target, projectItems)...)
		a.completeTask(label, a.now().Sub(startedAt))
	}

	sortMappingItems(items)

	sourceProject := ""
	sourceProjects := uniqueProjectIDs(targets)
	if len(sourceProjects) == 1 {
		sourceProject = sourceProjects[0]
	}

	destinationProject := ""
	if len(destinationProjects) == 1 {
		for projectID := range destinationProjects {
			destinationProject = projectID
		}
	}

	return model.Report{
		Type:               opts.Type,
		SourceProject:      sourceProject,
		DestinationProject: destinationProject,
		Selectors: model.Selectors{
			Org:         opts.Org,
			Workload:    opts.Workload,
			Environment: opts.Environment,
		},
		Items: items,
	}, nil
}

func (a *App) buildVPNProjectItems(ctx context.Context, sourceProject string, destCache map[string]vpnProjectData) ([]model.MappingItem, error) {
	sourceData, err := a.discoverVPNProject(ctx, sourceProject)
	if err != nil {
		return nil, err
	}
	if len(sourceData.Gateways) == 0 {
		return nil, fmt.Errorf("no vpn gateways found in source project %q", sourceProject)
	}

	tunnelsByGateway := make(map[string][]model.VPNTunnel)
	for _, tunnel := range sourceData.Tunnels {
		key := vpnTunnelGatewayKey(tunnel)
		if key == "" {
			continue
		}
		tunnelsByGateway[key] = append(tunnelsByGateway[key], tunnel)
	}

	var items []model.MappingItem
	for _, gateway := range sourceData.Gateways {
		tunnels := tunnelsByGateway[vpnGatewayKey(gateway.Type, gateway.Name)]
		sort.Slice(tunnels, func(i, j int) bool {
			if tunnels[i].Region != tunnels[j].Region {
				return tunnels[i].Region < tunnels[j].Region
			}
			return tunnels[i].Name < tunnels[j].Name
		})

		if len(tunnels) == 0 {
			items = append(items, vpnBaseItem(sourceProject, gateway, model.VPNTunnel{}))
			continue
		}

		for _, tunnel := range tunnels {
			base := vpnBaseItem(sourceProject, gateway, tunnel)
			sourceCandidates := vpnSourceInterfaceCandidates(sourceData, tunnel)

			if strings.TrimSpace(tunnel.PeerGCPGateway) == "" {
				items = append(items, vpnItemsForSourceCandidates(base, sourceCandidates)...)
				continue
			}

			dstProject := projectIDFromResourceURL(tunnel.PeerGCPGateway)
			if strings.TrimSpace(dstProject) == "" {
				items = append(items, vpnItemsForSourceCandidates(base, sourceCandidates)...)
				continue
			}
			base.Mapped = true
			base.DstProject = dstProject

			destData, ok := destCache[dstProject]
			if !ok {
				destData, err = a.discoverVPNProject(ctx, dstProject)
				if err != nil {
					return nil, err
				}
				destCache[dstProject] = destData
			}

			destGateway, ok := destData.GatewayByLink[strings.TrimSpace(tunnel.PeerGCPGateway)]
			if !ok {
				destGateway = destData.GatewayByKey[vpnGatewayKey("ha", resourceNameFromURL(tunnel.PeerGCPGateway))]
			}
			if strings.TrimSpace(destGateway.Name) != "" {
				base.DstRegion = firstNonEmpty(base.DstRegion, destGateway.Region)
				base.DstVPNGateway = destGateway.Name
				base.DstVPNGatewayType = destGateway.Type
				base.DstVPC = firstNonEmpty(base.DstVPC, destGateway.Network)
			}

			destCandidates := vpnDestinationInterfaceCandidates(tunnel, gateway, destGateway, destData)
			if len(destCandidates) == 0 {
				sourceItems := vpnItemsForSourceCandidates(base, sourceCandidates)
				for idx := range sourceItems {
					sourceItems[idx].BGPPeeringStatus = "unknown"
				}
				items = append(items, sourceItems...)
				continue
			}

			sharedDestination, hasSharedDestination := sharedVPNDestinationContext(destCandidates)
			availableDestinations := append([]vpnDestinationInterfaceCandidate(nil), destCandidates...)
			for _, sourceCandidate := range sourceCandidates {
				item := base
				applyVPNSourceCandidate(&item, sourceCandidate)

				if matchIndex, matched := chooseVPNDestinationCandidate(sourceCandidate, availableDestinations); matched {
					destinationCandidate := availableDestinations[matchIndex]
					applyVPNDestinationCandidate(&item, destinationCandidate)
					item.BGPPeeringStatus = vpnPairStatus(sourceCandidate, destinationCandidate)
					availableDestinations = append(availableDestinations[:matchIndex], availableDestinations[matchIndex+1:]...)
					items = append(items, item)
					continue
				}

				if hasSharedDestination {
					applyVPNDestinationCandidate(&item, sharedDestination)
					item.DstCloudRouterInterface = ""
					item.DstCloudRouterInterfaceIP = ""
				}
				item.BGPPeeringStatus = "unknown"
				items = append(items, item)
			}
		}
	}

	return items, nil
}

func (a *App) discoverVPNProject(ctx context.Context, project string) (vpnProjectData, error) {
	haGateways, err := a.provider.ListVPNGateways(ctx, project)
	if err != nil {
		return vpnProjectData{}, err
	}
	classicGateways, err := a.provider.ListTargetVPNGateways(ctx, project)
	if err != nil {
		return vpnProjectData{}, err
	}
	tunnels, err := a.provider.ListVPNTunnels(ctx, project)
	if err != nil {
		return vpnProjectData{}, err
	}
	routers, err := a.provider.ListCloudRouters(ctx, project)
	if err != nil {
		return vpnProjectData{}, err
	}

	statuses := make(map[string]model.RouterStatus, len(routers))
	for _, router := range routers {
		status, err := a.provider.GetCloudRouterStatus(ctx, project, router.Region, router.Name)
		if err != nil {
			return vpnProjectData{}, err
		}
		statuses[routerKey(router.Region, router.Name)] = status
	}

	gateways := append([]model.VPNGateway{}, haGateways...)
	gateways = append(gateways, classicGateways...)
	sort.Slice(gateways, func(i, j int) bool {
		if gateways[i].Region != gateways[j].Region {
			return gateways[i].Region < gateways[j].Region
		}
		if gateways[i].Type != gateways[j].Type {
			return gateways[i].Type < gateways[j].Type
		}
		return gateways[i].Name < gateways[j].Name
	})

	gatewayByKey := make(map[string]model.VPNGateway, len(gateways))
	gatewayByLink := make(map[string]model.VPNGateway, len(gateways))
	for _, gateway := range gateways {
		gatewayByKey[vpnGatewayKey(gateway.Type, gateway.Name)] = gateway
		if strings.TrimSpace(gateway.SelfLink) != "" {
			gatewayByLink[strings.TrimSpace(gateway.SelfLink)] = gateway
		}
	}

	routerByKey := make(map[string]model.CloudRouter, len(routers))
	for _, router := range routers {
		routerByKey[routerKey(router.Region, router.Name)] = router
	}

	return vpnProjectData{
		Gateways:      gateways,
		Tunnels:       tunnels,
		Routers:       routers,
		Statuses:      statuses,
		GatewayByKey:  gatewayByKey,
		GatewayByLink: gatewayByLink,
		RouterByKey:   routerByKey,
	}, nil
}

func (a *App) startStatusTable() {
	if a.status == nil {
		return
	}
	a.stopStatusTable()
	table := newStatusTable(a.status, a.now)
	a.statusTable = table
	table.Start()
}

func (a *App) stopStatusTable() {
	if a.statusTable == nil {
		return
	}
	a.statusTable.Stop()
	a.statusTable = nil
}

func (a *App) startTask(label string) {
	if a.statusTable == nil {
		return
	}
	a.statusTable.StartTask(label)
}

func (a *App) completeTask(label string, elapsed time.Duration) {
	if a.statusTable == nil {
		return
	}
	a.statusTable.CompleteTask(label, elapsed)
}

func (a *App) failTask(label string, elapsed time.Duration) {
	if a.statusTable == nil {
		return
	}
	a.statusTable.FailTask(label, elapsed)
}

func (a *App) finishStatusTable(outputPath string) {
	if a.statusTable == nil {
		return
	}
	a.statusTable.Finish(outputPath)
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
					item.RemoteBGPPeerASN = peer.PeerASN
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
		DstVPC:                  firstNonEmpty(router.Network, attachment.Network),
		DstVLANAttachment:       attachment.Name,
		DstVLANAttachmentState:  attachment.State,
		DstVLANAttachmentVLANID: attachment.VLANID,
		DstCloudRouter:          router.Name,
		DstCloudRouterASN:       router.ASN,
	}
}

func vpnBaseItem(srcProject string, gateway model.VPNGateway, tunnel model.VPNTunnel) model.MappingItem {
	item := model.MappingItem{
		SrcProject:         srcProject,
		SrcRegion:          firstNonEmpty(tunnel.Region, gateway.Region),
		SrcVPNGateway:      gateway.Name,
		SrcVPNGatewayType:  gateway.Type,
		SrcVPNTunnel:       tunnel.Name,
		SrcVPNTunnelStatus: firstNonEmpty(tunnel.Status),
		Mapped:             false,
	}
	return item
}

func vpnSourceInterfaceCandidates(sourceData vpnProjectData, tunnel model.VPNTunnel) []vpnSourceInterfaceCandidate {
	router := sourceData.RouterByKey[routerKey(tunnel.Region, tunnel.Router)]
	base := vpnSourceInterfaceCandidate{
		RouterName: firstNonEmpty(router.Name, tunnel.Router),
		RouterASN:  router.ASN,
	}
	interfaces := interfacesForTunnel(router, tunnel.Name)
	if len(interfaces) == 0 {
		return []vpnSourceInterfaceCandidate{base}
	}

	peersByInterface := peersForRouter(router, sourceData.Statuses[routerKey(router.Region, router.Name)])
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Name < interfaces[j].Name
	})

	candidates := make([]vpnSourceInterfaceCandidate, 0, len(interfaces))
	for _, iface := range interfaces {
		peers := append([]model.BGPPeer(nil), peersByInterface[iface.Name]...)
		candidates = append(candidates, vpnSourceInterfaceCandidate{
			RouterName:    base.RouterName,
			RouterASN:     base.RouterASN,
			InterfaceName: iface.Name,
			InterfaceIP:   preferredInterfaceIP(iface, peers),
			Peers:         peers,
		})
	}
	return candidates
}

func vpnItemsForSourceCandidates(base model.MappingItem, candidates []vpnSourceInterfaceCandidate) []model.MappingItem {
	items := make([]model.MappingItem, 0, len(candidates))
	for _, candidate := range candidates {
		item := base
		applyVPNSourceCandidate(&item, candidate)
		items = append(items, item)
	}
	return items
}

func applyVPNSourceCandidate(item *model.MappingItem, candidate vpnSourceInterfaceCandidate) {
	item.SrcCloudRouter = candidate.RouterName
	item.SrcCloudRouterASN = candidate.RouterASN
	item.SrcCloudRouterInterface = candidate.InterfaceName
	item.SrcCloudRouterInterfaceIP = candidate.InterfaceIP
}

func applyVPNDestinationCandidate(item *model.MappingItem, candidate vpnDestinationInterfaceCandidate) {
	item.DstRegion = firstNonEmpty(candidate.Region, item.DstRegion)
	item.DstVPC = firstNonEmpty(candidate.VPC, item.DstVPC)
	item.DstVPNGateway = firstNonEmpty(candidate.GatewayName, item.DstVPNGateway)
	item.DstVPNGatewayType = firstNonEmpty(candidate.GatewayType, item.DstVPNGatewayType)
	item.DstVPNTunnel = candidate.TunnelName
	item.DstVPNTunnelStatus = candidate.TunnelStatus
	item.DstCloudRouter = candidate.RouterName
	item.DstCloudRouterASN = candidate.RouterASN
	item.DstCloudRouterInterface = candidate.InterfaceName
	item.DstCloudRouterInterfaceIP = candidate.InterfaceIP
}

func vpnDestinationInterfaceCandidates(sourceTunnel model.VPNTunnel, sourceGateway, destinationGateway model.VPNGateway, destinationData vpnProjectData) []vpnDestinationInterfaceCandidate {
	tunnels := matchingDestinationVPNTunnels(sourceTunnel, sourceGateway, destinationGateway, destinationData.Tunnels)
	if len(tunnels) == 0 {
		return nil
	}

	var candidates []vpnDestinationInterfaceCandidate
	for _, tunnel := range tunnels {
		router := destinationData.RouterByKey[routerKey(tunnel.Region, tunnel.Router)]
		base := vpnDestinationInterfaceCandidate{
			Region:       firstNonEmpty(tunnel.Region, destinationGateway.Region),
			VPC:          firstNonEmpty(router.Network, destinationGateway.Network),
			GatewayName:  firstNonEmpty(destinationGateway.Name, tunnel.VPNGateway),
			GatewayType:  firstNonEmpty(destinationGateway.Type),
			TunnelName:   tunnel.Name,
			TunnelStatus: tunnel.Status,
			RouterName:   firstNonEmpty(router.Name, tunnel.Router),
			RouterASN:    router.ASN,
		}

		interfaces := interfacesForTunnel(router, tunnel.Name)
		if len(interfaces) == 0 {
			candidates = append(candidates, base)
			continue
		}

		sort.Slice(interfaces, func(i, j int) bool {
			return interfaces[i].Name < interfaces[j].Name
		})
		peersByInterface := peersForRouter(router, destinationData.Statuses[routerKey(router.Region, router.Name)])
		for _, iface := range interfaces {
			peers := append([]model.BGPPeer(nil), peersByInterface[iface.Name]...)
			candidates = append(candidates, vpnDestinationInterfaceCandidate{
				Region:        base.Region,
				VPC:           base.VPC,
				GatewayName:   base.GatewayName,
				GatewayType:   base.GatewayType,
				TunnelName:    base.TunnelName,
				TunnelStatus:  base.TunnelStatus,
				RouterName:    base.RouterName,
				RouterASN:     base.RouterASN,
				InterfaceName: iface.Name,
				InterfaceIP:   preferredInterfaceIP(iface, peers),
				Peers:         peers,
			})
		}
	}
	return candidates
}

func matchingDestinationVPNTunnels(sourceTunnel model.VPNTunnel, sourceGateway, destinationGateway model.VPNGateway, destinationTunnels []model.VPNTunnel) []model.VPNTunnel {
	var candidates []model.VPNTunnel
	for _, tunnel := range destinationTunnels {
		if strings.TrimSpace(destinationGateway.Name) != "" && strings.TrimSpace(tunnel.VPNGateway) != "" && tunnel.VPNGateway != destinationGateway.Name {
			continue
		}
		if strings.TrimSpace(sourceGateway.SelfLink) != "" && strings.TrimSpace(tunnel.PeerGCPGateway) != "" && strings.TrimSpace(tunnel.PeerGCPGateway) != strings.TrimSpace(sourceGateway.SelfLink) {
			continue
		}
		if strings.TrimSpace(sourceTunnel.Region) != "" && strings.TrimSpace(destinationGateway.Region) != "" && strings.TrimSpace(tunnel.Region) != "" && tunnel.Region != destinationGateway.Region {
			continue
		}
		candidates = append(candidates, tunnel)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Region != candidates[j].Region {
			return candidates[i].Region < candidates[j].Region
		}
		if candidates[i].Name != candidates[j].Name {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].VPNGatewayInterface < candidates[j].VPNGatewayInterface
	})
	return candidates
}

func chooseVPNDestinationCandidate(source vpnSourceInterfaceCandidate, destinations []vpnDestinationInterfaceCandidate) (int, bool) {
	if len(destinations) == 0 {
		return -1, false
	}

	bestIndex := -1
	bestScore := -1
	tied := false
	for idx, destination := range destinations {
		score := vpnPairScore(source, destination)
		if score > bestScore {
			bestIndex = idx
			bestScore = score
			tied = false
			continue
		}
		if score == bestScore {
			tied = true
		}
	}

	switch {
	case bestScore >= 4 && !tied:
		return bestIndex, true
	case bestScore >= 2 && !tied:
		return bestIndex, true
	case bestScore == 0 && len(destinations) == 1:
		return 0, true
	default:
		return -1, false
	}
}

func vpnPairScore(source vpnSourceInterfaceCandidate, destination vpnDestinationInterfaceCandidate) int {
	score := 0
	if anySharedString(vpnRemoteIPs(source.Peers), vpnLocalIPs(destination.InterfaceIP, destination.Peers)) {
		score += 4
	}
	if anySharedString(vpnRemoteIPs(destination.Peers), vpnLocalIPs(source.InterfaceIP, source.Peers)) {
		score += 4
	}
	if containsString(vpnPeerASNs(source.Peers), destination.RouterASN) {
		score += 2
	}
	if containsString(vpnPeerASNs(destination.Peers), source.RouterASN) {
		score += 2
	}
	return score
}

func vpnPairStatus(source vpnSourceInterfaceCandidate, destination vpnDestinationInterfaceCandidate) string {
	return firstNonEmpty(firstPeerSessionState(destination.Peers), firstPeerSessionState(source.Peers), "unknown")
}

func sharedVPNDestinationContext(candidates []vpnDestinationInterfaceCandidate) (vpnDestinationInterfaceCandidate, bool) {
	if len(candidates) == 0 {
		return vpnDestinationInterfaceCandidate{}, false
	}
	shared := candidates[0]
	shared.InterfaceName = ""
	shared.InterfaceIP = ""
	for _, candidate := range candidates[1:] {
		if candidate.Region != shared.Region ||
			candidate.VPC != shared.VPC ||
			candidate.GatewayName != shared.GatewayName ||
			candidate.GatewayType != shared.GatewayType ||
			candidate.TunnelName != shared.TunnelName ||
			candidate.TunnelStatus != shared.TunnelStatus ||
			candidate.RouterName != shared.RouterName ||
			candidate.RouterASN != shared.RouterASN {
			return vpnDestinationInterfaceCandidate{}, false
		}
	}
	return shared, true
}

func preferredInterfaceIP(iface model.RouterInterface, peers []model.BGPPeer) string {
	for _, peer := range peers {
		if strings.TrimSpace(peer.LocalIP) != "" {
			return strings.TrimSpace(peer.LocalIP)
		}
	}
	return iface.IPRange
}

func vpnLocalIPs(interfaceIP string, peers []model.BGPPeer) []string {
	values := make([]string, 0, len(peers)+1)
	if normalized := normalizeIPAddress(interfaceIP); normalized != "" {
		values = append(values, normalized)
	}
	for _, peer := range peers {
		if normalized := normalizeIPAddress(peer.LocalIP); normalized != "" {
			values = append(values, normalized)
		}
	}
	return uniqueStrings(values)
}

func vpnRemoteIPs(peers []model.BGPPeer) []string {
	values := make([]string, 0, len(peers))
	for _, peer := range peers {
		if normalized := normalizeIPAddress(peer.RemoteIP); normalized != "" {
			values = append(values, normalized)
		}
	}
	return uniqueStrings(values)
}

func vpnPeerASNs(peers []model.BGPPeer) []string {
	values := make([]string, 0, len(peers))
	for _, peer := range peers {
		if trimmed := strings.TrimSpace(peer.PeerASN); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return uniqueStrings(values)
}

func firstPeerSessionState(peers []model.BGPPeer) string {
	for _, peer := range peers {
		if strings.TrimSpace(peer.SessionState) != "" {
			return strings.TrimSpace(peer.SessionState)
		}
	}
	return ""
}

func normalizeIPAddress(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if slash := strings.IndexRune(value, '/'); slash >= 0 {
		return strings.TrimSpace(value[:slash])
	}
	return value
}

func anySharedString(left, right []string) bool {
	if len(left) == 0 || len(right) == 0 {
		return false
	}
	seen := make(map[string]struct{}, len(left))
	for _, value := range left {
		seen[value] = struct{}{}
	}
	for _, value := range right {
		if _, ok := seen[value]; ok {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
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

func interfacesForTunnel(router model.CloudRouter, tunnel string) []model.RouterInterface {
	var result []model.RouterInterface
	for _, iface := range router.Interfaces {
		if iface.LinkedVPNTunnel == tunnel {
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
	var base string
	switch report.Type {
	case TypeVPN:
		base = vpnOutputBaseName(opts, timestamp)
	default:
		target := report.DestinationProject
		if strings.TrimSpace(target) == "" {
			target = opts.Org + "-all"
		}
		base = fmt.Sprintf("netmap-interconnect-%s-to-%s-%s", opts.SourceProject, target, timestamp)
	}
	if format == render.FormatJSON {
		return base + ".json"
	}
	if format == render.FormatTree {
		return base + ".tree.txt"
	}
	return base + "." + ext
}

func vpnOutputBaseName(opts Options, timestamp string) string {
	org := statusValueOrAll(opts.Org)
	workload := strings.TrimSpace(opts.Workload)
	environment := strings.TrimSpace(opts.Environment)
	switch {
	case workload == "" && environment == "":
		return fmt.Sprintf("netmap-vpn-%s-all-%s", org, timestamp)
	case workload == "":
		return fmt.Sprintf("netmap-vpn-%s-all-%s-%s", org, environment, timestamp)
	case environment == "":
		return fmt.Sprintf("netmap-vpn-%s-%s-all-%s", org, workload, timestamp)
	default:
		return fmt.Sprintf("netmap-vpn-%s-%s-%s-%s", org, workload, environment, timestamp)
	}
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

func sortMappingItems(items []model.MappingItem) {
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
		if items[i].SrcRegion != items[j].SrcRegion {
			return items[i].SrcRegion < items[j].SrcRegion
		}
		if items[i].SrcInterconnect != items[j].SrcInterconnect {
			return items[i].SrcInterconnect < items[j].SrcInterconnect
		}
		if items[i].SrcVPNGateway != items[j].SrcVPNGateway {
			return items[i].SrcVPNGateway < items[j].SrcVPNGateway
		}
		if items[i].SrcVPNTunnel != items[j].SrcVPNTunnel {
			return items[i].SrcVPNTunnel < items[j].SrcVPNTunnel
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
		if items[i].DstVPNGateway != items[j].DstVPNGateway {
			return items[i].DstVPNGateway < items[j].DstVPNGateway
		}
		if items[i].DstVPNTunnel != items[j].DstVPNTunnel {
			return items[i].DstVPNTunnel < items[j].DstVPNTunnel
		}
		if items[i].DstCloudRouter != items[j].DstCloudRouter {
			return items[i].DstCloudRouter < items[j].DstCloudRouter
		}
		if items[i].DstCloudRouterInterface != items[j].DstCloudRouterInterface {
			return items[i].DstCloudRouterInterface < items[j].DstCloudRouterInterface
		}
		return items[i].RemoteBGPPeer < items[j].RemoteBGPPeer
	})
}

func vpnGatewayKey(kind, name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	return kind + "\x00" + name
}

func vpnTunnelGatewayKey(tunnel model.VPNTunnel) string {
	if strings.TrimSpace(tunnel.VPNGateway) != "" {
		return vpnGatewayKey("ha", tunnel.VPNGateway)
	}
	if strings.TrimSpace(tunnel.TargetVPNGateway) != "" {
		return vpnGatewayKey("classic", tunnel.TargetVPNGateway)
	}
	return ""
}

func resourceNameFromURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(strings.TrimRight(value, "/"), "/")
	return parts[len(parts)-1]
}

func projectIDFromResourceURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(strings.Trim(value, "/"), "/")
	for idx := 0; idx < len(parts)-1; idx++ {
		if parts[idx] == "projects" {
			return parts[idx+1]
		}
	}
	return ""
}

func taskLabel(target config.ResolvedTarget) string {
	return fmt.Sprintf(
		"org=%s workload=%s environment=%s project=%s",
		target.Org,
		target.Workload,
		target.Environment,
		target.ProjectID,
	)
}

type taskState int

const (
	taskStateRunning taskState = iota
	taskStateCompleted
	taskStateFailed
)

type taskRow struct {
	Label     string
	State     taskState
	StartedAt time.Time
	Elapsed   time.Duration
}

type statusTable struct {
	writer   io.Writer
	now      func() time.Time
	interval time.Duration
	mu       sync.Mutex
	stopCh   chan struct{}
	doneCh   chan struct{}
	frameIdx int
	rows     []taskRow
	active   int
	summary  []string
	lines    int
	stopped  bool
}

func newStatusTable(writer io.Writer, now func() time.Time) *statusTable {
	return &statusTable{
		writer:   writer,
		now:      now,
		interval: 100 * time.Millisecond,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
		active:   -1,
	}
}

func (t *statusTable) Start() {
	if t == nil || t.writer == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		defer close(t.doneCh)

		for {
			select {
			case <-ticker.C:
				t.mu.Lock()
				if t.stopped {
					t.mu.Unlock()
					return
				}
				if t.active >= 0 {
					t.frameIdx = (t.frameIdx + 1) % len(brailleSpinnerFrames)
					t.renderLocked()
				}
				t.mu.Unlock()
			case <-t.stopCh:
				return
			}
		}
	}()
}

func (t *statusTable) Stop() {
	if t == nil || t.writer == nil {
		return
	}

	t.mu.Lock()
	if t.stopped {
		t.mu.Unlock()
		return
	}
	t.stopped = true
	close(t.stopCh)
	t.mu.Unlock()

	<-t.doneCh
}

func (t *statusTable) StartTask(label string) {
	if t == nil || t.writer == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}
	t.rows = append(t.rows, taskRow{
		Label:     label,
		State:     taskStateRunning,
		StartedAt: t.now().UTC(),
	})
	t.active = len(t.rows) - 1
	t.frameIdx = 0
	t.renderLocked()
}

func (t *statusTable) CompleteTask(label string, elapsed time.Duration) {
	t.finishTask(taskStateCompleted, label, elapsed)
}

func (t *statusTable) FailTask(label string, elapsed time.Duration) {
	t.finishTask(taskStateFailed, label, elapsed)
}

func (t *statusTable) finishTask(state taskState, label string, elapsed time.Duration) {
	if t == nil || t.writer == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped || t.active < 0 || t.active >= len(t.rows) {
		return
	}
	t.rows[t.active].State = state
	t.rows[t.active].Label = label
	t.rows[t.active].Elapsed = elapsed
	t.active = -1
	t.renderLocked()
}

func (t *statusTable) Finish(outputPath string) {
	if t == nil || t.writer == nil {
		return
	}
	t.mu.Lock()
	total := t.totalDurationLocked()
	t.summary = []string{
		"Output: " + outputPath,
		"Total Time: " + formatStatusDuration(total),
	}
	t.renderLocked()
	t.mu.Unlock()
	t.Stop()
}

func (t *statusTable) totalDurationLocked() time.Duration {
	var total time.Duration
	for _, row := range t.rows {
		if row.State == taskStateCompleted {
			total += row.Elapsed
		}
	}
	return total
}

func (t *statusTable) renderLocked() {
	lines := t.buildLinesLocked()
	if len(lines) == 0 {
		return
	}
	if t.lines > 0 {
		fmt.Fprintf(t.writer, "\x1b[%dA", t.lines)
	}
	for _, line := range lines {
		fmt.Fprintf(t.writer, "\r\x1b[2K%s\n", line)
	}
	t.lines = len(lines)
}

func (t *statusTable) buildLinesLocked() []string {
	if len(t.rows) == 0 && len(t.summary) == 0 {
		return nil
	}

	col1Width := len("Task")
	col2Width := len("Time")
	for idx, row := range t.rows {
		label := t.renderLabelLocked(idx, row)
		if width := utf8.RuneCountInString(label); width > col1Width {
			col1Width = width
		}
		timer := t.renderTimerLocked(idx, row)
		if width := utf8.RuneCountInString(timer); width > col2Width {
			col2Width = width
		}
	}

	mergedWidth := col1Width + col2Width + 3
	twoColBorder := fmt.Sprintf("+-%s-+-%s-+", strings.Repeat("-", col1Width), strings.Repeat("-", col2Width))
	mergedBorder := fmt.Sprintf("+-%s-+", strings.Repeat("-", mergedWidth))

	lines := []string{twoColBorder}
	for idx, row := range t.rows {
		lines = append(lines, fmt.Sprintf("| %-*s | %*s |", col1Width, t.renderLabelLocked(idx, row), col2Width, t.renderTimerLocked(idx, row)))
	}
	if len(t.summary) == 0 {
		lines = append(lines, twoColBorder)
		return lines
	}
	lines = append(lines, mergedBorder)
	for _, line := range t.summary {
		lines = append(lines, fmt.Sprintf("| %-*s |", mergedWidth, line))
	}
	lines = append(lines, mergedBorder)
	return lines
}

func (t *statusTable) renderLabelLocked(idx int, row taskRow) string {
	switch row.State {
	case taskStateCompleted:
		return "✅ Completed " + row.Label
	case taskStateFailed:
		return "❌ Failed " + row.Label
	default:
		frame := brailleSpinnerFrames[t.frameIdx%len(brailleSpinnerFrames)]
		if idx != t.active {
			frame = brailleSpinnerFrames[0]
		}
		return frame + " Running " + row.Label
	}
}

func (t *statusTable) renderTimerLocked(idx int, row taskRow) string {
	switch row.State {
	case taskStateCompleted, taskStateFailed:
		return formatStatusDuration(row.Elapsed)
	default:
		if idx != t.active {
			return formatStatusDuration(0)
		}
		return formatStatusDuration(t.now().UTC().Sub(row.StartedAt))
	}
}

func formatStatusDuration(duration time.Duration) string {
	if duration < 0 {
		duration = 0
	}
	switch {
	case duration < time.Second:
		return duration.Truncate(time.Millisecond).String()
	case duration < 10*time.Second:
		return duration.Truncate(100 * time.Millisecond).String()
	default:
		return duration.Truncate(time.Second).String()
	}
}
