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
		DestinationProject: "dst",
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
				DstProject:      "dst",
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
	if !strings.Contains(content, "src_project") || !strings.Contains(content, "us-central1") {
		t.Fatalf("unexpected csv output: %s", content)
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
	if !strings.Contains(content, `"region": "us-central1"`) {
		t.Fatalf("unexpected json output: %s", content)
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
	if !strings.Contains(content, "region: us-central1") {
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
	if !strings.Contains(content, "attachment-1") || !strings.Contains(content, "peer-1") {
		t.Fatalf("unexpected mermaid output: %s", content)
	}
}
