package app

import (
	"context"
	"errors"
	"strings"
	"testing"

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
	interconnects []model.DedicatedInterconnect
	attachments   []model.VLANAttachment
	routers       []model.CloudRouter
	statuses      map[string]model.RouterStatus
}

func (m mockProvider) ListDedicatedInterconnects(context.Context, string) ([]model.DedicatedInterconnect, error) {
	return m.interconnects, nil
}

func (m mockProvider) ListVLANAttachments(context.Context, string) ([]model.VLANAttachment, error) {
	return m.attachments, nil
}

func (m mockProvider) ListCloudRouters(context.Context, string) ([]model.CloudRouter, error) {
	return m.routers, nil
}

func (m mockProvider) GetCloudRouterStatus(context.Context, string, string, string) (model.RouterStatus, error) {
	for _, status := range m.statuses {
		return status, nil
	}
	return model.RouterStatus{}, nil
}

func TestParseOptionsValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing t", args: []string{}, want: "missing mandatory parameter -t"},
		{name: "invalid t", args: []string{"-t", "bad", "-o", "dbc", "-w", "native", "-e", "dev"}, want: "invalid -t value"},
		{name: "missing p for interconnect", args: []string{"-t", "interconnect", "-o", "dbc", "-w", "native", "-e", "dev"}, want: "missing mandatory parameter -p"},
		{name: "forbid p for vpn", args: []string{"-t", "vpn", "-o", "dbc", "-w", "native", "-e", "dev", "-p", "src"}, want: "-p must not be used"},
		{name: "invalid format", args: []string{"-t", "interconnect", "-o", "dbc", "-w", "native", "-e", "dev", "-p", "src", "-f", "xml"}, want: "invalid -f value"},
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

	data, ok := store.files["mindmap-interconnect-src-project-to-project.mmd"]
	if !ok {
		t.Fatalf("expected mermaid output file to be written")
	}
	content := string(data)
	if !strings.Contains(content, "attachment-1") || !strings.Contains(content, "peer-1") {
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

	if _, ok := store.files["mindmap-interconnect-src-project-to-project.mmd"]; ok {
		t.Fatalf("unexpected mermaid output")
	}
	if _, ok := store.files["mindmap-interconnect-src-project-to-project.json"]; !ok {
		t.Fatalf("expected json output")
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
