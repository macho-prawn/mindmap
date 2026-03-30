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

	a.startStatusTable()
	defer a.stopStatusTable()

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

	a.finishStatusTable(outputPath)
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
  Stderr shows an ASCII 2-column task table with a Braille spinner on active rows.
  Completed rows use a tick marker and print per-task elapsed time.
  The final merged row prints Output: <path> and Total Time: <duration>.
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
