package render

import (
	"strings"
	"testing"

	"netmap/internal/model"
)

func sampleReport() model.Report {
	return model.Report{
		Type:               "interconnect",
		SourceProject:      "src",
		DestinationProject: "",
		Selectors: model.Selectors{
			Org: "dbc",
		},
		Items: []model.MappingItem{
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src",
				SrcInterconnect:           "ic-1",
				Mapped:                    true,
				SrcRegion:                 "global",
				SrcState:                  "ACTIVE",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVLANAttachment:         "attachment-1",
				DstVLANAttachmentState:    "ACTIVE",
				DstVLANAttachmentVLANID:   "101",
				DstCloudRouter:            "router-1",
				DstCloudRouterInterface:   "if-1",
				DstCloudRouterInterfaceIP: "169.254.1.1",
				RemoteBGPPeer:             "peer-1",
				RemoteBGPPeerIP:           "169.254.1.2",
				BGPPeeringStatus:          "UP",
			},
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src",
				SrcInterconnect:           "ic-2",
				Mapped:                    true,
				SrcRegion:                 "global",
				SrcState:                  "ACTIVE",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVLANAttachment:         "attachment-2",
				DstVLANAttachmentState:    "ACTIVE",
				DstVLANAttachmentVLANID:   "102",
				DstCloudRouter:            "router-2",
				DstCloudRouterInterface:   "if-2",
				DstCloudRouterInterfaceIP: "169.254.2.1",
				RemoteBGPPeer:             "peer-2",
				RemoteBGPPeerIP:           "169.254.2.2",
				BGPPeeringStatus:          "UP",
			},
			{
				Org:             "dbc",
				Workload:        "platform",
				Environment:     "dev",
				SrcProject:      "src",
				SrcInterconnect: "ic-1",
				Mapped:          false,
				SrcRegion:       "global",
				SrcState:        "ACTIVE",
				DstProject:      "dst-b",
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
	if strings.Contains(content, "dst_cloud_router_state") {
		t.Fatalf("unexpected router state column in csv: %s", content)
	}
	if !strings.Contains(content, "org,workload,environment,src_project,src_interconnect,mapped,src_region,src_state,dst_project,dst_region") {
		t.Fatalf("unexpected csv header order: %s", content)
	}
	if !strings.Contains(content, "dst_cloud_router,dst_cloud_router_interface,dst_cloud_router_interface_ip,remote_bgp_peer,remote_bgp_peer_ip,bgp_peering_status") {
		t.Fatalf("unexpected csv tail column order: %s", content)
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
	if strings.Contains(content, `"dst_cloud_router_state"`) {
		t.Fatalf("unexpected router state in json output: %s", content)
	}
	if !strings.Contains(content, `"org": {`) || !strings.Contains(content, `"workloads"`) || !strings.Contains(content, `"environments"`) {
		t.Fatalf("unexpected json output: %s", content)
	}
	if !strings.Contains(content, `"name": "native"`) || !strings.Contains(content, `"name": "platform"`) {
		t.Fatalf("expected workload hierarchy, got: %s", content)
	}
	if !strings.Contains(content, `"src_interconnects"`) || !strings.Contains(content, `"dst_projects"`) || !strings.Contains(content, `"dst_regions"`) {
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
	if strings.Contains(content, "dst_cloud_router_state") {
		t.Fatalf("unexpected router state in tree output: %s", content)
	}
	if !strings.Contains(content, "org: dbc\n|-- workload: native\n|   `-- environment: dev\n|       `-- src_project: src") {
		t.Fatalf("unexpected tree hierarchy: %s", content)
	}
	if !strings.Contains(content, "`-- workload: platform") {
		t.Fatalf("expected second workload branch, got: %s", content)
	}
	if !strings.Contains(content, "dst_vlan_attachment: attachment-1 [dst_vlan_attachment_state: ACTIVE, dst_vlan_attachment_vlanid: 101]") || !strings.Contains(content, "dst_project: dst-b [mapped: false]") {
		t.Fatalf("unexpected tree output: %s", content)
	}
}

func TestRenderMermaidCollapsesSharedProjectAndRegion(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatMermaid)
	if err != nil {
		t.Fatalf("render mermaid: %v", err)
	}
	if ext != "mmd" {
		t.Fatalf("expected mmd extension, got %q", ext)
	}
	content := string(data)
	if strings.Contains(content, "dst_cloud_router_state") {
		t.Fatalf("unexpected router state in mermaid output: %s", content)
	}
	if !strings.Contains(content, "flowchart LR") {
		t.Fatalf("expected flowchart output, got %s", content)
	}
	if !strings.Contains(content, "org: dbc") || !strings.Contains(content, "workload: native") || !strings.Contains(content, "environment: dev") {
		t.Fatalf("expected selector hierarchy, got %s", content)
	}
	if !strings.Contains(content, "dst_region: us-central1") || !strings.Contains(content, "dst_vlan_attachment: attachment-1") || !strings.Contains(content, "remote_bgp_peer: peer-1") {
		t.Fatalf("expected destination fanout details, got %s", content)
	}
	if countSubstring(content, "dst_project: dst-a") != 1 {
		t.Fatalf("expected one shared dst_project node, got %d in %s", countSubstring(content, "dst_project: dst-a"), content)
	}
	if countSubstring(content, "dst_region: us-central1") != 1 {
		t.Fatalf("expected one shared dst_region node, got %d in %s", countSubstring(content, "dst_region: us-central1"), content)
	}
}

func TestRenderMermaidKeepsDestinationScopeUniquePerTuple(t *testing.T) {
	report := model.Report{
		Type:          "interconnect",
		SourceProject: "src",
		Selectors: model.Selectors{
			Org: "dbc",
		},
		Items: []model.MappingItem{
			{
				Org:                     "dbc",
				Workload:                "native",
				Environment:             "dev",
				SrcProject:              "src",
				SrcInterconnect:         "ic-native",
				Mapped:                  true,
				SrcRegion:               "global",
				SrcState:                "ACTIVE",
				DstProject:              "shared-project",
				DstRegion:               "us-central1",
				DstVLANAttachment:       "attachment-native",
				DstVLANAttachmentState:  "ACTIVE",
				DstVLANAttachmentVLANID: "100",
				DstCloudRouter:          "router-native",
			},
			{
				Org:                     "dbc",
				Workload:                "platform",
				Environment:             "dev",
				SrcProject:              "src",
				SrcInterconnect:         "ic-platform",
				Mapped:                  true,
				SrcRegion:               "global",
				SrcState:                "ACTIVE",
				DstProject:              "shared-project",
				DstRegion:               "us-central1",
				DstVLANAttachment:       "attachment-platform",
				DstVLANAttachmentState:  "ACTIVE",
				DstVLANAttachmentVLANID: "200",
				DstCloudRouter:          "router-platform",
			},
		},
	}

	data, _, err := Render(report, FormatMermaid)
	if err != nil {
		t.Fatalf("render mermaid: %v", err)
	}

	content := string(data)
	if countSubstring(content, "dst_project: shared-project") != 2 {
		t.Fatalf("expected tuple-scoped dst_project nodes, got %d in %s", countSubstring(content, "dst_project: shared-project"), content)
	}
	if countSubstring(content, "dst_region: us-central1") != 2 {
		t.Fatalf("expected tuple-scoped dst_region nodes, got %d in %s", countSubstring(content, "dst_region: us-central1"), content)
	}
}

func countSubstring(content, needle string) int {
	return strings.Count(content, needle)
}
