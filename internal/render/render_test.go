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
				DstVPC:                    "vpc-a",
				DstVLANAttachment:         "attachment-1",
				DstVLANAttachmentState:    "ACTIVE",
				DstVLANAttachmentVLANID:   "101",
				DstCloudRouter:            "router-1",
				DstCloudRouterASN:         "64512",
				DstCloudRouterInterface:   "if-1",
				DstCloudRouterInterfaceIP: "169.254.1.1",
				RemoteBGPPeer:             "peer-1",
				RemoteBGPPeerIP:           "169.254.1.2",
				RemoteBGPPeerASN:          "64550",
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
				DstVPC:                    "vpc-a",
				DstVLANAttachment:         "attachment-2",
				DstVLANAttachmentState:    "ACTIVE",
				DstVLANAttachmentVLANID:   "102",
				DstCloudRouter:            "router-2",
				DstCloudRouterASN:         "64513",
				DstCloudRouterInterface:   "if-2",
				DstCloudRouterInterfaceIP: "169.254.2.1",
				RemoteBGPPeer:             "peer-2",
				RemoteBGPPeerIP:           "169.254.2.2",
				RemoteBGPPeerASN:          "64551",
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

func sampleVPNReport() model.Report {
	return model.Report{
		Type: "vpn",
		Selectors: model.Selectors{
			Org: "dbc",
		},
		Items: []model.MappingItem{
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-a",
				SrcVPC:                    "src-vpc-a",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-a",
				SrcVPNGatewayType:         "ha",
				SrcCloudRouter:            "router-src-a",
				SrcCloudRouterASN:         "64510",
				SrcCloudRouterInterface:   "if-src-a-1",
				SrcCloudRouterInterfaceIP: "169.254.10.1",
				SrcVPNTunnel:              "tunnel-a-1",
				SrcVPNGatewayInterface:    "0",
				SrcVPNGatewayIP:           "34.0.0.1",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				Mapped:                    true,
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-a",
				DstVPNGateway:             "ha-peer",
				DstVPNGatewayType:         "ha",
				DstVPNTunnel:              "tunnel-peer-1",
				DstVPNGatewayInterface:    "0",
				DstVPNGatewayIP:           "35.0.0.1",
				DstVPNTunnelStatus:        "ESTABLISHED",
				DstCloudRouter:            "router-a",
				DstCloudRouterASN:         "64512",
				DstCloudRouterInterface:   "if-dst-a-1",
				DstCloudRouterInterfaceIP: "169.254.20.1",
				BGPPeeringStatus:          "UP",
			},
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-a",
				SrcVPC:                    "src-vpc-a",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-a",
				SrcVPNGatewayType:         "ha",
				SrcCloudRouter:            "router-src-a",
				SrcCloudRouterASN:         "64510",
				SrcCloudRouterInterface:   "if-src-a-2",
				SrcCloudRouterInterfaceIP: "169.254.10.5",
				SrcVPNTunnel:              "tunnel-a-2",
				SrcVPNGatewayInterface:    "1",
				SrcVPNGatewayIP:           "34.0.0.2",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				Mapped:                    true,
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-a",
				DstVPNGateway:             "ha-peer",
				DstVPNGatewayType:         "ha",
				DstVPNTunnel:              "tunnel-peer-2",
				DstVPNGatewayInterface:    "1",
				DstVPNGatewayIP:           "35.0.0.2",
				DstVPNTunnelStatus:        "ESTABLISHED",
				DstCloudRouter:            "router-b",
				DstCloudRouterASN:         "64513",
				DstCloudRouterInterface:   "if-dst-a-2",
				DstCloudRouterInterfaceIP: "169.254.20.5",
				BGPPeeringStatus:          "UP",
			},
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-b",
				SrcVPC:                    "src-vpc-b",
				SrcRegion:                 "europe-west1",
				SrcVPNGateway:             "ha-b",
				SrcVPNGatewayType:         "ha",
				SrcCloudRouter:            "router-src-b",
				SrcCloudRouterASN:         "64511",
				SrcCloudRouterInterface:   "if-src-b-1",
				SrcCloudRouterInterfaceIP: "169.254.30.1",
				SrcVPNTunnel:              "tunnel-b-1",
				SrcVPNGatewayInterface:    "0",
				SrcVPNGatewayIP:           "36.0.0.1",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				Mapped:                    true,
				DstProject:                "dst-b",
				DstRegion:                 "europe-west1",
				DstVPC:                    "dst-vpc-b",
				DstVPNGateway:             "ha-peer-b",
				DstVPNGatewayType:         "ha",
				DstVPNTunnel:              "tunnel-peer-b-1",
				DstVPNGatewayInterface:    "0",
				DstVPNGatewayIP:           "37.0.0.1",
				DstVPNTunnelStatus:        "ESTABLISHED",
				DstCloudRouter:            "router-c",
				DstCloudRouterASN:         "64514",
				DstCloudRouterInterface:   "if-dst-b-1",
				DstCloudRouterInterfaceIP: "169.254.40.1",
				BGPPeeringStatus:          "DOWN",
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
	if !strings.Contains(content, "org,workload,environment,src_project,src_interconnect,mapped,src_region,src_state,src_macsec_enabled,src_macsec_keyname,dst_project,dst_region,dst_vpc,dst_vlan_attachment") {
		t.Fatalf("unexpected csv header order: %s", content)
	}
	if !strings.Contains(content, "dst_vlan_attachment_state,dst_vlan_attachment_vlanid,dst_cloud_router,dst_cloud_router_asn,dst_cloud_router_interface,dst_cloud_router_interface_ip,remote_bgp_peer,remote_bgp_peer_ip,remote_bgp_peer_asn,bgp_peering_status") {
		t.Fatalf("unexpected csv tail column order: %s", content)
	}
}

func TestRenderVPNCSV(t *testing.T) {
	data, ext, err := Render(sampleVPNReport(), FormatCSV)
	if err != nil {
		t.Fatalf("render vpn csv: %v", err)
	}
	if ext != "csv" {
		t.Fatalf("expected csv extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "org,workload,environment,src_project,src_region,src_vpn_gateway,src_vpn_gateway_type,src_cloud_router,src_cloud_router_asn,src_cloud_router_interface,src_cloud_router_interface_ip,src_vpn_tunnel,src_vpn_gateway_interface,src_vpn_gateway_ip,src_vpn_tunnel_status,bgp_peering_status,dst_vpn_tunnel,dst_vpn_gateway_interface,dst_vpn_gateway_ip,dst_vpn_tunnel_status,dst_cloud_router,dst_cloud_router_asn,dst_cloud_router_interface,dst_cloud_router_interface_ip,dst_vpn_gateway,dst_vpn_gateway_type,dst_region,dst_project") {
		t.Fatalf("unexpected vpn csv header order: %s", content)
	}
	if strings.Contains(content, "mapped") || strings.Contains(content, "remote_bgp_peer") || strings.Contains(content, "src_vpn_gateway_status") || strings.Contains(content, "dst_vpn_gateway_status") {
		t.Fatalf("unexpected vpn-only removed fields in csv output: %s", content)
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
	if !strings.Contains(content, `"macsec_enabled": true`) || !strings.Contains(content, `"macsec_keyname": "macsec-key-a"`) {
		t.Fatalf("expected source macsec data in json output, got: %s", content)
	}
	if !strings.Contains(content, `"cloud_router_asn": "64512"`) {
		t.Fatalf("expected router asn in json output, got: %s", content)
	}
	if !strings.Contains(content, `"vpc": "vpc-a"`) {
		t.Fatalf("expected vpc in json output, got: %s", content)
	}
	if !strings.Contains(content, `"remote_bgp_peer_asn": "64550"`) {
		t.Fatalf("expected peer asn in json output, got: %s", content)
	}
	if !strings.Contains(content, `"src_interconnects"`) || !strings.Contains(content, `"dst_projects"`) || !strings.Contains(content, `"dst_regions"`) {
		t.Fatalf("expected hierarchical destination data, got: %s", content)
	}
	if strings.Contains(content, `"src_project"`) || strings.Contains(content, `"src_interconnect"`) || strings.Contains(content, `"src_region"`) || strings.Contains(content, `"dst_project"`) || strings.Contains(content, `"dst_region"`) || strings.Contains(content, `"dst_vpc"`) || strings.Contains(content, `"dst_vlan_attachment"`) || strings.Contains(content, `"dst_cloud_router"`) {
		t.Fatalf("expected interconnect json leaf keys to be unprefixed, got: %s", content)
	}
}

func TestRenderVPNJSON(t *testing.T) {
	data, ext, err := Render(sampleVPNReport(), FormatJSON)
	if err != nil {
		t.Fatalf("render vpn json: %v", err)
	}
	if ext != "json" {
		t.Fatalf("expected json extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, `"vpc": "src-vpc-a"`) || !strings.Contains(content, `"vpc": "dst-vpc-a"`) {
		t.Fatalf("expected source and destination vpc fields on vpn region nodes in json output, got: %s", content)
	}
	if !strings.Contains(content, `"cloud_router": "router-src-a"`) || !strings.Contains(content, `"cloud_router": "router-a"`) {
		t.Fatalf("expected source and destination router nodes in vpn json output, got: %s", content)
	}
	if !strings.Contains(content, `"vpn_gateway_interface": "0"`) || !strings.Contains(content, `"vpn_gateway_ip": "34.0.0.1"`) || !strings.Contains(content, `"vpn_gateway_ip": "35.0.0.1"`) {
		t.Fatalf("expected vpn gateway interface/ip fields in json output, got: %s", content)
	}
	if strings.Contains(content, `"src_vpn_tunnels":[{"vpn_tunnel":"tunnel-a-1","vpn_gateway_interface":"0","vpn_gateway_ip"`) || strings.Contains(content, `"dst_vpn_tunnels":[{"vpn_tunnel":"tunnel-peer-1","vpn_gateway_interface":"0","vpn_gateway_ip"`) {
		t.Fatalf("expected vpn gateway ip to be removed from tunnel json nodes, got: %s", content)
	}
	if !strings.Contains(content, `"bgp_peering_statuses"`) || !strings.Contains(content, `"dst_vpn_tunnels"`) {
		t.Fatalf("expected vpn hierarchy to include bgp status and destination tunnel nodes, got: %s", content)
	}
	if strings.Contains(content, `"src_project"`) || strings.Contains(content, `"src_region"`) || strings.Contains(content, `"src_vpn_gateway"`) || strings.Contains(content, `"src_vpn_tunnel"`) || strings.Contains(content, `"src_cloud_router"`) || strings.Contains(content, `"dst_project"`) || strings.Contains(content, `"dst_region"`) || strings.Contains(content, `"dst_vpn_gateway"`) || strings.Contains(content, `"dst_vpn_tunnel"`) || strings.Contains(content, `"dst_cloud_router"`) || strings.Contains(content, `"src_vpc"`) || strings.Contains(content, `"dst_vpc"`) {
		t.Fatalf("expected vpn json leaf keys to be unprefixed, got: %s", content)
	}
	if strings.Contains(content, `"mapped"`) || strings.Contains(content, `"remote_bgp_peer"`) || strings.Contains(content, `"src_vpn_gateway_status"`) || strings.Contains(content, `"dst_vpn_gateway_status"`) {
		t.Fatalf("unexpected removed vpn fields in json output: %s", content)
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
	if !strings.Contains(content, "org: dbc\n|-- workload: native\n|   `-- environment: dev\n|       `-- project: src") {
		t.Fatalf("unexpected tree hierarchy: %s", content)
	}
	if !strings.Contains(content, "`-- workload: platform") {
		t.Fatalf("expected second workload branch, got: %s", content)
	}
	if !strings.Contains(content, "macsec_enabled: true, macsec_keyname: macsec-key-a") {
		t.Fatalf("expected source macsec data in tree output: %s", content)
	}
	if !strings.Contains(content, "cloud_router: router-1 [cloud_router_asn: 64512]") {
		t.Fatalf("expected router asn in tree output: %s", content)
	}
	if !strings.Contains(content, "remote_bgp_peer: peer-1 [remote_bgp_peer_ip: 169.254.1.2, remote_bgp_peer_asn: 64550, bgp_peering_status: UP]") {
		t.Fatalf("expected peer asn in tree output: %s", content)
	}
	if !strings.Contains(content, "vlan_attachment: attachment-1 [vpc: vpc-a, vlan_attachment_state: ACTIVE, vlan_attachment_vlanid: 101]") || !strings.Contains(content, "project: dst-b [mapped: false]") {
		t.Fatalf("unexpected tree output: %s", content)
	}
}

func TestRenderVPNTree(t *testing.T) {
	data, ext, err := Render(sampleVPNReport(), FormatTree)
	if err != nil {
		t.Fatalf("render vpn tree: %v", err)
	}
	if ext != "tree.txt" {
		t.Fatalf("expected tree extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "region: us-central1 [vpc: src-vpc-a]") || !strings.Contains(content, "region: us-central1 [vpc: dst-vpc-a]") {
		t.Fatalf("expected source and destination vpc fields on region nodes in vpn tree output, got: %s", content)
	}
	if !strings.Contains(content, "cloud_router: router-src-a [cloud_router_asn: 64510, cloud_router_interface: if-src-a-1, cloud_router_interface_ip: 169.254.10.1]") {
		t.Fatalf("expected source router node in vpn tree output, got: %s", content)
	}
	if !strings.Contains(content, "vpn_gateway: ha-a [vpn_gateway_type: ha, vpn_gateway_interface: 0, vpn_gateway_ip: 34.0.0.1]") || !strings.Contains(content, "vpn_gateway: ha-peer [vpn_gateway_type: ha, vpn_gateway_interface: 0, vpn_gateway_ip: 35.0.0.1]") {
		t.Fatalf("expected vpn gateway interface/ip fields on gateway nodes in tree output, got: %s", content)
	}
	if !strings.Contains(content, "vpn_tunnel: tunnel-a-1 [vpn_gateway_interface: 0, vpn_tunnel_status: ESTABLISHED]") || !strings.Contains(content, "vpn_tunnel: tunnel-peer-1 [vpn_gateway_interface: 0, vpn_tunnel_status: ESTABLISHED]") {
		t.Fatalf("expected vpn tunnel nodes to keep only gateway interface and status in tree output, got: %s", content)
	}
	if !strings.Contains(content, "bgp_peering_status: UP") || !strings.Contains(content, "cloud_router: router-a [cloud_router_asn: 64512, cloud_router_interface: if-dst-a-1, cloud_router_interface_ip: 169.254.20.1]") {
		t.Fatalf("expected status and destination router nodes in vpn tree output, got: %s", content)
	}
	if strings.Index(content, "vpn_tunnel: tunnel-a-1 [vpn_gateway_interface: 0, vpn_tunnel_status: ESTABLISHED]") > strings.Index(content, "cloud_router: router-src-a [cloud_router_asn: 64510, cloud_router_interface: if-src-a-1, cloud_router_interface_ip: 169.254.10.1]") {
		t.Fatalf("expected source tunnel to render before source router in vpn tree output, got: %s", content)
	}
	if strings.Index(content, "bgp_peering_status: UP") > strings.Index(content, "cloud_router: router-a [cloud_router_asn: 64512, cloud_router_interface: if-dst-a-1, cloud_router_interface_ip: 169.254.20.1]") || strings.Index(content, "cloud_router: router-a [cloud_router_asn: 64512, cloud_router_interface: if-dst-a-1, cloud_router_interface_ip: 169.254.20.1]") > strings.Index(content, "vpn_tunnel: tunnel-peer-1 [vpn_gateway_interface: 0, vpn_tunnel_status: ESTABLISHED]") {
		t.Fatalf("expected bgp status between source and destination routers, and destination tunnel after destination router in vpn tree output, got: %s", content)
	}
	if strings.Contains(content, "mapped:") || strings.Contains(content, "remote_bgp_peer") || strings.Contains(content, "src_vpn_gateway_status") || strings.Contains(content, "dst_vpn_gateway_status") {
		t.Fatalf("unexpected removed vpn fields in tree output: %s", content)
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
	if !strings.Contains(content, "region: us-central1") || !strings.Contains(content, "vlan_attachment: attachment-1") || !strings.Contains(content, "remote_bgp_peer: peer-1") {
		t.Fatalf("expected destination fanout details, got %s", content)
	}
	if !strings.Contains(content, "<br>") {
		t.Fatalf("expected mermaid-compatible line breaks, got %s", content)
	}
	if strings.Contains(content, "\\n") {
		t.Fatalf("unexpected escaped newline in mermaid output: %s", content)
	}
	if !strings.Contains(content, "macsec_enabled: true") || !strings.Contains(content, "macsec_keyname: macsec-key-a") {
		t.Fatalf("expected source macsec data in mermaid output, got %s", content)
	}
	if !strings.Contains(content, "cloud_router_asn: 64512") {
		t.Fatalf("expected router asn in mermaid output, got %s", content)
	}
	if !strings.Contains(content, "vpc: vpc-a") {
		t.Fatalf("expected vpc in mermaid output, got %s", content)
	}
	if !strings.Contains(content, "remote_bgp_peer_asn: 64550") {
		t.Fatalf("expected peer asn in mermaid output, got %s", content)
	}
	if !strings.Contains(content, "bgp_peering_status: UP") {
		t.Fatalf("expected dedicated bgp status node in mermaid output, got %s", content)
	}
	if strings.Contains(content, "remote_bgp_peer: peer-1<br>remote_bgp_peer_ip: 169.254.1.2<br>bgp_peering_status: UP") {
		t.Fatalf("expected bgp status to be outside the remote peer node, got %s", content)
	}
	if countSubstring(content, "project: dst-a") != 1 {
		t.Fatalf("expected one shared destination project node, got %d in %s", countSubstring(content, "project: dst-a"), content)
	}
	if countSubstring(content, "region: us-central1") != 1 {
		t.Fatalf("expected one shared destination region node, got %d in %s", countSubstring(content, "region: us-central1"), content)
	}
	if countSubstring(content, "vpc: vpc-a") != 1 {
		t.Fatalf("expected one shared destination vpc label, got %d in %s", countSubstring(content, "vpc: vpc-a"), content)
	}
	if countSubstring(content, "environment: dev") != 1 {
		t.Fatalf("expected one shared environment node, got %d in %s", countSubstring(content, "environment: dev"), content)
	}
	if countSubstring(content, "project: src") != 1 {
		t.Fatalf("expected one shared source project node, got %d in %s", countSubstring(content, "project: src"), content)
	}
	if countSubstring(content, "interconnect: ic-1") != 1 {
		t.Fatalf("expected one shared ic-1 node, got %d in %s", countSubstring(content, "interconnect: ic-1"), content)
	}
}

func TestRenderHTML(t *testing.T) {
	data, ext, err := Render(sampleReport(), FormatHTML)
	if err != nil {
		t.Fatalf("render html: %v", err)
	}
	if ext != "html" {
		t.Fatalf("expected html extension, got %q", ext)
	}
	content := string(data)
	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Fatalf("expected html doctype, got %s", content)
	}
	if !strings.Contains(content, "<title>NetMap | Interconnect+DBC+All | HTML-Generated Mermaid Report</title>") {
		t.Fatalf("expected selector-based html title, got %s", content)
	}
	if !strings.Contains(content, `class="summary-label">Type`) || !strings.Contains(content, `class="summary-label">Org`) || !strings.Contains(content, `class="summary-label">Workload / Environment`) {
		t.Fatalf("expected selector summary headings in html output, got %s", content)
	}
	if !strings.Contains(content, `class="summary-value">Interconnect`) || !strings.Contains(content, `class="summary-value">DBC`) || !strings.Contains(content, `class="summary-value">All`) {
		t.Fatalf("expected selector summary values with All fallback in html output, got %s", content)
	}
	if !strings.Contains(content, `<h1>NetMap | Interconnect+DBC+All | HTML-Generated Mermaid Report</h1>`) {
		t.Fatalf("expected visible html title, got %s", content)
	}
	if strings.Contains(content, "netmap interconnect: src to") {
		t.Fatalf("expected old source-to-target html title to be removed, got %s", content)
	}
	if !strings.Contains(content, "flowchart LR") || !strings.Contains(content, "remote_bgp_peer: peer-1") {
		t.Fatalf("expected embedded mermaid graph source, got %s", content)
	}
	if !strings.Contains(content, "mermaid.initialize") || !strings.Contains(content, "mermaid.run") {
		t.Fatalf("expected embedded mermaid bootstrap, got %s", content)
	}
	if !strings.Contains(content, "The MIT License (MIT)") {
		t.Fatalf("expected embedded mermaid license notice, got %s", content)
	}
	if !strings.Contains(content, `rel="icon" type="image/svg+xml" href="data:image/svg+xml,`) || !strings.Contains(content, "o-o") || !strings.Contains(content, "%7Cx%7C") {
		t.Fatalf("expected inline ascii mesh favicon in html output, got %s", content)
	}
	if strings.Contains(content, "cdn.jsdelivr.net") || strings.Contains(content, "https://mermaid.live") {
		t.Fatalf("expected offline html without external mermaid references, got %s", content)
	}
}

func TestRenderHTMLUsesExactSelectorValuesWhenProvided(t *testing.T) {
	report := sampleVPNReport()
	report.Selectors = model.Selectors{
		Org:         "dbc",
		Workload:    "native",
		Environment: "dev",
	}

	data, _, err := Render(report, FormatHTML)
	if err != nil {
		t.Fatalf("render html: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "<title>NetMap | Vpn+DBC+Native+DEV | HTML-Generated Mermaid Report</title>") {
		t.Fatalf("expected exact selector values in html title, got %s", content)
	}
	if !strings.Contains(content, `class="summary-value">Vpn`) || !strings.Contains(content, `class="summary-value">DBC`) || !strings.Contains(content, `class="summary-value">DEV`) || !strings.Contains(content, `class="summary-value">Native`) {
		t.Fatalf("expected exact selector values in html summary, got %s", content)
	}
	if !strings.Contains(content, `class="summary-label">Environment`) || !strings.Contains(content, `class="summary-label">Workload`) {
		t.Fatalf("expected separate environment and workload headings for exact selectors, got %s", content)
	}
	if strings.Contains(content, `class="summary-label">Workload / Environment`) {
		t.Fatalf("did not expect combined workload/environment heading for exact selectors, got %s", content)
	}
	if strings.Contains(content, `class="summary-value">All`) {
		t.Fatalf("did not expect All fallback for exact selector values, got %s", content)
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
				DstVPC:                  "shared-vpc",
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
				DstVPC:                  "shared-vpc",
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
				DstVPC:                  "shared-vpc",
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
	if countSubstring(content, "project: src") != 1 {
		t.Fatalf("expected one shared source project node, got %d in %s", countSubstring(content, "project: src"), content)
	}
	if countSubstring(content, "interconnect: ic-native") != 1 {
		t.Fatalf("expected one shared dedicated interconnect node, got %d in %s", countSubstring(content, "interconnect: ic-native"), content)
	}
	if countSubstring(content, "region: us-central1") != 1 {
		t.Fatalf("expected one shared destination region node, got %d in %s", countSubstring(content, "region: us-central1"), content)
	}
	if countSubstring(content, "vpc: shared-vpc") != 1 {
		t.Fatalf("expected one shared destination vpc label, got %d in %s", countSubstring(content, "vpc: shared-vpc"), content)
	}
	if countSubstring(content, "project: project-") != 3 {
		t.Fatalf("expected three distinct destination project nodes, got %d in %s", countSubstring(content, "project: project-"), content)
	}
}

func TestRenderMermaidAddsSeparateVPCNodeWhenRegionHasMultipleVPCs(t *testing.T) {
	report := model.Report{
		Type:          "interconnect",
		SourceProject: "src",
		Selectors: model.Selectors{
			Org: "dbc",
		},
		Items: []model.MappingItem{
			{
				Org:               "dbc",
				Workload:          "native",
				Environment:       "dev",
				SrcProject:        "src",
				SrcInterconnect:   "ic-1",
				Mapped:            true,
				SrcRegion:         "global",
				SrcState:          "ACTIVE",
				DstProject:        "project-a",
				DstRegion:         "us-central1",
				DstVPC:            "vpc-a",
				DstVLANAttachment: "attachment-a",
				DstCloudRouter:    "router-a",
				DstCloudRouterASN: "64520",
				BGPPeeringStatus:  "UP",
				RemoteBGPPeer:     "peer-a",
				RemoteBGPPeerIP:   "169.254.10.2",
				RemoteBGPPeerASN:  "64560",
			},
			{
				Org:               "dbc",
				Workload:          "native",
				Environment:       "dev",
				SrcProject:        "src",
				SrcInterconnect:   "ic-2",
				Mapped:            true,
				SrcRegion:         "global",
				SrcState:          "ACTIVE",
				DstProject:        "project-b",
				DstRegion:         "us-central1",
				DstVPC:            "vpc-b",
				DstVLANAttachment: "attachment-b",
				DstCloudRouter:    "router-b",
				DstCloudRouterASN: "64521",
				BGPPeeringStatus:  "UP",
				RemoteBGPPeer:     "peer-b",
				RemoteBGPPeerIP:   "169.254.20.2",
				RemoteBGPPeerASN:  "64561",
			},
		},
	}

	data, _, err := Render(report, FormatMermaid)
	if err != nil {
		t.Fatalf("render mermaid: %v", err)
	}

	content := string(data)
	if countSubstring(content, "region: us-central1") != 1 {
		t.Fatalf("expected one shared destination region node, got %d in %s", countSubstring(content, "region: us-central1"), content)
	}
	if countSubstring(content, "vpc: vpc-a") != 1 || countSubstring(content, "vpc: vpc-b") != 1 {
		t.Fatalf("expected separate dst_vpc nodes, got %s", content)
	}
}

func TestRenderVPNMermaidCollapsesProjectRegionPairsIntoSeparateNodes(t *testing.T) {
	data, ext, err := Render(sampleVPNReport(), FormatMermaid)
	if err != nil {
		t.Fatalf("render vpn mermaid: %v", err)
	}
	if ext != "mmd" {
		t.Fatalf("expected mmd extension, got %q", ext)
	}

	content := string(data)
	if !strings.Contains(content, "vpn_gateway: ha-a") || !strings.Contains(content, "vpn_gateway: ha-peer") {
		t.Fatalf("expected vpn gateway labels, got %s", content)
	}
	if !strings.Contains(content, "vpc: src-vpc-a") || !strings.Contains(content, "vpc: dst-vpc-a") {
		t.Fatalf("expected source and destination vpc values on vpn region nodes, got %s", content)
	}
	if !strings.Contains(content, "vpn_gateway_interface: 0") || !strings.Contains(content, "vpn_gateway_ip: 34.0.0.1") || !strings.Contains(content, "vpn_gateway_ip: 35.0.0.1") {
		t.Fatalf("expected vpn gateway labels to include gateway interface/ip fields, got %s", content)
	}
	if !strings.Contains(content, "vpn_tunnel: tunnel-a-1<br>vpn_gateway_interface: 0<br>vpn_tunnel_status: ESTABLISHED") || !strings.Contains(content, "vpn_tunnel: tunnel-peer-1<br>vpn_gateway_interface: 0<br>vpn_tunnel_status: ESTABLISHED") {
		t.Fatalf("expected vpn tunnel labels to keep gateway interface but omit gateway ip, got %s", content)
	}
	if strings.Contains(content, "vpn_tunnel: tunnel-a-1<br>vpn_gateway_interface: 0<br>vpn_gateway_ip: 34.0.0.1") || strings.Contains(content, "vpn_tunnel: tunnel-peer-1<br>vpn_gateway_interface: 0<br>vpn_gateway_ip: 35.0.0.1") {
		t.Fatalf("expected vpn tunnel labels to omit gateway ip, got %s", content)
	}
	if !strings.Contains(content, "cloud_router: router-src-a") || !strings.Contains(content, "cloud_router: router-a") {
		t.Fatalf("expected dedicated vpn router nodes, got %s", content)
	}
	if countSubstring(content, "project: src-a") != 1 {
		t.Fatalf("expected one shared source project node, got %d in %s", countSubstring(content, "project: src-a"), content)
	}
	if countSubstring(content, "region: us-central1<br>vpc: src-vpc-a") != 1 {
		t.Fatalf("expected one shared source region node for src-a/us-central1/src-vpc-a, got %d in %s", countSubstring(content, "region: us-central1<br>vpc: src-vpc-a"), content)
	}
	if countSubstring(content, "project: dst-a") != 2 {
		t.Fatalf("expected one destination project node per source tunnel pair, got %d in %s", countSubstring(content, "project: dst-a"), content)
	}
	if countSubstring(content, "region: us-central1<br>vpc: dst-vpc-a") != 2 {
		t.Fatalf("expected one destination region node per source tunnel pair for dst-a/us-central1/dst-vpc-a, got %d in %s", countSubstring(content, "region: us-central1<br>vpc: dst-vpc-a"), content)
	}
	if !strings.Contains(content, "bgp_peering_status: UP") {
		t.Fatalf("expected dedicated bgp status node in vpn mermaid output, got %s", content)
	}
	if strings.Index(content, "vpn_tunnel: tunnel-a-1") > strings.Index(content, "cloud_router: router-src-a") {
		t.Fatalf("expected source tunnel to appear before source router in vpn mermaid output, got %s", content)
	}
	if strings.Index(content, "bgp_peering_status: UP") > strings.Index(content, "cloud_router: router-a") || strings.Index(content, "cloud_router: router-a") > strings.Index(content, "vpn_tunnel: tunnel-peer-1") {
		t.Fatalf("expected bgp status to connect source and destination routers, with destination tunnel after destination router, got %s", content)
	}
	if strings.Contains(content, "mapped:") || strings.Contains(content, "remote_bgp_peer") {
		t.Fatalf("unexpected remote peer fields in vpn mermaid output: %s", content)
	}
	if strings.Contains(content, "src_interconnect:") || strings.Contains(content, "src_vpn_gateway:") || strings.Contains(content, "dst_vpn_gateway:") || strings.Contains(content, "src_cloud_router:") || strings.Contains(content, "dst_cloud_router:") || strings.Contains(content, "src_vpn_tunnel:") || strings.Contains(content, "dst_vpn_tunnel:") || strings.Contains(content, "src_region:") || strings.Contains(content, "dst_region:") {
		t.Fatalf("unexpected interconnect nodes in vpn mermaid output: %s", content)
	}
}

func TestRenderVPNMermaidKeepsSameRegionLabelsSeparateAcrossDifferentProjects(t *testing.T) {
	report := model.Report{
		Type: "vpn",
		Selectors: model.Selectors{
			Org: "dbc",
		},
		Items: []model.MappingItem{
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-a",
				SrcVPC:                    "src-vpc-a",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-a",
				SrcVPNGatewayType:         "ha",
				SrcCloudRouter:            "router-src-a",
				SrcCloudRouterASN:         "64510",
				SrcCloudRouterInterface:   "if-src-a",
				SrcCloudRouterInterfaceIP: "169.254.1.1",
				SrcVPNTunnel:              "tunnel-a",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				Mapped:                    true,
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-a",
				DstVPNGateway:             "dst-ha-a",
				DstVPNGatewayType:         "ha",
				DstVPNTunnel:              "dst-tunnel-a",
				DstVPNTunnelStatus:        "ESTABLISHED",
				DstCloudRouter:            "router-dst-a",
				DstCloudRouterASN:         "64520",
				DstCloudRouterInterface:   "if-dst-a",
				DstCloudRouterInterfaceIP: "169.254.1.2",
				BGPPeeringStatus:          "UP",
			},
			{
				Org:                       "dbc",
				Workload:                  "platform",
				Environment:               "dev",
				SrcProject:                "src-b",
				SrcVPC:                    "src-vpc-b",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-b",
				SrcVPNGatewayType:         "ha",
				SrcCloudRouter:            "router-src-b",
				SrcCloudRouterASN:         "64511",
				SrcCloudRouterInterface:   "if-src-b",
				SrcCloudRouterInterfaceIP: "169.254.2.1",
				SrcVPNTunnel:              "tunnel-b",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				Mapped:                    true,
				DstProject:                "dst-b",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-b",
				DstVPNGateway:             "dst-ha-b",
				DstVPNGatewayType:         "ha",
				DstVPNTunnel:              "dst-tunnel-b",
				DstVPNTunnelStatus:        "ESTABLISHED",
				DstCloudRouter:            "router-dst-b",
				DstCloudRouterASN:         "64521",
				DstCloudRouterInterface:   "if-dst-b",
				DstCloudRouterInterfaceIP: "169.254.2.2",
				BGPPeeringStatus:          "UP",
			},
		},
	}

	data, _, err := Render(report, FormatMermaid)
	if err != nil {
		t.Fatalf("render vpn mermaid: %v", err)
	}

	content := string(data)
	if countSubstring(content, "region: us-central1<br>vpc: src-vpc-") != 2 {
		t.Fatalf("expected separate source region nodes per src project/vpc, got %d in %s", countSubstring(content, "region: us-central1<br>vpc: src-vpc-"), content)
	}
	if countSubstring(content, "region: us-central1<br>vpc: dst-vpc-") != 2 {
		t.Fatalf("expected separate destination region nodes per dst project/vpc, got %d in %s", countSubstring(content, "region: us-central1<br>vpc: dst-vpc-"), content)
	}
}

func TestRenderVPNMermaidKeepsDestinationBranchScopedPerSourceTunnel(t *testing.T) {
	report := model.Report{
		Type: "vpn",
		Selectors: model.Selectors{
			Org: "dbc",
		},
		Items: []model.MappingItem{
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-a",
				SrcVPC:                    "src-vpc-a",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-a",
				SrcVPNGatewayType:         "ha",
				SrcCloudRouter:            "router-src-a",
				SrcCloudRouterASN:         "64510",
				SrcCloudRouterInterface:   "if-src-a-1",
				SrcCloudRouterInterfaceIP: "169.254.1.1",
				SrcVPNTunnel:              "tunnel-a-1",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				Mapped:                    true,
				BGPPeeringStatus:          "UP",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-a",
				DstVPNGateway:             "ha-peer",
				DstVPNGatewayType:         "ha",
				DstVPNTunnel:              "tunnel-peer-shared",
				DstVPNTunnelStatus:        "ESTABLISHED",
				DstCloudRouter:            "router-dst",
				DstCloudRouterASN:         "64520",
				DstCloudRouterInterface:   "if-dst-shared",
				DstCloudRouterInterfaceIP: "169.254.2.1",
			},
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-a",
				SrcVPC:                    "src-vpc-a",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-a",
				SrcVPNGatewayType:         "ha",
				SrcCloudRouter:            "router-src-a",
				SrcCloudRouterASN:         "64510",
				SrcCloudRouterInterface:   "if-src-a-2",
				SrcCloudRouterInterfaceIP: "169.254.1.5",
				SrcVPNTunnel:              "tunnel-a-2",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				Mapped:                    true,
				BGPPeeringStatus:          "UP",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-a",
				DstVPNGateway:             "ha-peer",
				DstVPNGatewayType:         "ha",
				DstVPNTunnel:              "tunnel-peer-shared",
				DstVPNTunnelStatus:        "ESTABLISHED",
				DstCloudRouter:            "router-dst",
				DstCloudRouterASN:         "64520",
				DstCloudRouterInterface:   "if-dst-shared",
				DstCloudRouterInterfaceIP: "169.254.2.1",
			},
		},
	}

	data, _, err := Render(report, FormatMermaid)
	if err != nil {
		t.Fatalf("render vpn mermaid: %v", err)
	}

	content := string(data)
	if countSubstring(content, "vpn_tunnel: tunnel-a-1") != 1 || countSubstring(content, "vpn_tunnel: tunnel-a-2") != 1 {
		t.Fatalf("expected both source tunnel nodes, got %s", content)
	}
	if countSubstring(content, "vpn_tunnel: tunnel-peer-shared") != 2 {
		t.Fatalf("expected one destination tunnel node per source tunnel pair, got %d in %s", countSubstring(content, "vpn_tunnel: tunnel-peer-shared"), content)
	}
	if countSubstring(content, "cloud_router: router-dst") != 2 {
		t.Fatalf("expected destination router nodes to stay pair-scoped, got %d in %s", countSubstring(content, "cloud_router: router-dst"), content)
	}
	if countSubstring(content, "vpn_gateway: ha-peer") != 2 {
		t.Fatalf("expected destination gateway nodes to stay pair-scoped, got %d in %s", countSubstring(content, "vpn_gateway: ha-peer"), content)
	}
}

func TestRenderVPNMermaidCollapsesIdenticalDestinationGatewayRegionProjectWithinSourceBranch(t *testing.T) {
	report := model.Report{
		Type: "vpn",
		Selectors: model.Selectors{
			Org: "dbc",
		},
		Items: []model.MappingItem{
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-a",
				SrcVPC:                    "src-vpc-a",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-a",
				SrcVPNGatewayType:         "ha",
				SrcVPNTunnel:              "tunnel-a-1",
				SrcVPNGatewayInterface:    "0",
				SrcVPNGatewayIP:           "34.0.0.1",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				SrcCloudRouter:            "router-src-a",
				SrcCloudRouterASN:         "64510",
				SrcCloudRouterInterface:   "if-src-a-1",
				SrcCloudRouterInterfaceIP: "169.254.1.1",
				Mapped:                    true,
				BGPPeeringStatus:          "UP",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-a",
				DstVPNGateway:             "ha-peer",
				DstVPNGatewayType:         "ha",
				DstCloudRouter:            "router-dst",
				DstCloudRouterASN:         "64520",
				DstCloudRouterInterface:   "if-dst-shared",
				DstCloudRouterInterfaceIP: "169.254.2.1",
				DstVPNTunnel:              "tunnel-peer-1",
				DstVPNGatewayInterface:    "0",
				DstVPNGatewayIP:           "35.0.0.1",
				DstVPNTunnelStatus:        "ESTABLISHED",
			},
			{
				Org:                       "dbc",
				Workload:                  "native",
				Environment:               "dev",
				SrcProject:                "src-a",
				SrcVPC:                    "src-vpc-a",
				SrcRegion:                 "us-central1",
				SrcVPNGateway:             "ha-a",
				SrcVPNGatewayType:         "ha",
				SrcVPNTunnel:              "tunnel-a-1",
				SrcVPNGatewayInterface:    "0",
				SrcVPNGatewayIP:           "34.0.0.1",
				SrcVPNTunnelStatus:        "ESTABLISHED",
				SrcCloudRouter:            "router-src-a",
				SrcCloudRouterASN:         "64510",
				SrcCloudRouterInterface:   "if-src-a-1",
				SrcCloudRouterInterfaceIP: "169.254.1.1",
				Mapped:                    true,
				BGPPeeringStatus:          "UP",
				DstProject:                "dst-a",
				DstRegion:                 "us-central1",
				DstVPC:                    "dst-vpc-a",
				DstVPNGateway:             "ha-peer",
				DstVPNGatewayType:         "ha",
				DstCloudRouter:            "router-dst",
				DstCloudRouterASN:         "64520",
				DstCloudRouterInterface:   "if-dst-shared",
				DstCloudRouterInterfaceIP: "169.254.2.1",
				DstVPNTunnel:              "tunnel-peer-2",
				DstVPNGatewayInterface:    "1",
				DstVPNGatewayIP:           "35.0.0.2",
				DstVPNTunnelStatus:        "ESTABLISHED",
			},
		},
	}

	data, _, err := Render(report, FormatMermaid)
	if err != nil {
		t.Fatalf("render vpn mermaid: %v", err)
	}

	content := string(data)
	if countSubstring(content, "cloud_router: router-dst") != 1 {
		t.Fatalf("expected one destination router node within the same source branch, got %d in %s", countSubstring(content, "cloud_router: router-dst"), content)
	}
	if countSubstring(content, "vpn_tunnel: tunnel-peer-1") != 1 || countSubstring(content, "vpn_tunnel: tunnel-peer-2") != 1 {
		t.Fatalf("expected separate destination tunnel nodes within the source branch, got %s", content)
	}
	if countSubstring(content, "vpn_gateway: ha-peer") != 2 {
		t.Fatalf("expected one destination gateway node per gateway interface within the source branch, got %d in %s", countSubstring(content, "vpn_gateway: ha-peer"), content)
	}
	if countSubstring(content, "region: us-central1<br>vpc: dst-vpc-a") != 1 {
		t.Fatalf("expected one shared destination region/vpc node within the source branch, got %d in %s", countSubstring(content, "region: us-central1<br>vpc: dst-vpc-a"), content)
	}
	if countSubstring(content, "project: dst-a") != 1 {
		t.Fatalf("expected one shared destination project node within the source branch, got %d in %s", countSubstring(content, "project: dst-a"), content)
	}
}

func countSubstring(content, needle string) int {
	return strings.Count(content, needle)
}
