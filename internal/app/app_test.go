package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"mindmap/internal/model"
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
}

func (m mockProvider) ListDedicatedInterconnects(context.Context, string) ([]model.DedicatedInterconnect, error) {
	return m.interconnects, nil
}

func (m mockProvider) ListVLANAttachments(_ context.Context, project string) ([]model.VLANAttachment, error) {
	if len(m.attachmentsByProject) > 0 {
		return m.attachmentsByProject[project], nil
	}
	return m.attachments, nil
}

func (m mockProvider) ListCloudRouters(_ context.Context, project string) ([]model.CloudRouter, error) {
	if len(m.routersByProject) > 0 {
		return m.routersByProject[project], nil
	}
	return m.routers, nil
}

func (m mockProvider) GetCloudRouterStatus(_ context.Context, project, region, router string) (model.RouterStatus, error) {
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
}

func TestRunWritesMermaidByDefault(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}
	app, err := New(store, mockProvider{
		interconnects: []model.DedicatedInterconnect{{Name: "ic-1", State: "ACTIVE"}},
		attachments: []model.VLANAttachment{{
			Name:         "attachment-1",
			Region:       "us-central1",
			State:        "ACTIVE",
			Interconnect: "ic-1",
			Router:       "router-1",
		}},
		routers: []model.CloudRouter{{
			Name:   "router-1",
			Region: "us-central1",
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

	data, ok := store.files["mindmap-interconnect-src-project-to-project-20260328T000000Z.mmd"]
	if !ok {
		t.Fatalf("expected mermaid output file to be written")
	}
	content := string(data)
	if !strings.Contains(content, "flowchart LR") || !strings.Contains(content, "peer-1") || !strings.Contains(content, "if: if-1") {
		t.Fatalf("unexpected mermaid content: %s", content)
	}
}

func TestRunSuppressesMermaidWhenFormatProvided(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(validConfig),
		},
	}
	app, err := New(store, mockProvider{
		interconnects: []model.DedicatedInterconnect{{Name: "ic-1", State: "ACTIVE"}},
	})
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	app.now = func() time.Time {
		return time.Date(2026, time.March, 28, 0, 0, 0, 0, time.UTC)
	}

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

	if _, ok := store.files["mindmap-interconnect-src-project-to-project-20260328T000000Z.mmd"]; ok {
		t.Fatalf("unexpected mermaid output")
	}
	if _, ok := store.files["mindmap-interconnect-src-project-to-project-20260328T000000Z.json"]; !ok {
		t.Fatalf("expected json output")
	}
}

func TestRunWithOrgFanoutWritesCombinedOutput(t *testing.T) {
	store := &memoryFileStore{
		files: map[string][]byte{
			"config.yaml": []byte(fanoutConfig),
		},
	}
	app, err := New(store, mockProvider{
		interconnects: []model.DedicatedInterconnect{{Name: "ic-1", State: "ACTIVE"}},
		attachmentsByProject: map[string][]model.VLANAttachment{
			"project-a": {{
				Name:         "attachment-a",
				Region:       "us-central1",
				State:        "ACTIVE",
				Interconnect: "ic-1",
				Router:       "router-a",
			}},
			"project-b": {{
				Name:         "attachment-b",
				Region:       "europe-west1",
				State:        "ACTIVE",
				Interconnect: "ic-1",
				Router:       "router-b",
			}},
		},
		routersByProject: map[string][]model.CloudRouter{
			"project-a": {{
				Name:   "router-a",
				Region: "us-central1",
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
					SessionState: "UP",
				}},
			}},
			"project-b": {{
				Name:   "router-b",
				Region: "europe-west1",
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

	err = app.Run(context.Background(), []string{
		"-t", "interconnect",
		"-o", "dbc",
		"-p", "src-project",
		"-f", "tree",
	})
	if err != nil {
		t.Fatalf("run app: %v", err)
	}

	data, ok := store.files["mindmap-interconnect-src-project-to-dbc-all-20260328T000000Z.tree.txt"]
	if !ok {
		t.Fatalf("expected combined output file to be written")
	}
	content := string(data)
	if !strings.Contains(content, "project-a") || !strings.Contains(content, "project-b") {
		t.Fatalf("expected fanout destinations in tree output, got: %s", content)
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
			{Name: "unmapped", State: "DOWN"},
		},
		[]model.VLANAttachment{{
			Name:         "attachment-1",
			Region:       "europe-west1",
			State:        "ACTIVE",
			Interconnect: "mapped",
			Router:       "router-1",
		}},
		[]model.CloudRouter{{
			Name:   "router-1",
			Region: "europe-west1",
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
	foundUnmapped := false
	for _, item := range items {
		if item.SrcInterconnect == "unmapped" && !item.Mapped {
			foundUnmapped = true
		}
	}
	if !foundUnmapped {
		t.Fatalf("expected unmapped interconnect item")
	}
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
