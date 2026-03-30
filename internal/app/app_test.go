package app

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"netmap/internal/model"
)

type memoryFileStore struct {
	files map[string][]byte
}

func (m *memoryFileStore) ReadFile(name string) ([]byte, error) {
	data, ok := m.files[name]
	if !ok {
		return nil, errors.New("missing file")
	}
	return data, nil
}

func (m *memoryFileStore) WriteFile(name string, data []byte) error {
	m.files[name] = data
	return nil
}

type mockProvider struct {
	interconnects          []model.DedicatedInterconnect
	attachments            []model.VLANAttachment
	routers                []model.CloudRouter
	statuses               map[string]model.RouterStatus
	attachmentsByProject   map[string][]model.VLANAttachment
	routersByProject       map[string][]model.CloudRouter
	statusesByProjectRoute map[string]model.RouterStatus
	attachmentCalls        map[string]int
	routerCalls            map[string]int
	statusCalls            map[string]int
}

func (m mockProvider) ListDedicatedInterconnects(context.Context, string) ([]model.DedicatedInterconnect, error) {
	return m.interconnects, nil
}

func (m mockProvider) ListVLANAttachments(_ context.Context, project string) ([]model.VLANAttachment, error) {
	if m.attachmentCalls != nil {
		m.attachmentCalls[project]++
	}
	if len(m.attachmentsByProject) > 0 {
		return m.attachmentsByProject[project], nil
	}
	return m.attachments, nil
}

func (m mockProvider) ListCloudRouters(_ context.Context, project string) ([]model.CloudRouter, error) {
	if m.routerCalls != nil {
		m.routerCalls[project]++
	}
	if len(m.routersByProject) > 0 {
		return m.routersByProject[project], nil
	}
	return m.routers, nil
}

func (m mockProvider) GetCloudRouterStatus(_ context.Context, project, region, router string) (model.RouterStatus, error) {
	if m.statusCalls != nil {
		m.statusCalls[project+"/"+region+"/"+router]++
	}
	if len(m.statusesByProjectRoute) > 0 {
		return m.statusesByProjectRoute[project+"/"+region+"/"+router], nil
	}
	return m.statuses[region+"/"+router], nil
}

func TestParseOptionsValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing t", args: []string{}, want: "missing mandatory parameter -t"},
		{name: "invalid t", args: []string{"-t", "bad", "-o", "dbc"}, want: "invalid -t value"},
		{name: "missing o", args: []string{"-t", "interconnect", "-p", "src"}, want: "missing mandatory parameter -o"},
		{name: "missing p for interconnect", args: []string{"-t", "interconnect", "-o", "dbc"}, want: "missing mandatory parameter -p"},
		{name: "forbid p for vpn", args: []string{"-t", "vpn", "-o", "dbc", "-p", "src"}, want: "-p must not be used"},
		{name: "invalid format", args: []string{"-t", "interconnect", "-o", "dbc", "-p", "src", "-f", "xml"}, want: "invalid -f value"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseOptions(tc.args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestParseOptionsAllowsOptionalWorkloadAndEnv(t *testing.T) {
	opts, err := ParseOptions([]string{"-t", "interconnect", "-o", "dbc", "-p", "src-project"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if opts.Workload != "" || opts.Environment != "" {
		t.Fatalf("expected optional selectors to be empty, got %+v", opts)
	}
}

func TestParseOptionsHelp(t *testing.T) {
	opts, err := ParseOptions([]string{"-h"})
	if err != nil {
		t.Fatalf("parse help: %v", err)
	}
	if !opts.ShowHelp || !strings.Contains(opts.Usage, "Usage:") {
		t.Fatalf("expected help usage, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "Selector Expansion:") {
		t.Fatalf("expected selector expansion guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "-o + -e        expands all workloads containing that environment") {
		t.Fatalf("expected explicit -o + -e help text, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "Omit -f to write Mermaid output by default.") {
		t.Fatalf("expected default mermaid guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "Mermaid output can be viewed in https://mermaid.live") {
		t.Fatalf("expected mermaid.live guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "Output: <path>") || !strings.Contains(opts.Usage, "Total Time: <duration>") {
		t.Fatalf("expected final summary guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "2-column task table") {
		t.Fatalf("expected task table guidance, got %+v", opts)
	}
}

func TestRunWritesMermaidByDefault(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}
	app, err := New(store, mockProvider{
		interconnects: []model.DedicatedInterconnect{{
			Name:          "ic-1",
			State:         "ACTIVE",
			MacsecEnabled: true,
			MacsecKeyName: "macsec-key-a",
		}},
		attachments: []model.VLANAttachment{{
			Name:         "attachment-1",
			Region:       "us-central1",
			Network:      "vpc-a",
			State:        "ACTIVE",
			Interconnect: "ic-1",
			Router:       "router-1",
		}},
		routers: []model.CloudRouter{{
			Name:   "router-1",
			Region: "us-central1",
			ASN:    "64512",
			Interfaces: []model.RouterInterface{{
				Name:                     "if-1",
				LinkedInterconnectAttach: "attachment-1",
				IPRange:                  "169.254.1.1/30",
			}},
			BGPPeers: []model.BGPPeer{{
				Name:         "peer-1",
				Interface:    "if-1",
				LocalIP:      "169.254.1.1",
				RemoteIP:     "169.254.1.2",
				PeerASN:      "64550",
				SessionState: "UP",
			}},
		}},
		statuses: map[string]model.RouterStatus{
			"us-central1/router-1": {
				RouterName: "router-1",
				Region:     "us-central1",
				Peers: []model.BGPPeerStatus{{
					Name:         "peer-1",
					LocalIP:      "169.254.1.1",
					RemoteIP:     "169.254.1.2",
					SessionState: "UP",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	app.now = func() time.Time {
		return time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC)
	}
	var status bytes.Buffer
	app.status = &status

	err = app.Run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-w", "native",
		"-e", "dev",
		"-p", "src-project",
	})
	if err != nil {
		t.Fatalf("run app: %v", err)
	}

	data, ok := store.files["netmap-interconnect-src-project-to-project-20260328T000000Z.mmd"]
	if !ok {
		t.Fatalf("expected mermaid output file to be written")
	}
	content := string(data)
	if !strings.Contains(content, "flowchart LR") || !strings.Contains(content, "remote_bgp_peer: peer-1") || !strings.Contains(content, "dst_cloud_router_interface: if-1") {
		t.Fatalf("unexpected mermaid content: %s", content)
	}
	if !strings.Contains(content, "<br>") || strings.Contains(content, "\\n") {
		t.Fatalf("expected mermaid.live-compatible line breaks, got: %s", content)
	}
	if !strings.Contains(content, "src_macsec_enabled: true") || !strings.Contains(content, "src_macsec_keyname: macsec-key-a") {
		t.Fatalf("unexpected mermaid content: %s", content)
	}
	if !strings.Contains(content, "dst_cloud_router_asn: 64512") {
		t.Fatalf("unexpected mermaid content: %s", content)
	}
	if !strings.Contains(content, "dst_vpc: vpc-a") {
		t.Fatalf("unexpected mermaid content: %s", content)
	}
	if !strings.Contains(content, "remote_bgp_peer_asn: 64550") {
		t.Fatalf("unexpected mermaid content: %s", content)
	}
	if !strings.Contains(content, "bgp_peering_status: UP") {
		t.Fatalf("expected dedicated bgp status node in mermaid output: %s", content)
	}
	statusOutput := status.String()
	if !containsBrailleSpinner(statusOutput) || !strings.Contains(statusOutput, "Running org=dbc workload=native environment=dev project=project") {
		t.Fatalf("expected running task row, got: %s", statusOutput)
	}
	if strings.Contains(statusOutput, "⏳") {
		t.Fatalf("unexpected hourglass status output, got: %s", statusOutput)
	}
	if !strings.Contains(statusOutput, "✅ Completed org=dbc workload=native environment=dev project=project") {
		t.Fatalf("expected completed task row, got: %s", statusOutput)
	}
	if !strings.Contains(statusOutput, "Output: netmap-interconnect-src-project-to-project-20260328T000000Z.mmd") || !strings.Contains(statusOutput, "Total Time: 0s") {
		t.Fatalf("expected final summary row, got: %s", statusOutput)
	}
	if !containsTaskTable(statusOutput) {
		t.Fatalf("expected ascii task table, got: %s", statusOutput)
	}
}

func TestRunSuppressesMermaidWhenFormatProvided(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}
	app, err := New(store, mockProvider{
		interconnects: []model.DedicatedInterconnect{{
			Name:          "ic-1",
			State:         "ACTIVE",
			MacsecEnabled: true,
			MacsecKeyName: "macsec-key-a",
		}},
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	app.now = func() time.Time {
		return time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC)
	}
	var status bytes.Buffer
	app.status = &status

	err = app.Run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-w", "native",
		"-e", "dev",
		"-p", "src-project",
		"-f", "json",
	})
	if err != nil {
		t.Fatalf("run app: %v", err)
	}

	if _, ok := store.files["netmap-interconnect-src-project-to-project-20260328T000000Z.mmd"]; ok {
		t.Fatalf("unexpected mermaid output")
	}
	if _, ok := store.files["netmap-interconnect-src-project-to-project-20260328T000000Z.json"]; !ok {
		t.Fatalf("expected json output")
	}
	if !strings.Contains(status.String(), "Output: netmap-interconnect-src-project-to-project-20260328T000000Z.json") || !strings.Contains(status.String(), "Total Time: 0s") {
		t.Fatalf("expected final summary row, got: %s", status.String())
	}
}

func TestRunWithOrgFanoutWritesCombinedOutput(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(fanoutConfig),
		},
	}
	app, err := New(store, mockProvider{
		interconnects: []model.DedicatedInterconnect{{Name: "ic-1", State: "ACTIVE", MacsecEnabled: true, MacsecKeyName: "fanout-key"}},
		attachmentsByProject: map[string][]model.VLANAttachment{
			"project-a": {{
				Name:         "attachment-a",
				Region:       "us-central1",
				Network:      "vpc-a",
				State:        "ACTIVE",
				Interconnect: "ic-1",
				Router:       "router-a",
			}},
			"project-b": {{
				Name:         "attachment-b",
				Region:       "europe-west1",
				Network:      "vpc-b",
				State:        "ACTIVE",
				Interconnect: "ic-1",
				Router:       "router-b",
			}},
		},
		routersByProject: map[string][]model.CloudRouter{
			"project-a": {{
				Name:   "router-a",
				Region: "us-central1",
				ASN:    "64520",
				Interfaces: []model.RouterInterface{{
					Name:                     "if-a",
					LinkedInterconnectAttach: "attachment-a",
					IPRange:                  "169.254.10.1/30",
				}},
				BGPPeers: []model.BGPPeer{{
					Name:         "peer-a",
					Interface:    "if-a",
					LocalIP:      "169.254.10.1",
					RemoteIP:     "169.254.10.2",
					PeerASN:      "64561",
					SessionState: "UP",
				}},
			}},
			"project-b": {{
				Name:   "router-b",
				Region: "europe-west1",
				ASN:    "64521",
				Interfaces: []model.RouterInterface{{
					Name:                     "if-b",
					LinkedInterconnectAttach: "attachment-b",
					IPRange:                  "169.254.20.1/30",
				}},
				BGPPeers: []model.BGPPeer{{
					Name:         "peer-b",
					Interface:    "if-b",
					LocalIP:      "169.254.20.1",
					RemoteIP:     "169.254.20.2",
					PeerASN:      "64562",
					SessionState: "UP",
				}},
			}},
		},
		statusesByProjectRoute: map[string]model.RouterStatus{
			"project-a/us-central1/router-a": {
				RouterName: "router-a",
				Region:     "us-central1",
				Peers: []model.BGPPeerStatus{{
					Name:         "peer-a",
					LocalIP:      "169.254.10.1",
					RemoteIP:     "169.254.10.2",
					SessionState: "UP",
				}},
			},
			"project-b/europe-west1/router-b": {
				RouterName: "router-b",
				Region:     "europe-west1",
				Peers: []model.BGPPeerStatus{{
					Name:         "peer-b",
					LocalIP:      "169.254.20.1",
					RemoteIP:     "169.254.20.2",
					SessionState: "UP",
				}},
			},
		},
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	app.now = func() time.Time {
		return time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC)
	}
	var status bytes.Buffer
	app.status = &status

	err = app.Run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-p", "src-project",
		"-f", "tree",
	})
	if err != nil {
		t.Fatalf("run app: %v", err)
	}

	data, ok := store.files["netmap-interconnect-src-project-to-dbc-all-20260328T000000Z.tree.txt"]
	if !ok {
		t.Fatalf("expected combined output file to be written")
	}
	content := string(data)
	if !strings.Contains(content, "workload: native") || !strings.Contains(content, "environment: dev") || !strings.Contains(content, "environment: prod") || !strings.Contains(content, "project-a") || !strings.Contains(content, "project-b") {
		t.Fatalf("expected fanout destinations in tree output, got: %s", content)
	}
	statusOutput := status.String()
	if !strings.Contains(statusOutput, "✅ Completed org=dbc workload=native environment=dev project=project-a") {
		t.Fatalf("expected dev completion status, got: %s", statusOutput)
	}
	if !strings.Contains(statusOutput, "✅ Completed org=dbc workload=native environment=prod project=project-b") {
		t.Fatalf("expected prod completion status, got: %s", statusOutput)
	}
	if !strings.Contains(statusOutput, "Output: netmap-interconnect-src-project-to-dbc-all-20260328T000000Z.tree.txt") || !strings.Contains(statusOutput, "Total Time: 0s") {
		t.Fatalf("expected final summary row, got: %s", statusOutput)
	}
}

func TestVPNNotImplemented(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}
	app, err := New(store, mockProvider{})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	err = app.Run(context.Background(), []string{
		"-t", "vpn",
		"-o", "dbc",
		"-w", "native",
		"-e", "dev",
	})
	if err == nil || !strings.Contains(err.Error(), "vpn is not implemented yet") {
		t.Fatalf("expected vpn not implemented error, got %v", err)
	}
}

func TestBuildMappingItemsIncludesGlobalSrcRegionAndUnmapped(t *testing.T) {
	items := buildMappingItems(
		"src-project",
		"dst-project",
		[]model.DedicatedInterconnect{
			{Name: "mapped", State: "ACTIVE"},
			{Name: "unmapped", State: "DOWN", MacsecEnabled: true, MacsecKeyName: "macsec-key-unmapped"},
		},
		[]model.VLANAttachment{{
			Name:         "attachment-1",
			Region:       "europe-west1",
			Network:      "vpc-a",
			State:        "ACTIVE",
			Interconnect: "mapped",
			Router:       "router-1",
		}},
		[]model.CloudRouter{{
			Name:   "router-1",
			Region: "europe-west1",
			ASN:    "64530",
			Interfaces: []model.RouterInterface{{
				Name:                     "if-1",
				LinkedInterconnectAttach: "attachment-1",
				IPRange:                  "169.254.10.1/30",
			}},
		}},
		map[string]model.RouterStatus{},
	)

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].SrcRegion != "global" && items[1].SrcRegion != "global" {
		t.Fatalf("expected src_region=global in all items")
	}
	foundMacsec := false
	foundRouterASN := false
	foundVPC := false
	foundUnmapped := false
	for _, item := range items {
		if item.SrcInterconnect == "mapped" && item.DstCloudRouterASN == "64530" {
			foundRouterASN = true
		}
		if item.SrcInterconnect == "mapped" && item.DstVPC == "vpc-a" {
			foundVPC = true
		}
		if item.SrcInterconnect == "unmapped" && item.SrcMacsecEnabled && item.SrcMacsecKeyName == "macsec-key-unmapped" {
			foundMacsec = true
		}
		if item.SrcInterconnect == "unmapped" && !item.Mapped {
			foundUnmapped = true
		}
	}
	if !foundMacsec {
		t.Fatalf("expected source macsec fields to propagate")
	}
	if !foundRouterASN {
		t.Fatalf("expected router asn to propagate")
	}
	if !foundVPC {
		t.Fatalf("expected vpc to propagate")
	}
	if !foundUnmapped {
		t.Fatalf("expected unmapped interconnect item")
	}
}

func TestRunWithDuplicateProjectFanoutCachesDiscoveryAndLogsEachTuple(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(duplicateProjectConfig),
		},
	}
	provider := mockProvider{
		interconnects: []model.DedicatedInterconnect{{Name: "ic-1", State: "ACTIVE", MacsecEnabled: true, MacsecKeyName: "shared-key"}},
		attachmentsByProject: map[string][]model.VLANAttachment{
			"shared-project": {{
				Name:         "attachment-shared",
				Region:       "us-central1",
				Network:      "shared-vpc",
				State:        "ACTIVE",
				Interconnect: "ic-1",
				Router:       "router-shared",
			}},
		},
		routersByProject: map[string][]model.CloudRouter{
			"shared-project": {{
				Name:   "router-shared",
				Region: "us-central1",
				ASN:    "64540",
				Interfaces: []model.RouterInterface{{
					Name:                     "if-shared",
					LinkedInterconnectAttach: "attachment-shared",
					IPRange:                  "169.254.30.1/30",
				}},
				BGPPeers: []model.BGPPeer{{
					Name:         "peer-shared",
					Interface:    "if-shared",
					LocalIP:      "169.254.30.1",
					RemoteIP:     "169.254.30.2",
					PeerASN:      "64560",
					SessionState: "UP",
				}},
			}},
		},
		statusesByProjectRoute: map[string]model.RouterStatus{
			"shared-project/us-central1/router-shared": {
				RouterName: "router-shared",
				Region:     "us-central1",
				Peers: []model.BGPPeerStatus{{
					Name:         "peer-shared",
					LocalIP:      "169.254.30.1",
					RemoteIP:     "169.254.30.2",
					SessionState: "UP",
				}},
			},
		},
		attachmentCalls: map[string]int{},
		routerCalls:     map[string]int{},
		statusCalls:     map[string]int{},
	}
	app, err := New(store, provider)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	app.now = func() time.Time {
		return time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC)
	}
	var status bytes.Buffer
	app.status = &status

	err = app.Run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-e", "dev",
		"-p", "src-project",
		"-f", "csv",
	})
	if err != nil {
		t.Fatalf("run app: %v", err)
	}

	if provider.attachmentCalls["shared-project"] != 1 {
		t.Fatalf("expected one attachment discovery call, got %d", provider.attachmentCalls["shared-project"])
	}
	if provider.routerCalls["shared-project"] != 1 {
		t.Fatalf("expected one router discovery call, got %d", provider.routerCalls["shared-project"])
	}
	if provider.statusCalls["shared-project/us-central1/router-shared"] != 1 {
		t.Fatalf("expected one router status call, got %d", provider.statusCalls["shared-project/us-central1/router-shared"])
	}

	statusOutput := status.String()
	if !strings.Contains(statusOutput, "✅ Completed org=dbc workload=native environment=dev project=shared-project") {
		t.Fatalf("expected native/dev completion status, got: %s", statusOutput)
	}
	if !strings.Contains(statusOutput, "✅ Completed org=dbc workload=platform environment=dev project=shared-project") {
		t.Fatalf("expected platform/dev completion status, got: %s", statusOutput)
	}
	if !strings.Contains(statusOutput, "Output: netmap-interconnect-src-project-to-shared-project-20260328T000000Z.csv") || !strings.Contains(statusOutput, "Total Time: 0s") {
		t.Fatalf("expected final summary row, got: %s", statusOutput)
	}

	data := string(store.files["netmap-interconnect-src-project-to-shared-project-20260328T000000Z.csv"])
	if count := strings.Count(data, "dbc,native,dev,src-project"); count != 1 {
		t.Fatalf("expected one native/dev csv branch, got %d in %s", count, data)
	}
	if count := strings.Count(data, "dbc,platform,dev,src-project"); count != 1 {
		t.Fatalf("expected one platform/dev csv branch, got %d in %s", count, data)
	}
	if !strings.Contains(data, ",global,ACTIVE,true,shared-key,shared-project,us-central1,shared-vpc,attachment-shared,ACTIVE,,router-shared,64540,if-shared,169.254.30.1,peer-shared,169.254.30.2,64560,UP") {
		t.Fatalf("expected source macsec fields in csv output, got %s", data)
	}
}

func containsBrailleSpinner(value string) bool {
	for _, frame := range brailleSpinnerFrames {
		if strings.Contains(value, frame) {
			return true
		}
	}
	return false
}

func containsTaskTable(value string) bool {
	return strings.Contains(value, "+-") && strings.Contains(value, "|")
}

const validConfig = `
org:
  - name: dbc
    workload:
      - name: native
        env:
          - name: dev
            project_id: project
`

const fanoutConfig = `
org:
  - name: dbc
    workload:
      - name: native
        env:
          - name: dev
            project_id: project-a
          - name: prod
            project_id: project-b
`

const duplicateProjectConfig = `
org:
  - name: dbc
    workload:
      - name: native
        env:
          - name: dev
            project_id: shared-project
      - name: platform
        env:
          - name: dev
            project_id: shared-project
`
