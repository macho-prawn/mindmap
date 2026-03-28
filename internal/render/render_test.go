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
				SrcMacsecEnabled:          true,
				SrcMacsecKeyName:          "macsec-key-a",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVLANAttachment:         "attachment-1",
				DstVLANAttachmentState:    "ACTIVE",
				DstVLANAttachmentVLANID:   "101",
				DstCloudRouter:            "router-1",
				DstCloudRouterASN:         "64512",
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
				SrcMacsecEnabled:          false,
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVLANAttachment:         "attachment-2",
				DstVLANAttachmentState:    "ACTIVE",
				DstVLANAttachmentVLANID:   "102",
				DstCloudRouter:            "router-2",
				DstCloudRouterASN:         "64513",
				DstCloudRouterInterface:   "if-2",
				DstCloudRouterInterfaceIP: "169.254.2.1",
				RemoteBGPPeer:             "peer-2",
				RemoteBGPPeerIP:           "169.254.2.2",
				BGPPeeringStatus:          "UP",
			},
			{
				Org:              "dbc",
				Workload:         "platform",
				Environment:      "dev",
				SrcProject:       "src",
				SrcInterconnect:  "ic-1",
				Mapped:           false,
				SrcRegion:        "global",
				SrcState:         "ACTIVE",
				SrcMacsecEnabled: true,
				SrcMacsecKeyName: "macsec-key-platform",
				DstProject:       "dst-b",
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
	if !strings.Contains(content, "org,workload,environment,src_project,src_interconnect,mapped,src_region,src_state,src_macsec_enabled,src_macsec_keyname,dst_project,dst_region,dst_vlan_attachment") {
		t.Fatalf("unexpected csv header order: %s", content)
	}
	if !strings.Contains(content, "dst_cloud_router,dst_cloud_router_asn,dst_cloud_router_interface,dst_cloud_router_interface_ip,remote_bgp_peer,remote_bgp_peer_ip,bgp_peering_status") {
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
	if !strings.Contains(content, `"src_macsec_enabled": true`) || !strings.Contains(content, `"src_macsec_keyname": "macsec-key-a"`) {
		t.Fatalf("expected source macsec data in json output, got: %s", content)
	}
	if !strings.Contains(content, `"dst_cloud_router_asn": "64512"`) {
		t.Fatalf("expected router asn in json output, got: %s", content)
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
	if !strings.Contains(content, "src_macsec_enabled: true, src_macsec_keyname: macsec-key-a") {
		t.Fatalf("expected source macsec data in tree output: %s", content)
	}
	if !strings.Contains(content, "dst_cloud_router: router-1 [dst_cloud_router_asn: 64512]") {
		t.Fatalf("expected router asn in tree output: %s", content)
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
	if !strings.Contains(content, "<br>") {
		t.Fatalf("expected mermaid.live-compatible line breaks, got %s", content)
	}
	if strings.Contains(content, "\\n") {
		t.Fatalf("unexpected escaped newline in mermaid output: %s", content)
	}
	if !strings.Contains(content, "src_macsec_enabled: true") || !strings.Contains(content, "src_macsec_keyname: macsec-key-a") {
		t.Fatalf("expected source macsec data in mermaid output, got %s", content)
	}
	if !strings.Contains(content, "dst_cloud_router_asn: 64512") {
		t.Fatalf("expected router asn in mermaid output, got %s", content)
	}
	if countSubstring(content, "dst_project: dst-a") != 1 {
		t.Fatalf("expected one shared dst_project node, got %d in %s", countSubstring(content, "dst_project: dst-a"), content)
	}
	if countSubstring(content, "dst_region: us-central1") != 1 {
		t.Fatalf("expected one shared dst_region node, got %d in %s", countSubstring(content, "dst_region: us-central1"), content)
	}
	if countSubstring(content, "environment: dev") != 1 {
		t.Fatalf("expected one shared environment node, got %d in %s", countSubstring(content, "environment: dev"), content)
	}
	if countSubstring(content, "src_project: src") != 1 {
		t.Fatalf("expected one shared src_project node, got %d in %s", countSubstring(content, "src_project: src"), content)
	}
	if countSubstring(content, "src_interconnect: ic-1") != 1 {
		t.Fatalf("expected one shared ic-1 node, got %d in %s", countSubstring(content, "src_interconnect: ic-1"), content)
	}
}

func TestRenderMermaidCollapsesAcrossWorkloadsEnvironmentsAndDestinations(t *testing.T) {
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
				SrcMacsecEnabled:        true,
				SrcMacsecKeyName:        "shared-key",
				DstProject:              "project-a",
				DstRegion:               "us-central1",
				DstVLANAttachment:       "attachment-native",
				DstVLANAttachmentState:  "ACTIVE",
				DstVLANAttachmentVLANID: "100",
				DstCloudRouter:          "router-native",
				DstCloudRouterASN:       "64520",
			},
			{
				Org:                     "dbc",
				Workload:                "platform",
				Environment:             "dev",
				SrcProject:              "src",
				SrcInterconnect:         "ic-native",
				Mapped:                  true,
				SrcRegion:               "global",
				SrcState:                "ACTIVE",
				SrcMacsecEnabled:        true,
				SrcMacsecKeyName:        "shared-key",
				DstProject:              "project-b",
				DstRegion:               "us-central1",
				DstVLANAttachment:       "attachment-platform-dev",
				DstVLANAttachmentState:  "ACTIVE",
				DstVLANAttachmentVLANID: "200",
				DstCloudRouter:          "router-platform-dev",
				DstCloudRouterASN:       "64521",
			},
			{
				Org:                     "dbc",
				Workload:                "native",
				Environment:             "prod",
				SrcProject:              "src",
				SrcInterconnect:         "ic-native",
				Mapped:                  true,
				SrcRegion:               "global",
				SrcState:                "ACTIVE",
				SrcMacsecEnabled:        true,
				SrcMacsecKeyName:        "shared-key",
				DstProject:              "project-c",
				DstRegion:               "us-central1",
				DstVLANAttachment:       "attachment-native-prod",
				DstVLANAttachmentState:  "ACTIVE",
				DstVLANAttachmentVLANID: "300",
				DstCloudRouter:          "router-native-prod",
				DstCloudRouterASN:       "64522",
			},
		},
	}

	data, _, err := Render(report, FormatMermaid)
	if err != nil {
		t.Fatalf("render mermaid: %v", err)
	}

	content := string(data)
	if countSubstring(content, "environment: dev") != 1 {
		t.Fatalf("expected one shared dev environment node, got %d in %s", countSubstring(content, "environment: dev"), content)
	}
	if countSubstring(content, "src_project: src") != 1 {
		t.Fatalf("expected one shared src_project node, got %d in %s", countSubstring(content, "src_project: src"), content)
	}
	if countSubstring(content, "src_interconnect: ic-native") != 1 {
		t.Fatalf("expected one shared dedicated interconnect node, got %d in %s", countSubstring(content, "src_interconnect: ic-native"), content)
	}
	if countSubstring(content, "dst_region: us-central1") != 1 {
		t.Fatalf("expected one shared dst_region node, got %d in %s", countSubstring(content, "dst_region: us-central1"), content)
	}
	if countSubstring(content, "dst_project: project-") != 3 {
		t.Fatalf("expected three distinct dst_project nodes, got %d in %s", countSubstring(content, "dst_project: project-"), content)
	}
}

func countSubstring(content, needle string) int {
	return strings.Count(content, needle)
}
