package app

import (
	"bytes"
	"context"
	"errors"
	"os"
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
	vpnGateways            []model.VPNGateway
	targetVPNGateways      []model.VPNGateway
	vpnTunnels             []model.VPNTunnel
	routers                []model.CloudRouter
	statuses               map[string]model.RouterStatus
	attachmentsByProject   map[string][]model.VLANAttachment
	vpnGatewaysByProject   map[string][]model.VPNGateway
	targetVPNByProject     map[string][]model.VPNGateway
	vpnTunnelsByProject    map[string][]model.VPNTunnel
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

func (m mockProvider) ListVPNGateways(_ context.Context, project string) ([]model.VPNGateway, error) {
	if len(m.vpnGatewaysByProject) > 0 {
		return m.vpnGatewaysByProject[project], nil
	}
	return m.vpnGateways, nil
}

func (m mockProvider) ListTargetVPNGateways(_ context.Context, project string) ([]model.VPNGateway, error) {
	if len(m.targetVPNByProject) > 0 {
		return m.targetVPNByProject[project], nil
	}
	return m.targetVPNGateways, nil
}

func (m mockProvider) ListVPNTunnels(_ context.Context, project string) ([]model.VPNTunnel, error) {
	if len(m.vpnTunnelsByProject) > 0 {
		return m.vpnTunnelsByProject[project], nil
	}
	return m.vpnTunnels, nil
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
		{name: "missing t", args: []string{"-o", "dbc"}, want: "missing mandatory parameter -t"},
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

func TestParseOptionsWithoutArgsShowsHelp(t *testing.T) {
	opts, err := ParseOptions(nil)
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if !opts.ShowHelp || !strings.Contains(opts.Usage, "Usage:") {
		t.Fatalf("expected help usage, got %+v", opts)
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

func TestParseOptionsAcceptsShortConfigFlag(t *testing.T) {
	opts, err := ParseOptions([]string{"-t", "interconnect", "-o", "dbc", "-p", "src-project", "-c", "custom.yaml"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}
	if opts.ConfigPath != "custom.yaml" {
		t.Fatalf("expected custom config path, got %+v", opts)
	}
}

func TestParseOptionsRejectsLegacyConfigFlag(t *testing.T) {
	_, err := ParseOptions([]string{"-t", "interconnect", "-o", "dbc", "-p", "src-project", "-config", "custom.yaml"})
	if err == nil || !strings.Contains(err.Error(), "flag provided but not defined: -config") {
		t.Fatalf("expected legacy config flag error, got %v", err)
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
	if !strings.Contains(opts.Usage, "Usage:\n\n  netmap [-h]\n  netmap version") {
		t.Fatalf("expected bare command and version usage, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "-o + -e        expands all workloads containing that environment") {
		t.Fatalf("expected explicit -o + -e help text, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "Omit -f to write Mermaid output by default.") {
		t.Fatalf("expected default mermaid guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "HTML output file:    netmap-interconnect-<src>-to-<dst>-<timestamp>.html") {
		t.Fatalf("expected html output guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "-c        optional, defaults to config.yaml") || strings.Contains(opts.Usage, "-config") {
		t.Fatalf("expected short config flag help text, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "-f        optional, output format override: html, csv, tsv, json, or tree") {
		t.Fatalf("expected html in format help text, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "netmap version") || !strings.Contains(opts.Usage, "print the current netmap version and exit") {
		t.Fatalf("expected version command help text, got %+v", opts)
	}
	if strings.Contains(opts.Usage, "Stderr shows an ASCII 2-column task table") || strings.Contains(opts.Usage, "Completed rows use a tick marker") || strings.Contains(opts.Usage, "The final merged row prints Output: <path> and Total Time: <duration>.") {
		t.Fatalf("expected simplified output section, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "Org fanout output:   netmap-interconnect-<src>-to-<org>-all-<timestamp>.<ext>") {
		t.Fatalf("expected org fanout output guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "VPN output file:     netmap-vpn-<src>-to-<dst>-<timestamp>.<ext>") {
		t.Fatalf("expected vpn output guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "VPN aggregate file:  netmap-vpn-<org>-all-<timestamp>.<ext>") {
		t.Fatalf("expected vpn aggregate output guidance, got %+v", opts)
	}
	if !strings.Contains(opts.Usage, "Use -f html to write a self-contained offline Mermaid viewer page.") {
		t.Fatalf("expected blank line before html viewer note, got %+v", opts)
	}
}

func TestUsageTextMatchesEmbeddedSourceFile(t *testing.T) {
	data, err := os.ReadFile("usage.txt")
	if err != nil {
		t.Fatalf("read usage.txt: %v", err)
	}
	if usageText() != string(data) {
		t.Fatalf("expected embedded usage text to match usage.txt")
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
		t.Fatalf("expected mermaid-compatible line breaks, got: %s", content)
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

func TestRunWritesOfflineHTMLWhenFormatProvided(t *testing.T) {
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
		"-f", "html",
	})
	if err != nil {
		t.Fatalf("run app: %v", err)
	}

	if _, ok := store.files["netmap-interconnect-src-project-to-project-20260328T000000Z.mmd"]; ok {
		t.Fatalf("unexpected mermaid output")
	}
	data, ok := store.files["netmap-interconnect-src-project-to-project-20260328T000000Z.html"]
	if !ok {
		t.Fatalf("expected html output")
	}
	content := string(data)
	if !strings.Contains(content, "<!DOCTYPE html>") || !strings.Contains(content, "flowchart LR") || !strings.Contains(content, "mermaid.initialize") {
		t.Fatalf("expected offline mermaid html output, got: %s", content)
	}
	if strings.Contains(content, "https://mermaid.live") {
		t.Fatalf("expected offline html output without external viewer guidance, got: %s", content)
	}
	if !strings.Contains(status.String(), "Output: netmap-interconnect-src-project-to-project-20260328T000000Z.html") || !strings.Contains(status.String(), "Total Time: 0s") {
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

func TestRunWritesVPNMermaidByDefault(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}
	app, err := New(store, mockProvider{
		vpnGatewaysByProject: map[string][]model.VPNGateway{
			"project": {{
				Name:     "ha-src",
				Region:   "us-central1",
				Network:  "src-vpc",
				Type:     "ha",
				Status:   "unknown",
				SelfLink: "https://www.googleapis.com/compute/v1/projects/project/regions/us-central1/vpnGateways/ha-src",
			}},
			"peer-project": {{
				Name:     "ha-dst",
				Region:   "us-central1",
				Network:  "dst-vpc",
				Type:     "ha",
				Status:   "unknown",
				SelfLink: "https://www.googleapis.com/compute/v1/projects/peer-project/regions/us-central1/vpnGateways/ha-dst",
			}},
		},
		vpnTunnelsByProject: map[string][]model.VPNTunnel{
			"project": {{
				Name:                "tunnel-src",
				Region:              "us-central1",
				Status:              "ESTABLISHED",
				VPNGateway:          "ha-src",
				PeerGCPGateway:      "https://www.googleapis.com/compute/v1/projects/peer-project/regions/us-central1/vpnGateways/ha-dst",
				Router:              "router-src",
				VPNGatewayInterface: "0",
			}},
			"peer-project": {{
				Name:                "tunnel-dst",
				Region:              "us-central1",
				Status:              "ESTABLISHED",
				VPNGateway:          "ha-dst",
				PeerGCPGateway:      "https://www.googleapis.com/compute/v1/projects/project/regions/us-central1/vpnGateways/ha-src",
				Router:              "router-dst",
				VPNGatewayInterface: "0",
			}},
		},
		routersByProject: map[string][]model.CloudRouter{
			"project": {{
				Name:    "router-src",
				Region:  "us-central1",
				ASN:     "64512",
				Network: "src-vpc",
				Interfaces: []model.RouterInterface{{
					Name:            "if-src",
					LinkedVPNTunnel: "tunnel-src",
					IPRange:         "169.254.1.1/30",
				}},
			}},
			"peer-project": {{
				Name:    "router-dst",
				Region:  "us-central1",
				ASN:     "64513",
				Network: "dst-vpc",
				Interfaces: []model.RouterInterface{{
					Name:            "if-dst",
					LinkedVPNTunnel: "tunnel-dst",
					IPRange:         "169.254.1.2/30",
				}},
				BGPPeers: []model.BGPPeer{{
					Name:         "peer-dst",
					Interface:    "if-dst",
					LocalIP:      "169.254.1.2",
					RemoteIP:     "169.254.1.1",
					PeerASN:      "64512",
					SessionState: "UP",
				}},
			}},
		},
		statusesByProjectRoute: map[string]model.RouterStatus{
			"project/us-central1/router-src": {
				RouterName: "router-src",
				Region:     "us-central1",
			},
			"peer-project/us-central1/router-dst": {
				RouterName: "router-dst",
				Region:     "us-central1",
				Peers: []model.BGPPeerStatus{{
					Name:         "peer-dst",
					LocalIP:      "169.254.1.2",
					RemoteIP:     "169.254.1.1",
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

	projectItems, err := app.buildVPNProjectItems(context.Background(), "project", map[string]vpnProjectData{})
	if err != nil {
		t.Fatalf("build vpn project items: %v", err)
	}
	if len(projectItems) == 0 {
		t.Fatalf("expected vpn project items to be built")
	}

	err = app.Run(context.Background(), []string{
		"-t", "vpn",
		"-o", "dbc",
		"-w", "native",
		"-e", "dev",
	})
	if err != nil {
		t.Fatalf("run app: %v", err)
	}

	data, ok := store.files["netmap-vpn-project-to-peer-project-20260328T000000Z.mmd"]
	if !ok {
		t.Fatalf("expected vpn mermaid output file to be written, got files: %#v", store.files)
	}
	content := string(data)
	if !strings.Contains(content, "src_vpn_gateway: ha-src") || !strings.Contains(content, "src_vpn_tunnel: tunnel-src") {
		t.Fatalf("expected vpn source nodes in mermaid output, got: %s", content)
	}
	if !strings.Contains(content, "dst_vpn_gateway: ha-dst") || !strings.Contains(content, "dst_vpn_tunnel: tunnel-dst") {
		t.Fatalf("expected vpn destination nodes in mermaid output, got: %s", content)
	}
	if !strings.Contains(content, "dst_cloud_router: router-dst") || !strings.Contains(content, "remote_bgp_peer: peer-dst") {
		t.Fatalf("expected destination router and peer details in mermaid output, got: %s", content)
	}
	if strings.Contains(content, "src_interconnect:") {
		t.Fatalf("unexpected interconnect node in vpn mermaid output: %s", content)
	}
	statusOutput := status.String()
	if !strings.Contains(statusOutput, "Output: netmap-vpn-project-to-peer-project-20260328T000000Z.mmd") || !strings.Contains(statusOutput, "Total Time: 0s") {
		t.Fatalf("expected vpn final summary row, got: %s", statusOutput)
	}
}

func TestBuildVPNProjectItemsIncludesClassicUnmappedTunnel(t *testing.T) {
	store := &memoryFileStore{files: map[string][]byte{}}
	app, err := New(store, mockProvider{
		targetVPNByProject: map[string][]model.VPNGateway{
			"project": {{
				Name:     "classic-src",
				Region:   "us-central1",
				Network:  "src-vpc",
				Type:     "classic",
				Status:   "READY",
				SelfLink: "https://www.googleapis.com/compute/v1/projects/project/regions/us-central1/targetVpnGateways/classic-src",
			}},
		},
		vpnTunnelsByProject: map[string][]model.VPNTunnel{
			"project": {{
				Name:             "classic-tunnel",
				Region:           "us-central1",
				Status:           "ESTABLISHED",
				TargetVPNGateway: "classic-src",
				Router:           "router-src",
				PeerIP:           "203.0.113.10",
			}},
		},
		routersByProject: map[string][]model.CloudRouter{
			"project": {{
				Name:    "router-src",
				Region:  "us-central1",
				ASN:     "64512",
				Network: "src-vpc",
				Interfaces: []model.RouterInterface{{
					Name:            "if-src",
					LinkedVPNTunnel: "classic-tunnel",
					IPRange:         "169.254.10.1/30",
				}},
			}},
		},
		statusesByProjectRoute: map[string]model.RouterStatus{
			"project/us-central1/router-src": {
				RouterName: "router-src",
				Region:     "us-central1",
			},
		},
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}

	items, err := app.buildVPNProjectItems(context.Background(), "project", map[string]vpnProjectData{})
	if err != nil {
		t.Fatalf("build vpn project items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one classic vpn item, got %d", len(items))
	}
	item := items[0]
	if item.Mapped {
		t.Fatalf("expected classic vpn item to remain unmapped, got %+v", item)
	}
	if item.SrcVPNGateway != "classic-src" || item.SrcVPNGatewayType != "classic" || item.SrcVPNTunnel != "classic-tunnel" {
		t.Fatalf("expected classic vpn source fields, got %+v", item)
	}
	if item.DstProject != "" || item.DstVPNGateway != "" || item.DstVPNTunnel != "" {
		t.Fatalf("expected no destination mapping for classic vpn item, got %+v", item)
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
	if !strings.Contains(data, ",,,,,true,global,ACTIVE,true,shared-key,shared-project,us-central1,shared-vpc,attachment-shared,ACTIVE,,,,,,,router-shared,64540,if-shared,169.254.30.1,peer-shared,169.254.30.2,64560,UP") {
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
