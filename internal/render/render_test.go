package render

import (
	"strings"
	"testing"

	"mindmap/internal/model"
)

func sampleReport() model.Report {
	return model.Report{
		Type:               "interconnect",
		SourceProject:      "src",
		DestinationProject: "",
		Selectors: model.Selectors{
			Org:         "dbc",
			Workload:    "native",
			Environment: "dev",
		},
		Items: []model.MappingItem{
			{
				SrcProject:      "src",
				SrcInterconnect: "ic-1",
				SrcRegion:       "global",
				SrcState:        "ACTIVE",
				DstProject:      "dst-a",
				Region:          "us-central1",
				Attachment:      "attachment-1",
				AttachmentState: "ACTIVE",
				Router:          "router-1",
				Interface:       "if-1",
				BGPPeerName:     "peer-1",
				LocalIP:         "169.254.1.1",
				RemoteIP:        "169.254.1.2",
				BGPStatus:       "UP",
				Mapped:          true,
			},
			{
				SrcProject:      "src",
				SrcInterconnect: "ic-1",
				SrcRegion:       "global",
				SrcState:        "ACTIVE",
				DstProject:      "dst-b",
				Mapped:          false,
			},
		},
	}
}

func TestRenderCSV(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatCSV)
	if err != nil {
		t.Fatalf("render csv: %v", err)
	}
	if ext != "csv" {
		t.Fatalf("expected csv extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "org,src_project,src_interconnect,dst_project,region") {
		t.Fatalf("unexpected csv header order: %s", content)
	}
	if !strings.Contains(content, "interface,local_ip,bgp_peer_name,remote_ip") {
		t.Fatalf("unexpected csv IP column order: %s", content)
	}
}

func TestRenderJSON(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatJSON)
	if err != nil {
		t.Fatalf("render json: %v", err)
	}
	if ext != "json" {
		t.Fatalf("expected json extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, `"org": {`) || !strings.Contains(content, `"src_projects"`) {
		t.Fatalf("unexpected json output: %s", content)
	}
	if !strings.Contains(content, `"dst_projects"`) || !strings.Contains(content, `"regions"`) {
		t.Fatalf("expected hierarchical destination data, got: %s", content)
	}
}

func TestRenderTree(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatTree)
	if err != nil {
		t.Fatalf("render tree: %v", err)
	}
	if ext != "tree.txt" {
		t.Fatalf("expected tree extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "dbc\n`-- src") {
		t.Fatalf("unexpected tree root: %s", content)
	}
	if !strings.Contains(content, "attachment: attachment-1 [ACTIVE]") || !strings.Contains(content, "dst-b") {
		t.Fatalf("unexpected tree output: %s", content)
	}
}

func TestRenderMermaid(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatMermaid)
	if err != nil {
		t.Fatalf("render mermaid: %v", err)
	}
	if ext != "mmd" {
		t.Fatalf("expected mmd extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "flowchart LR") {
		t.Fatalf("expected flowchart output, got %s", content)
	}
	if !strings.Contains(content, "if: if-1 | local: 169.254.1.1") || !strings.Contains(content, "bgp: peer-1 | remote: 169.254.1.2 | UP") {
		t.Fatalf("expected compact interface/BGP details, got %s", content)
	}
	if strings.Contains(content, "root((GCP") {
		t.Fatalf("expected minimal mermaid output, got %s", content)
	}
}
