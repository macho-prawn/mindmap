package render

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"netmap/internal/model"
)

const (
	FormatMermaid = "mermaid"
	FormatHTML    = "html"
	FormatCSV     = "csv"
	FormatTSV     = "tsv"
	FormatJSON    = "json"
	FormatTree    = "tree"
)

var interconnectSeparatedHeader = []string{
	"org",
	"workload",
	"environment",
	"src_project",
	"src_interconnect",
	"mapped",
	"src_region",
	"src_state",
	"src_macsec_enabled",
	"src_macsec_keyname",
	"dst_project",
	"dst_region",
	"dst_vpc",
	"dst_vlan_attachment",
	"dst_vlan_attachment_state",
	"dst_vlan_attachment_vlanid",
	"dst_cloud_router",
	"dst_cloud_router_asn",
	"dst_cloud_router_interface",
	"dst_cloud_router_interface_ip",
	"remote_bgp_peer",
	"remote_bgp_peer_ip",
	"remote_bgp_peer_asn",
	"bgp_peering_status",
}

var vpnSeparatedHeader = []string{
	"org",
	"workload",
	"environment",
	"src_project",
	"src_region",
	"src_vpn_gateway",
	"src_vpn_gateway_type",
	"src_cloud_router",
	"src_cloud_router_asn",
	"src_cloud_router_interface",
	"src_cloud_router_interface_ip",
	"src_routes",
	"src_vpn_tunnel",
	"src_vpn_gateway_interface",
	"src_vpn_gateway_ip",
	"src_vpn_tunnel_status",
	"bgp_peering_status",
	"dst_vpn_tunnel",
	"dst_vpn_gateway_interface",
	"dst_vpn_gateway_ip",
	"dst_vpn_tunnel_status",
	"dst_cloud_router",
	"dst_cloud_router_asn",
	"dst_cloud_router_interface",
	"dst_cloud_router_interface_ip",
	"dst_routes",
	"dst_vpn_gateway",
	"dst_vpn_gateway_type",
	"dst_region",
	"dst_project",
}

func Render(report model.Report, format string) ([]byte, string, error) {
	switch format {
	case "", FormatMermaid:
		return renderMermaid(report), "mmd", nil
	case FormatHTML:
		data, err := renderHTML(report)
		return data, "html", err
	case FormatCSV:
		data, err := renderSeparated(report, ',')
		return data, "csv", err
	case FormatTSV:
		data, err := renderSeparated(report, '\t')
		return data, "tsv", err
	case FormatJSON:
		data, err := renderJSON(report)
		return data, "json", err
	case FormatTree:
		return renderTree(report), "tree.txt", nil
	default:
		return nil, "", fmt.Errorf("unsupported output format %q", format)
	}
}

func renderSeparated(report model.Report, delimiter rune) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = delimiter
	header := interconnectSeparatedHeader
	if report.Type == "vpn" {
		header = vpnSeparatedHeader
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, item := range normalizedItems(report) {
		record := interconnectSeparatedRecord(item)
		if report.Type == "vpn" {
			record = vpnSeparatedRecord(item)
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func interconnectSeparatedRecord(item model.MappingItem) []string {
	return []string{
		item.Org,
		item.Workload,
		item.Environment,
		item.SrcProject,
		item.SrcInterconnect,
		fmt.Sprintf("%t", item.Mapped),
		item.SrcRegion,
		item.SrcState,
		fmt.Sprintf("%t", item.SrcMacsecEnabled),
		item.SrcMacsecKeyName,
		item.DstProject,
		item.DstRegion,
		item.DstVPC,
		item.DstVLANAttachment,
		item.DstVLANAttachmentState,
		item.DstVLANAttachmentVLANID,
		item.DstCloudRouter,
		item.DstCloudRouterASN,
		item.DstCloudRouterInterface,
		item.DstCloudRouterInterfaceIP,
		item.RemoteBGPPeer,
		item.RemoteBGPPeerIP,
		item.RemoteBGPPeerASN,
		item.BGPPeeringStatus,
	}
}

func vpnSeparatedRecord(item model.MappingItem) []string {
	return []string{
		item.Org,
		item.Workload,
		item.Environment,
		item.SrcProject,
		item.SrcRegion,
		item.SrcVPNGateway,
		item.SrcVPNGatewayType,
		item.SrcCloudRouter,
		item.SrcCloudRouterASN,
		item.SrcCloudRouterInterface,
		item.SrcCloudRouterInterfaceIP,
		item.SrcRoutes,
		item.SrcVPNTunnel,
		item.SrcVPNGatewayInterface,
		item.SrcVPNGatewayIP,
		item.SrcVPNTunnelStatus,
		item.BGPPeeringStatus,
		item.DstVPNTunnel,
		item.DstVPNGatewayInterface,
		item.DstVPNGatewayIP,
		item.DstVPNTunnelStatus,
		item.DstCloudRouter,
		item.DstCloudRouterASN,
		item.DstCloudRouterInterface,
		item.DstCloudRouterInterfaceIP,
		item.DstRoutes,
		item.DstVPNGateway,
		item.DstVPNGatewayType,
		item.DstRegion,
		item.DstProject,
	}
}

func renderJSON(report model.Report) ([]byte, error) {
	if report.Type == "vpn" {
		return json.MarshalIndent(buildVPNJSONReport(report), "", "  ")
	}
	return json.MarshalIndent(buildJSONReport(report), "", "  ")
}

func renderTree(report model.Report) []byte {
	if report.Type == "vpn" {
		return renderVPNTree(report)
	}
	hierarchy := buildHierarchy(report)
	var b strings.Builder
	for idx, org := range hierarchy.Orgs {
		if idx > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "org: %s\n", valueOrUnknown(org.Name))
		for workloadIdx, workload := range org.Workloads {
			drawTreeWorkload(&b, workload, "", workloadIdx == len(org.Workloads)-1)
		}
	}
	return []byte(b.String())
}

func renderMermaid(report model.Report) []byte {
	if report.Type == "vpn" {
		return renderVPNMermaid(report)
	}
	items := normalizedItems(report)
	var b strings.Builder
	b.WriteString("flowchart LR\n")

	regionVPCs := buildRegionVPCs(items)
	seen := make(map[string]struct{})
	for _, item := range items {
		orgID := mermaidID("org-" + item.Org)
		workloadID := mermaidID("workload-" + item.Org + "-" + item.Workload)
		environmentID := mermaidID("environment-" + item.Org + "-" + item.Environment)
		srcID := mermaidID("src-" + item.Org + "-" + item.SrcProject)
		interconnectID := mermaidID("ic-" + item.Org + "-" + item.SrcProject + "-" + item.SrcInterconnect)
		dstProjectID := mermaidID("dst-project-" + item.Org + "-" + item.SrcProject + "-" + item.DstProject)

		defineMermaidNode(&b, seen, orgID, "org: "+valueOrUnknown(item.Org))
		linkIfMissing(&b, seen, orgID, workloadID, "workload: "+valueOrUnknown(item.Workload))
		linkIfMissing(&b, seen, workloadID, environmentID, "environment: "+valueOrUnknown(item.Environment))
		linkIfMissing(&b, seen, environmentID, srcID, "src_project: "+valueOrUnknown(item.SrcProject))
		linkIfMissing(&b, seen, srcID, interconnectID, interconnectItemLabel(item))
		linkIfMissing(&b, seen, interconnectID, dstProjectID, destinationProjectItemLabel(item))
		if !item.Mapped {
			unmappedID := mermaidID("unmapped-" + item.Org + "-" + item.SrcProject + "-" + item.DstProject)
			linkIfMissing(&b, seen, dstProjectID, unmappedID, "unmapped")
			continue
		}

		regionKey := mermaidRegionKey(item)
		regionVPC := regionVPCs[regionKey]
		regionID := mermaidID("dst-region-" + item.Org + "-" + item.SrcProject + "-" + item.DstRegion)
		vpcID := mermaidID("dst-vpc-" + item.Org + "-" + item.SrcProject + "-" + item.DstRegion + "-" + item.DstVPC)
		attachmentID := mermaidID("attachment-" + item.Org + "-" + item.SrcProject + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment)
		routerID := mermaidID("router-" + item.Org + "-" + item.SrcProject + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment + "-" + item.DstCloudRouter)
		interfaceID := mermaidID("interface-" + item.Org + "-" + item.SrcProject + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment + "-" + item.DstCloudRouter + "-" + item.DstCloudRouterInterface)
		statusID := mermaidID("bgp-status-" + item.Org + "-" + item.SrcProject + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment + "-" + item.DstCloudRouter + "-" + item.DstCloudRouterInterface + "-" + item.RemoteBGPPeer + "-" + item.RemoteBGPPeerIP + "-" + item.BGPPeeringStatus)
		peerID := mermaidID("peer-" + item.Org + "-" + item.SrcProject + "-" + item.DstProject + "-" + item.DstRegion + "-" + item.DstVLANAttachment + "-" + item.DstCloudRouter + "-" + item.DstCloudRouterInterface + "-" + item.RemoteBGPPeer + "-" + item.RemoteBGPPeerIP + "-" + item.RemoteBGPPeerASN)

		linkIfMissing(&b, seen, dstProjectID, regionID, destinationRegionItemLabel(item, regionVPC))
		attachmentParentID := regionID
		if !regionVPC.Shared {
			linkIfMissing(&b, seen, regionID, vpcID, destinationVPCItemLabel(item))
			attachmentParentID = vpcID
		}
		linkIfMissing(&b, seen, attachmentParentID, attachmentID, attachmentItemLabel(item))
		linkIfMissing(&b, seen, attachmentID, routerID, routerItemLabel(item))
		if hasInterfaceItem(item) {
			linkIfMissing(&b, seen, routerID, interfaceID, interfaceItemLabel(item))
		}
		if hasStatusItem(item) {
			parentID := routerID
			if hasInterfaceItem(item) {
				parentID = interfaceID
			}
			linkIfMissing(&b, seen, parentID, statusID, peeringStatusItemLabel(item))
			parentID = statusID
			if hasPeerItem(item) {
				linkIfMissing(&b, seen, parentID, peerID, peerItemLabel(item))
			}
		}
	}
	return []byte(b.String())
}

type jsonReport struct {
	Type string      `json:"type"`
	Org  jsonOrgNode `json:"org"`
}

type jsonOrgNode struct {
	Name      string             `json:"name"`
	Workloads []jsonWorkloadNode `json:"workloads,omitempty"`
}

type jsonWorkloadNode struct {
	Name         string                `json:"name"`
	Environments []jsonEnvironmentNode `json:"environments,omitempty"`
}

type jsonEnvironmentNode struct {
	Name        string           `json:"name"`
	SrcProjects []jsonSourceNode `json:"src_projects,omitempty"`
}

type jsonSourceNode struct {
	SrcProject      string                 `json:"project"`
	SrcInterconnect []jsonInterconnectNode `json:"src_interconnects,omitempty"`
}

type jsonInterconnectNode struct {
	SrcInterconnect  string                `json:"interconnect"`
	Mapped           bool                  `json:"mapped"`
	SrcRegion        string                `json:"region"`
	SrcState         string                `json:"state"`
	SrcMacsecEnabled bool                  `json:"macsec_enabled"`
	SrcMacsecKeyName string                `json:"macsec_keyname"`
	DstProjects      []jsonDestinationNode `json:"dst_projects,omitempty"`
}

type jsonDestinationNode struct {
	DstProject string           `json:"project"`
	Mapped     bool             `json:"mapped"`
	DstRegions []jsonRegionNode `json:"dst_regions,omitempty"`
}

type jsonRegionNode struct {
	DstRegion          string               `json:"region"`
	DstVLANAttachments []jsonAttachmentNode `json:"dst_vlan_attachments,omitempty"`
}

type jsonAttachmentNode struct {
	DstVPC                    string `json:"vpc"`
	DstVLANAttachment         string `json:"vlan_attachment"`
	DstVLANAttachmentState    string `json:"vlan_attachment_state"`
	DstVLANAttachmentVLANID   string `json:"vlan_attachment_vlanid"`
	DstCloudRouter            string `json:"cloud_router"`
	DstCloudRouterASN         string `json:"cloud_router_asn"`
	DstCloudRouterInterface   string `json:"cloud_router_interface"`
	DstCloudRouterInterfaceIP string `json:"cloud_router_interface_ip"`
	RemoteBGPPeer             string `json:"remote_bgp_peer"`
	RemoteBGPPeerIP           string `json:"remote_bgp_peer_ip"`
	RemoteBGPPeerASN          string `json:"remote_bgp_peer_asn"`
	BGPPeeringStatus          string `json:"bgp_peering_status"`
}

type hierarchy struct {
	Orgs []orgGroup
}

type orgGroup struct {
	Name      string
	Workloads []workloadGroup
}

type workloadGroup struct {
	Name         string
	Environments []environmentGroup
}

type environmentGroup struct {
	Name        string
	SrcProjects []sourceGroup
}

type sourceGroup struct {
	SrcProject    string
	Interconnects []interconnectGroup
}

type interconnectGroup struct {
	SrcInterconnect  string
	Mapped           bool
	SrcRegion        string
	SrcState         string
	SrcMacsecEnabled bool
	SrcMacsecKeyName string
	DstProjects      []destinationGroup
}

type destinationGroup struct {
	DstProject string
	Mapped     bool
	DstRegions []regionGroup
}

type regionGroup struct {
	DstRegion          string
	DstVLANAttachments []attachmentGroup
}

type attachmentGroup struct {
	DstVPC                    string
	DstVLANAttachment         string
	DstVLANAttachmentState    string
	DstVLANAttachmentVLANID   string
	DstCloudRouter            string
	DstCloudRouterASN         string
	DstCloudRouterInterface   string
	DstCloudRouterInterfaceIP string
	RemoteBGPPeer             string
	RemoteBGPPeerIP           string
	RemoteBGPPeerASN          string
	BGPPeeringStatus          string
}

func buildJSONReport(report model.Report) jsonReport {
	hierarchy := buildHierarchy(report)
	root := jsonOrgNode{Name: valueOrUnknown(report.Selectors.Org)}
	if len(hierarchy.Orgs) > 0 {
		root = buildJSONOrg(hierarchy.Orgs[0])
	}
	return jsonReport{
		Type: report.Type,
		Org:  root,
	}
}

func buildJSONOrg(group orgGroup) jsonOrgNode {
	node := jsonOrgNode{Name: valueOrUnknown(group.Name)}
	for _, workload := range group.Workloads {
		workloadNode := jsonWorkloadNode{Name: valueOrUnknown(workload.Name)}
		for _, environment := range workload.Environments {
			environmentNode := jsonEnvironmentNode{Name: valueOrUnknown(environment.Name)}
			for _, srcProject := range environment.SrcProjects {
				srcNode := jsonSourceNode{
					SrcProject:      valueOrUnknown(srcProject.SrcProject),
					SrcInterconnect: buildJSONInterconnects(srcProject.Interconnects),
				}
				environmentNode.SrcProjects = append(environmentNode.SrcProjects, srcNode)
			}
			workloadNode.Environments = append(workloadNode.Environments, environmentNode)
		}
		node.Workloads = append(node.Workloads, workloadNode)
	}
	return node
}

func buildJSONInterconnects(groups []interconnectGroup) []jsonInterconnectNode {
	result := make([]jsonInterconnectNode, 0, len(groups))
	for _, interconnect := range groups {
		node := jsonInterconnectNode{
			SrcInterconnect:  valueOrUnknown(interconnect.SrcInterconnect),
			Mapped:           interconnect.Mapped,
			SrcRegion:        valueOrUnknown(interconnect.SrcRegion),
			SrcState:         valueOrUnknown(interconnect.SrcState),
			SrcMacsecEnabled: interconnect.SrcMacsecEnabled,
			SrcMacsecKeyName: valueOrUnknown(interconnect.SrcMacsecKeyName),
		}
		for _, dst := range interconnect.DstProjects {
			dstNode := jsonDestinationNode{
				DstProject: valueOrUnknown(dst.DstProject),
				Mapped:     dst.Mapped,
			}
			for _, region := range dst.DstRegions {
				regionNode := jsonRegionNode{
					DstRegion: valueOrUnknown(region.DstRegion),
				}
				for _, attachment := range region.DstVLANAttachments {
					regionNode.DstVLANAttachments = append(regionNode.DstVLANAttachments, jsonAttachmentNode{
						DstVPC:                    valueOrUnknown(attachment.DstVPC),
						DstVLANAttachment:         valueOrUnknown(attachment.DstVLANAttachment),
						DstVLANAttachmentState:    valueOrUnknown(attachment.DstVLANAttachmentState),
						DstVLANAttachmentVLANID:   valueOrUnknown(attachment.DstVLANAttachmentVLANID),
						DstCloudRouter:            valueOrUnknown(attachment.DstCloudRouter),
						DstCloudRouterASN:         valueOrUnknown(attachment.DstCloudRouterASN),
						DstCloudRouterInterface:   valueOrUnknown(attachment.DstCloudRouterInterface),
						DstCloudRouterInterfaceIP: valueOrUnknown(attachment.DstCloudRouterInterfaceIP),
						RemoteBGPPeer:             valueOrUnknown(attachment.RemoteBGPPeer),
						RemoteBGPPeerIP:           valueOrUnknown(attachment.RemoteBGPPeerIP),
						RemoteBGPPeerASN:          valueOrUnknown(attachment.RemoteBGPPeerASN),
						BGPPeeringStatus:          valueOrUnknown(attachment.BGPPeeringStatus),
					})
				}
				dstNode.DstRegions = append(dstNode.DstRegions, regionNode)
			}
			node.DstProjects = append(node.DstProjects, dstNode)
		}
		result = append(result, node)
	}
	return result
}

func buildHierarchy(report model.Report) hierarchy {
	grouped := make(map[string]map[string]map[string]map[string][]model.MappingItem)
	for _, item := range normalizedItems(report) {
		workloads, ok := grouped[item.Org]
		if !ok {
			workloads = make(map[string]map[string]map[string][]model.MappingItem)
			grouped[item.Org] = workloads
		}
		environments, ok := workloads[item.Workload]
		if !ok {
			environments = make(map[string]map[string][]model.MappingItem)
			workloads[item.Workload] = environments
		}
		srcProjects, ok := environments[item.Environment]
		if !ok {
			srcProjects = make(map[string][]model.MappingItem)
			environments[item.Environment] = srcProjects
		}
		srcProjects[item.SrcProject] = append(srcProjects[item.SrcProject], item)
	}

	orgNames := sortedKeys(grouped)
	result := hierarchy{Orgs: make([]orgGroup, 0, len(orgNames))}
	for _, orgName := range orgNames {
		workloadMap := grouped[orgName]
		org := orgGroup{Name: orgName}
		for _, workloadName := range sortedKeys(workloadMap) {
			environmentMap := workloadMap[workloadName]
			workload := workloadGroup{Name: workloadName}
			for _, environmentName := range sortedKeys(environmentMap) {
				srcProjectMap := environmentMap[environmentName]
				environment := environmentGroup{Name: environmentName}
				for _, srcProjectName := range sortedKeys(srcProjectMap) {
					environment.SrcProjects = append(environment.SrcProjects, sourceGroup{
						SrcProject:    srcProjectName,
						Interconnects: groupInterconnects(srcProjectMap[srcProjectName]),
					})
				}
				workload.Environments = append(workload.Environments, environment)
			}
			org.Workloads = append(org.Workloads, workload)
		}
		result.Orgs = append(result.Orgs, org)
	}
	return result
}

func normalizedItems(report model.Report) []model.MappingItem {
	items := make([]model.MappingItem, 0, len(report.Items))
	for _, item := range report.Items {
		current := item
		current.Org = firstNonEmpty(current.Org, report.Selectors.Org)
		current.Workload = firstNonEmpty(current.Workload, report.Selectors.Workload)
		current.Environment = firstNonEmpty(current.Environment, report.Selectors.Environment)
		current.SrcProject = firstNonEmpty(current.SrcProject, report.SourceProject)
		items = append(items, current)
	}
	return items
}

func groupInterconnects(items []model.MappingItem) []interconnectGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if _, ok := grouped[item.SrcInterconnect]; !ok {
			names = append(names, item.SrcInterconnect)
		}
		grouped[item.SrcInterconnect] = append(grouped[item.SrcInterconnect], item)
	}
	sort.Strings(names)

	result := make([]interconnectGroup, 0, len(names))
	for _, name := range names {
		groupItems := grouped[name]
		if len(groupItems) == 0 {
			continue
		}
		group := interconnectGroup{
			SrcInterconnect:  name,
			SrcRegion:        groupItems[0].SrcRegion,
			SrcState:         groupItems[0].SrcState,
			SrcMacsecEnabled: groupItems[0].SrcMacsecEnabled,
			SrcMacsecKeyName: groupItems[0].SrcMacsecKeyName,
			DstProjects:      groupDestinations(groupItems),
		}
		for _, item := range groupItems {
			if item.Mapped {
				group.Mapped = true
				break
			}
		}
		result = append(result, group)
	}
	return result
}

func groupDestinations(items []model.MappingItem) []destinationGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if _, ok := grouped[item.DstProject]; !ok {
			names = append(names, item.DstProject)
		}
		grouped[item.DstProject] = append(grouped[item.DstProject], item)
	}
	sort.Strings(names)

	result := make([]destinationGroup, 0, len(names))
	for _, name := range names {
		dstItems := grouped[name]
		dst := destinationGroup{DstProject: name}
		for _, item := range dstItems {
			if item.Mapped {
				dst.Mapped = true
				break
			}
		}
		if dst.Mapped {
			dst.DstRegions = groupRegions(dstItems)
		}
		result = append(result, dst)
	}
	return result
}

func groupRegions(items []model.MappingItem) []regionGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if !item.Mapped {
			continue
		}
		if _, ok := grouped[item.DstRegion]; !ok {
			names = append(names, item.DstRegion)
		}
		grouped[item.DstRegion] = append(grouped[item.DstRegion], item)
	}
	sort.Strings(names)

	result := make([]regionGroup, 0, len(names))
	for _, name := range names {
		region := regionGroup{DstRegion: name}
		for _, item := range grouped[name] {
			region.DstVLANAttachments = append(region.DstVLANAttachments, attachmentGroup{
				DstVPC:                    item.DstVPC,
				DstVLANAttachment:         item.DstVLANAttachment,
				DstVLANAttachmentState:    item.DstVLANAttachmentState,
				DstVLANAttachmentVLANID:   item.DstVLANAttachmentVLANID,
				DstCloudRouter:            item.DstCloudRouter,
				DstCloudRouterASN:         item.DstCloudRouterASN,
				DstCloudRouterInterface:   item.DstCloudRouterInterface,
				DstCloudRouterInterfaceIP: item.DstCloudRouterInterfaceIP,
				RemoteBGPPeer:             item.RemoteBGPPeer,
				RemoteBGPPeerIP:           item.RemoteBGPPeerIP,
				RemoteBGPPeerASN:          item.RemoteBGPPeerASN,
				BGPPeeringStatus:          item.BGPPeeringStatus,
			})
		}
		result = append(result, region)
	}
	return result
}

func drawTreeWorkload(b *strings.Builder, workload workloadGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s workload: %s\n", indent, prefix, valueOrUnknown(workload.Name))
	for idx, environment := range workload.Environments {
		drawTreeEnvironment(b, environment, childIndent, idx == len(workload.Environments)-1)
	}
}

func drawTreeEnvironment(b *strings.Builder, environment environmentGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s environment: %s\n", indent, prefix, valueOrUnknown(environment.Name))
	for idx, srcProject := range environment.SrcProjects {
		drawTreeSourceProject(b, srcProject, childIndent, idx == len(environment.SrcProjects)-1)
	}
}

func drawTreeSourceProject(b *strings.Builder, srcProject sourceGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s project: %s\n", indent, prefix, valueOrUnknown(srcProject.SrcProject))
	for idx, interconnect := range srcProject.Interconnects {
		drawTreeInterconnect(b, interconnect, childIndent, idx == len(srcProject.Interconnects)-1)
	}
}

func drawTreeInterconnect(b *strings.Builder, interconnect interconnectGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s interconnect: %s [mapped: %t, region: %s, state: %s, macsec_enabled: %t, macsec_keyname: %s]\n",
		indent,
		prefix,
		valueOrUnknown(interconnect.SrcInterconnect),
		interconnect.Mapped,
		valueOrUnknown(interconnect.SrcRegion),
		valueOrUnknown(interconnect.SrcState),
		interconnect.SrcMacsecEnabled,
		valueOrUnknown(interconnect.SrcMacsecKeyName),
	)
	for idx, dst := range interconnect.DstProjects {
		drawTreeDestination(b, dst, childIndent, idx == len(interconnect.DstProjects)-1)
	}
}

func drawTreeDestination(b *strings.Builder, dst destinationGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s project: %s [mapped: %t]\n", indent, prefix, valueOrUnknown(dst.DstProject), dst.Mapped)
	if !dst.Mapped {
		fmt.Fprintf(b, "%s`-- unmapped\n", childIndent)
		return
	}
	for idx, region := range dst.DstRegions {
		drawTreeRegion(b, region, childIndent, idx == len(dst.DstRegions)-1)
	}
}

func drawTreeRegion(b *strings.Builder, region regionGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s region: %s\n", indent, prefix, valueOrUnknown(region.DstRegion))
	for idx, attachment := range region.DstVLANAttachments {
		drawTreeAttachment(b, attachment, childIndent, idx == len(region.DstVLANAttachments)-1)
	}
}

func drawTreeAttachment(b *strings.Builder, attachment attachmentGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s vlan_attachment: %s [vpc: %s, vlan_attachment_state: %s, vlan_attachment_vlanid: %s]\n",
		indent,
		prefix,
		valueOrUnknown(attachment.DstVLANAttachment),
		valueOrUnknown(attachment.DstVPC),
		valueOrUnknown(attachment.DstVLANAttachmentState),
		valueOrUnknown(attachment.DstVLANAttachmentVLANID),
	)
	fmt.Fprintf(
		b,
		"%s`-- cloud_router: %s [cloud_router_asn: %s]\n",
		childIndent,
		valueOrUnknown(attachment.DstCloudRouter),
		valueOrUnknown(attachment.DstCloudRouterASN),
	)
	fmt.Fprintf(
		b,
		"%s    `-- cloud_router_interface: %s [cloud_router_interface_ip: %s]\n",
		childIndent,
		valueOrUnknown(attachment.DstCloudRouterInterface),
		valueOrUnknown(attachment.DstCloudRouterInterfaceIP),
	)
	fmt.Fprintf(
		b,
		"%s        `-- remote_bgp_peer: %s [remote_bgp_peer_ip: %s, remote_bgp_peer_asn: %s, bgp_peering_status: %s]\n",
		childIndent,
		valueOrUnknown(attachment.RemoteBGPPeer),
		valueOrUnknown(attachment.RemoteBGPPeerIP),
		valueOrUnknown(attachment.RemoteBGPPeerASN),
		valueOrUnknown(attachment.BGPPeeringStatus),
	)
}

func interconnectNodeLabel(interconnect interconnectGroup) string {
	return fmt.Sprintf(
		"interconnect: %s<br>region: %s<br>state: %s<br>macsec_enabled: %t<br>macsec_keyname: %s",
		valueOrUnknown(interconnect.SrcInterconnect),
		valueOrUnknown(interconnect.SrcRegion),
		valueOrUnknown(interconnect.SrcState),
		interconnect.SrcMacsecEnabled,
		valueOrUnknown(interconnect.SrcMacsecKeyName),
	)
}

func interconnectItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"interconnect: %s<br>region: %s<br>state: %s<br>macsec_enabled: %t<br>macsec_keyname: %s",
		valueOrUnknown(item.SrcInterconnect),
		valueOrUnknown(item.SrcRegion),
		valueOrUnknown(item.SrcState),
		item.SrcMacsecEnabled,
		valueOrUnknown(item.SrcMacsecKeyName),
	)
}

func destinationProjectItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"project: %s<br>mapped: %t",
		valueOrUnknown(item.DstProject),
		item.Mapped,
	)
}

type regionVPCSummary struct {
	Shared bool
	Value  string
}

func buildRegionVPCs(items []model.MappingItem) map[string]regionVPCSummary {
	regionVPCSets := make(map[string]map[string]struct{})
	for _, item := range items {
		if !item.Mapped {
			continue
		}
		key := mermaidRegionKey(item)
		if _, ok := regionVPCSets[key]; !ok {
			regionVPCSets[key] = make(map[string]struct{})
		}
		regionVPCSets[key][item.DstVPC] = struct{}{}
	}

	result := make(map[string]regionVPCSummary, len(regionVPCSets))
	for key, values := range regionVPCSets {
		if len(values) == 1 {
			result[key] = regionVPCSummary{
				Shared: true,
				Value:  soleKey(values),
			}
			continue
		}
		result[key] = regionVPCSummary{}
	}
	return result
}

func mermaidRegionKey(item model.MappingItem) string {
	return item.Org + "\x00" + item.SrcProject + "\x00" + item.DstRegion
}

func destinationRegionItemLabel(item model.MappingItem, summary regionVPCSummary) string {
	if summary.Shared {
		return fmt.Sprintf(
			"region: %s<br>vpc: %s",
			valueOrUnknown(item.DstRegion),
			valueOrUnknown(summary.Value),
		)
	}
	return "region: " + valueOrUnknown(item.DstRegion)
}

func destinationVPCItemLabel(item model.MappingItem) string {
	return "vpc: " + valueOrUnknown(item.DstVPC)
}

func attachmentItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"vlan_attachment: %s<br>vlan_attachment_state: %s<br>vlan_attachment_vlanid: %s",
		valueOrUnknown(item.DstVLANAttachment),
		valueOrUnknown(item.DstVLANAttachmentState),
		valueOrUnknown(item.DstVLANAttachmentVLANID),
	)
}

func routerItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"cloud_router: %s<br>cloud_router_asn: %s",
		valueOrUnknown(item.DstCloudRouter),
		valueOrUnknown(item.DstCloudRouterASN),
	)
}

func interfaceItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"cloud_router_interface: %s<br>cloud_router_interface_ip: %s",
		valueOrUnknown(item.DstCloudRouterInterface),
		valueOrUnknown(item.DstCloudRouterInterfaceIP),
	)
}

func peerItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"remote_bgp_peer: %s<br>remote_bgp_peer_ip: %s<br>remote_bgp_peer_asn: %s",
		valueOrUnknown(item.RemoteBGPPeer),
		valueOrUnknown(item.RemoteBGPPeerIP),
		valueOrUnknown(item.RemoteBGPPeerASN),
	)
}

func peeringStatusItemLabel(item model.MappingItem) string {
	return "bgp_peering_status: " + valueOrUnknown(item.BGPPeeringStatus)
}

func destinationProjectNodeLabel(dst destinationGroup) string {
	return fmt.Sprintf(
		"project: %s<br>mapped: %t",
		valueOrUnknown(dst.DstProject),
		dst.Mapped,
	)
}

func destinationRegionNodeLabel(region regionGroup) string {
	return "region: " + valueOrUnknown(region.DstRegion)
}

func attachmentNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"vlan_attachment: %s<br>vlan_attachment_state: %s<br>vlan_attachment_vlanid: %s",
		valueOrUnknown(attachment.DstVLANAttachment),
		valueOrUnknown(attachment.DstVLANAttachmentState),
		valueOrUnknown(attachment.DstVLANAttachmentVLANID),
	)
}

func routerNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"cloud_router: %s<br>cloud_router_asn: %s",
		valueOrUnknown(attachment.DstCloudRouter),
		valueOrUnknown(attachment.DstCloudRouterASN),
	)
}

func interfaceNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"cloud_router_interface: %s<br>cloud_router_interface_ip: %s",
		valueOrUnknown(attachment.DstCloudRouterInterface),
		valueOrUnknown(attachment.DstCloudRouterInterfaceIP),
	)
}

func peerNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"remote_bgp_peer: %s<br>remote_bgp_peer_ip: %s<br>remote_bgp_peer_asn: %s",
		valueOrUnknown(attachment.RemoteBGPPeer),
		valueOrUnknown(attachment.RemoteBGPPeerIP),
		valueOrUnknown(attachment.RemoteBGPPeerASN),
	)
}

func peeringStatusNodeLabel(attachment attachmentGroup) string {
	return "bgp_peering_status: " + valueOrUnknown(attachment.BGPPeeringStatus)
}

func hasInterface(attachment attachmentGroup) bool {
	return strings.TrimSpace(attachment.DstCloudRouterInterface) != "" || strings.TrimSpace(attachment.DstCloudRouterInterfaceIP) != ""
}

func hasPeer(attachment attachmentGroup) bool {
	return strings.TrimSpace(attachment.RemoteBGPPeer) != "" || strings.TrimSpace(attachment.RemoteBGPPeerIP) != "" || strings.TrimSpace(attachment.RemoteBGPPeerASN) != ""
}

func hasInterfaceItem(item model.MappingItem) bool {
	return strings.TrimSpace(item.DstCloudRouterInterface) != "" || strings.TrimSpace(item.DstCloudRouterInterfaceIP) != ""
}

func hasPeerItem(item model.MappingItem) bool {
	return strings.TrimSpace(item.RemoteBGPPeer) != "" || strings.TrimSpace(item.RemoteBGPPeerIP) != "" || strings.TrimSpace(item.RemoteBGPPeerASN) != ""
}

func hasStatusItem(item model.MappingItem) bool {
	return strings.TrimSpace(item.BGPPeeringStatus) != "" || hasPeerItem(item)
}

type vpnHierarchy struct {
	Orgs []vpnOrgGroup
}

type vpnOrgGroup struct {
	Name      string
	Workloads []vpnWorkloadGroup
}

type vpnWorkloadGroup struct {
	Name         string
	Environments []vpnEnvironmentGroup
}

type vpnEnvironmentGroup struct {
	Name        string
	SrcProjects []vpnSourceProjectGroup
}

type vpnSourceProjectGroup struct {
	SrcProject string
	SrcRegions []vpnSourceRegionGroup
}

type vpnSourceRegionGroup struct {
	SrcRegion   string
	SrcVPC      string
	SrcGateways []vpnSourceGatewayGroup
}

type vpnSourceGatewayGroup struct {
	SrcVPNGateway          string
	SrcVPNGatewayType      string
	SrcVPNGatewayInterface string
	SrcVPNGatewayIP        string
	SrcTunnels             []vpnSourceTunnelGroup
}

type vpnSourceTunnelGroup struct {
	SrcVPNTunnel           string
	SrcVPNGatewayInterface string
	SrcVPNTunnelStatus     string
	Mapped                 bool
	SrcRouters             []vpnSourceRouterGroup
}

type vpnSourceRouterGroup struct {
	SrcCloudRouter            string
	SrcCloudRouterASN         string
	SrcCloudRouterInterface   string
	SrcCloudRouterInterfaceIP string
	SrcRoutes                 string
	BGPStatuses               []vpnBGPStatusGroup
}

type vpnBGPStatusGroup struct {
	BGPPeeringStatus string
	DstRouters       []vpnDestinationRouterGroup
}

type vpnDestinationRouterGroup struct {
	DstCloudRouter            string
	DstCloudRouterASN         string
	DstCloudRouterInterface   string
	DstCloudRouterInterfaceIP string
	DstRoutes                 string
	DstTunnels                []vpnDestinationTunnelGroup
}

type vpnDestinationTunnelGroup struct {
	DstVPNTunnel           string
	DstVPNGatewayInterface string
	DstVPNTunnelStatus     string
	DstGateways            []vpnDestinationGatewayGroup
}

type vpnDestinationGatewayGroup struct {
	DstVPNGateway          string
	DstVPNGatewayType      string
	DstVPNGatewayInterface string
	DstVPNGatewayIP        string
	DstRegions             []vpnDestinationRegionGroup
}

type vpnDestinationRegionGroup struct {
	DstRegion   string
	DstVPC      string
	DstProjects []vpnDestinationProjectGroup
}

type vpnDestinationProjectGroup struct {
	DstProject string
}

type vpnJSONReport struct {
	Type string         `json:"type"`
	Org  vpnJSONOrgNode `json:"org"`
}

type vpnJSONOrgNode struct {
	Name      string                `json:"name"`
	Workloads []vpnJSONWorkloadNode `json:"workloads,omitempty"`
}

type vpnJSONWorkloadNode struct {
	Name         string                   `json:"name"`
	Environments []vpnJSONEnvironmentNode `json:"environments,omitempty"`
}

type vpnJSONEnvironmentNode struct {
	Name        string                 `json:"name"`
	SrcProjects []vpnJSONSourceProject `json:"src_projects,omitempty"`
}

type vpnJSONSourceProject struct {
	SrcProject string                `json:"project"`
	SrcRegions []vpnJSONSourceRegion `json:"src_regions,omitempty"`
}

type vpnJSONSourceRegion struct {
	SrcRegion   string                 `json:"region"`
	SrcVPC      string                 `json:"vpc"`
	SrcGateways []vpnJSONSourceGateway `json:"src_vpn_gateways,omitempty"`
}

type vpnJSONSourceGateway struct {
	SrcVPNGateway          string                `json:"vpn_gateway"`
	SrcVPNGatewayType      string                `json:"vpn_gateway_type"`
	SrcVPNGatewayInterface string                `json:"vpn_gateway_interface"`
	SrcVPNGatewayIP        string                `json:"vpn_gateway_ip"`
	SrcTunnels             []vpnJSONSourceTunnel `json:"src_vpn_tunnels,omitempty"`
}

type vpnJSONSourceTunnel struct {
	SrcVPNTunnel           string                `json:"vpn_tunnel"`
	SrcVPNGatewayInterface string                `json:"vpn_gateway_interface"`
	SrcVPNTunnelStatus     string                `json:"vpn_tunnel_status"`
	SrcRouters             []vpnJSONSourceRouter `json:"src_cloud_routers,omitempty"`
}

type vpnJSONSourceRouter struct {
	SrcCloudRouter            string             `json:"cloud_router"`
	SrcCloudRouterASN         string             `json:"cloud_router_asn"`
	SrcCloudRouterInterface   string             `json:"cloud_router_interface"`
	SrcCloudRouterInterfaceIP string             `json:"cloud_router_interface_ip"`
	SrcRoutes                 string             `json:"routes"`
	BGPStatuses               []vpnJSONBGPStatus `json:"bgp_peering_statuses,omitempty"`
}

type vpnJSONBGPStatus struct {
	BGPPeeringStatus string                     `json:"bgp_peering_status"`
	DstRouters       []vpnJSONDestinationRouter `json:"dst_cloud_routers,omitempty"`
}

type vpnJSONDestinationRouter struct {
	DstCloudRouter            string                     `json:"cloud_router"`
	DstCloudRouterASN         string                     `json:"cloud_router_asn"`
	DstCloudRouterInterface   string                     `json:"cloud_router_interface"`
	DstCloudRouterInterfaceIP string                     `json:"cloud_router_interface_ip"`
	DstRoutes                 string                     `json:"routes"`
	DstTunnels                []vpnJSONDestinationTunnel `json:"dst_vpn_tunnels,omitempty"`
}

type vpnJSONDestinationTunnel struct {
	DstVPNTunnel           string                      `json:"vpn_tunnel"`
	DstVPNGatewayInterface string                      `json:"vpn_gateway_interface"`
	DstVPNTunnelStatus     string                      `json:"vpn_tunnel_status"`
	DstGateways            []vpnJSONDestinationGateway `json:"dst_vpn_gateways,omitempty"`
}

type vpnJSONDestinationGateway struct {
	DstVPNGateway          string                     `json:"vpn_gateway"`
	DstVPNGatewayType      string                     `json:"vpn_gateway_type"`
	DstVPNGatewayInterface string                     `json:"vpn_gateway_interface"`
	DstVPNGatewayIP        string                     `json:"vpn_gateway_ip"`
	DstRegions             []vpnJSONDestinationRegion `json:"dst_regions,omitempty"`
}

type vpnJSONDestinationRegion struct {
	DstRegion   string                      `json:"region"`
	DstVPC      string                      `json:"vpc"`
	DstProjects []vpnJSONDestinationProject `json:"dst_projects,omitempty"`
}

type vpnJSONDestinationProject struct {
	DstProject string `json:"project"`
}

func buildVPNJSONReport(report model.Report) vpnJSONReport {
	hierarchy := buildVPNHierarchy(report)
	root := vpnJSONOrgNode{Name: valueOrUnknown(report.Selectors.Org)}
	if len(hierarchy.Orgs) > 0 {
		root = buildVPNJSONOrg(hierarchy.Orgs[0])
	}
	return vpnJSONReport{
		Type: report.Type,
		Org:  root,
	}
}

func buildVPNJSONOrg(group vpnOrgGroup) vpnJSONOrgNode {
	node := vpnJSONOrgNode{Name: valueOrUnknown(group.Name)}
	for _, workload := range group.Workloads {
		workloadNode := vpnJSONWorkloadNode{Name: valueOrUnknown(workload.Name)}
		for _, environment := range workload.Environments {
			environmentNode := vpnJSONEnvironmentNode{Name: valueOrUnknown(environment.Name)}
			for _, srcProject := range environment.SrcProjects {
				projectNode := vpnJSONSourceProject{SrcProject: valueOrUnknown(srcProject.SrcProject)}
				for _, srcRegion := range srcProject.SrcRegions {
					regionNode := vpnJSONSourceRegion{
						SrcRegion: valueOrUnknown(srcRegion.SrcRegion),
						SrcVPC:    valueOrUnknown(srcRegion.SrcVPC),
					}
					for _, srcGateway := range srcRegion.SrcGateways {
						gatewayNode := vpnJSONSourceGateway{
							SrcVPNGateway:          valueOrUnknown(srcGateway.SrcVPNGateway),
							SrcVPNGatewayType:      valueOrUnknown(srcGateway.SrcVPNGatewayType),
							SrcVPNGatewayInterface: valueOrUnknown(srcGateway.SrcVPNGatewayInterface),
							SrcVPNGatewayIP:        valueOrUnknown(srcGateway.SrcVPNGatewayIP),
						}
						for _, srcTunnel := range srcGateway.SrcTunnels {
							tunnelNode := vpnJSONSourceTunnel{
								SrcVPNTunnel:           valueOrUnknown(srcTunnel.SrcVPNTunnel),
								SrcVPNGatewayInterface: valueOrUnknown(srcTunnel.SrcVPNGatewayInterface),
								SrcVPNTunnelStatus:     valueOrUnknown(srcTunnel.SrcVPNTunnelStatus),
							}
							for _, srcRouter := range srcTunnel.SrcRouters {
								routerNode := vpnJSONSourceRouter{
									SrcCloudRouter:            valueOrUnknown(srcRouter.SrcCloudRouter),
									SrcCloudRouterASN:         valueOrUnknown(srcRouter.SrcCloudRouterASN),
									SrcCloudRouterInterface:   valueOrUnknown(srcRouter.SrcCloudRouterInterface),
									SrcCloudRouterInterfaceIP: valueOrUnknown(srcRouter.SrcCloudRouterInterfaceIP),
									SrcRoutes:                 valueOrUnknown(srcRouter.SrcRoutes),
								}
								for _, status := range srcRouter.BGPStatuses {
									statusNode := vpnJSONBGPStatus{
										BGPPeeringStatus: valueOrUnknown(status.BGPPeeringStatus),
									}
									for _, dstRouter := range status.DstRouters {
										dstRouterNode := vpnJSONDestinationRouter{
											DstCloudRouter:            valueOrUnknown(dstRouter.DstCloudRouter),
											DstCloudRouterASN:         valueOrUnknown(dstRouter.DstCloudRouterASN),
											DstCloudRouterInterface:   valueOrUnknown(dstRouter.DstCloudRouterInterface),
											DstCloudRouterInterfaceIP: valueOrUnknown(dstRouter.DstCloudRouterInterfaceIP),
											DstRoutes:                 valueOrUnknown(dstRouter.DstRoutes),
										}
										for _, dstTunnel := range dstRouter.DstTunnels {
											dstTunnelNode := vpnJSONDestinationTunnel{
												DstVPNTunnel:           valueOrUnknown(dstTunnel.DstVPNTunnel),
												DstVPNGatewayInterface: valueOrUnknown(dstTunnel.DstVPNGatewayInterface),
												DstVPNTunnelStatus:     valueOrUnknown(dstTunnel.DstVPNTunnelStatus),
											}
											for _, dstGateway := range dstTunnel.DstGateways {
												dstGatewayNode := vpnJSONDestinationGateway{
													DstVPNGateway:          valueOrUnknown(dstGateway.DstVPNGateway),
													DstVPNGatewayType:      valueOrUnknown(dstGateway.DstVPNGatewayType),
													DstVPNGatewayInterface: valueOrUnknown(dstGateway.DstVPNGatewayInterface),
													DstVPNGatewayIP:        valueOrUnknown(dstGateway.DstVPNGatewayIP),
												}
												for _, dstRegion := range dstGateway.DstRegions {
													dstRegionNode := vpnJSONDestinationRegion{
														DstRegion: valueOrUnknown(dstRegion.DstRegion),
														DstVPC:    valueOrUnknown(dstRegion.DstVPC),
													}
													for _, dstProject := range dstRegion.DstProjects {
														dstRegionNode.DstProjects = append(dstRegionNode.DstProjects, vpnJSONDestinationProject{
															DstProject: valueOrUnknown(dstProject.DstProject),
														})
													}
													dstGatewayNode.DstRegions = append(dstGatewayNode.DstRegions, dstRegionNode)
												}
												dstTunnelNode.DstGateways = append(dstTunnelNode.DstGateways, dstGatewayNode)
											}
											dstRouterNode.DstTunnels = append(dstRouterNode.DstTunnels, dstTunnelNode)
										}
										statusNode.DstRouters = append(statusNode.DstRouters, dstRouterNode)
									}
									routerNode.BGPStatuses = append(routerNode.BGPStatuses, statusNode)
								}
								tunnelNode.SrcRouters = append(tunnelNode.SrcRouters, routerNode)
							}
							gatewayNode.SrcTunnels = append(gatewayNode.SrcTunnels, tunnelNode)
						}
						regionNode.SrcGateways = append(regionNode.SrcGateways, gatewayNode)
					}
					projectNode.SrcRegions = append(projectNode.SrcRegions, regionNode)
				}
				environmentNode.SrcProjects = append(environmentNode.SrcProjects, projectNode)
			}
			workloadNode.Environments = append(workloadNode.Environments, environmentNode)
		}
		node.Workloads = append(node.Workloads, workloadNode)
	}
	return node
}

func renderVPNTree(report model.Report) []byte {
	hierarchy := buildVPNHierarchy(report)
	var b strings.Builder
	for idx, org := range hierarchy.Orgs {
		if idx > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "org: %s\n", valueOrUnknown(org.Name))
		for workloadIdx, workload := range org.Workloads {
			drawVPNTreeWorkload(&b, workload, "", workloadIdx == len(org.Workloads)-1)
		}
	}
	return []byte(b.String())
}

func renderVPNMermaid(report model.Report) []byte {
	items := normalizedItems(report)
	var b strings.Builder
	b.WriteString("flowchart LR\n")

	seen := make(map[string]struct{})
	for _, item := range items {
		orgID := mermaidID("vpn-org-" + item.Org)
		workloadID := mermaidID("vpn-workload-" + item.Org + "-" + item.Workload)
		environmentID := mermaidID("vpn-environment-" + item.Org + "-" + item.Environment)
		srcProjectID := mermaidID("vpn-src-project-" + item.Org + "-" + item.SrcProject)
		srcRegionID := mermaidID("vpn-src-region-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion + "-" + item.SrcVPC)
		srcGatewayID := mermaidID("vpn-src-gateway-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion + "-" + item.SrcVPC + "-" + item.SrcVPNGateway + "-" + item.SrcVPNGatewayType + "-" + item.SrcVPNGatewayInterface + "-" + item.SrcVPNGatewayIP)
		srcTunnelID := mermaidID("vpn-src-tunnel-" + srcGatewayID + "-" + item.SrcVPNTunnel + "-" + item.SrcVPNGatewayInterface + "-" + item.SrcVPNTunnelStatus)
		srcRouterID := mermaidID("vpn-src-router-" + srcTunnelID + "-" + item.SrcCloudRouter + "-" + item.SrcCloudRouterASN + "-" + item.SrcCloudRouterInterface + "-" + item.SrcCloudRouterInterfaceIP + "-" + item.SrcRoutes)

		defineMermaidNode(&b, seen, orgID, "org: "+valueOrUnknown(item.Org))
		linkIfMissing(&b, seen, orgID, workloadID, "workload: "+valueOrUnknown(item.Workload))
		linkIfMissing(&b, seen, workloadID, environmentID, "environment: "+valueOrUnknown(item.Environment))
		linkIfMissing(&b, seen, environmentID, srcProjectID, "src_project: "+valueOrUnknown(item.SrcProject))
		linkIfMissing(&b, seen, srcProjectID, srcRegionID, vpnSourceRegionItemLabel(item))
		linkIfMissing(&b, seen, srcRegionID, srcGatewayID, vpnSourceGatewayItemLabel(item))
		linkIfMissing(&b, seen, srcGatewayID, srcTunnelID, vpnSourceTunnelItemLabel(item))
		linkIfMissing(&b, seen, srcTunnelID, srcRouterID, vpnSourceRouterItemLabel(item))

		if !item.Mapped || strings.TrimSpace(item.DstProject) == "" {
			unmappedID := mermaidID("vpn-unmapped-" + srcTunnelID)
			linkIfMissing(&b, seen, srcRouterID, unmappedID, "unmapped")
			continue
		}

		statusID := mermaidID("vpn-bgp-status-" + srcRouterID + "-" + item.BGPPeeringStatus)
		linkIfMissing(&b, seen, srcRouterID, statusID, peeringStatusItemLabel(item))

		if strings.TrimSpace(item.DstCloudRouter) == "" {
			continue
		}

		// Keep the destination branch scoped to the current source-tunnel/status path
		// so distinct tunnel pairs never collapse into one shared Mermaid subtree.
		branchScopeID := mermaidID("vpn-dst-scope-" + statusID + "-" + item.SrcVPNTunnel)
		dstRouterID := mermaidID("vpn-dst-router-" + branchScopeID + "-" + item.DstCloudRouter + "-" + item.DstCloudRouterASN + "-" + item.DstCloudRouterInterface + "-" + item.DstCloudRouterInterfaceIP + "-" + item.DstRoutes)

		linkIfMissing(&b, seen, statusID, dstRouterID, vpnDestinationRouterItemLabel(item))
		if strings.TrimSpace(item.DstVPNTunnel) == "" {
			continue
		}

		dstTunnelID := mermaidID("vpn-dst-tunnel-" + dstRouterID + "-" + item.DstVPNTunnel + "-" + item.DstVPNGatewayInterface + "-" + item.DstVPNTunnelStatus)
		dstGatewayID := mermaidID("vpn-dst-gateway-" + branchScopeID + "-" + item.DstVPNGateway + "-" + item.DstVPNGatewayType + "-" + item.DstVPNGatewayInterface + "-" + item.DstVPNGatewayIP + "-" + item.DstRegion + "-" + item.DstVPC + "-" + item.DstProject)
		dstRegionID := mermaidID("vpn-dst-region-" + branchScopeID + "-" + item.DstVPNGateway + "-" + item.DstVPNGatewayType + "-" + item.DstRegion + "-" + item.DstVPC + "-" + item.DstProject)
		dstProjectID := mermaidID("vpn-dst-project-" + branchScopeID + "-" + item.DstVPNGateway + "-" + item.DstVPNGatewayType + "-" + item.DstRegion + "-" + item.DstVPC + "-" + item.DstProject)

		linkIfMissing(&b, seen, dstRouterID, dstTunnelID, vpnDestinationTunnelItemLabel(item))
		linkIfMissing(&b, seen, dstTunnelID, dstGatewayID, vpnDestinationGatewayItemLabel(item))
		linkIfMissing(&b, seen, dstGatewayID, dstRegionID, vpnDestinationRegionItemLabel(item))
		linkIfMissing(&b, seen, dstRegionID, dstProjectID, "dst_project: "+valueOrUnknown(item.DstProject))
	}
	return []byte(b.String())
}

func buildVPNHierarchy(report model.Report) vpnHierarchy {
	grouped := make(map[string]map[string]map[string]map[string][]model.MappingItem)
	for _, item := range normalizedItems(report) {
		workloads, ok := grouped[item.Org]
		if !ok {
			workloads = make(map[string]map[string]map[string][]model.MappingItem)
			grouped[item.Org] = workloads
		}
		environments, ok := workloads[item.Workload]
		if !ok {
			environments = make(map[string]map[string][]model.MappingItem)
			workloads[item.Workload] = environments
		}
		srcProjects, ok := environments[item.Environment]
		if !ok {
			srcProjects = make(map[string][]model.MappingItem)
			environments[item.Environment] = srcProjects
		}
		srcProjects[item.SrcProject] = append(srcProjects[item.SrcProject], item)
	}

	orgNames := sortedKeys(grouped)
	result := vpnHierarchy{Orgs: make([]vpnOrgGroup, 0, len(orgNames))}
	for _, orgName := range orgNames {
		workloadMap := grouped[orgName]
		org := vpnOrgGroup{Name: orgName}
		for _, workloadName := range sortedKeys(workloadMap) {
			environmentMap := workloadMap[workloadName]
			workload := vpnWorkloadGroup{Name: workloadName}
			for _, environmentName := range sortedKeys(environmentMap) {
				srcProjectMap := environmentMap[environmentName]
				environment := vpnEnvironmentGroup{Name: environmentName}
				for _, srcProjectName := range sortedKeys(srcProjectMap) {
					environment.SrcProjects = append(environment.SrcProjects, vpnSourceProjectGroup{
						SrcProject: srcProjectName,
						SrcRegions: groupVPNSourceRegions(srcProjectMap[srcProjectName]),
					})
				}
				workload.Environments = append(workload.Environments, environment)
			}
			org.Workloads = append(org.Workloads, workload)
		}
		result.Orgs = append(result.Orgs, org)
	}
	return result
}

func groupVPNSourceRegions(items []model.MappingItem) []vpnSourceRegionGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.SrcRegion + "\x00" + item.SrcVPC
		if _, ok := grouped[key]; !ok {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(keys)
	var result []vpnSourceRegionGroup
	for _, key := range keys {
		groupItems := grouped[key]
		result = append(result, vpnSourceRegionGroup{
			SrcRegion:   groupItems[0].SrcRegion,
			SrcVPC:      groupItems[0].SrcVPC,
			SrcGateways: groupVPNSourceGateways(groupItems),
		})
	}
	return result
}

func groupVPNSourceGateways(items []model.MappingItem) []vpnSourceGatewayGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.SrcVPNGateway + "\x00" + item.SrcVPNGatewayType + "\x00" + item.SrcVPNGatewayInterface + "\x00" + item.SrcVPNGatewayIP
		if _, ok := grouped[key]; !ok {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(keys)
	var result []vpnSourceGatewayGroup
	for _, key := range keys {
		groupItems := grouped[key]
		if len(groupItems) == 0 {
			continue
		}
		result = append(result, vpnSourceGatewayGroup{
			SrcVPNGateway:          groupItems[0].SrcVPNGateway,
			SrcVPNGatewayType:      groupItems[0].SrcVPNGatewayType,
			SrcVPNGatewayInterface: groupItems[0].SrcVPNGatewayInterface,
			SrcVPNGatewayIP:        groupItems[0].SrcVPNGatewayIP,
			SrcTunnels:             groupVPNSrcTunnels(groupItems),
		})
	}
	return result
}

func groupVPNSrcTunnels(items []model.MappingItem) []vpnSourceTunnelGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		key := item.SrcVPNTunnel + "\x00" + item.SrcVPNGatewayInterface + "\x00" + item.SrcVPNTunnelStatus
		if _, ok := grouped[key]; !ok {
			names = append(names, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(names)
	var tunnelResult []vpnSourceTunnelGroup
	for _, key := range names {
		groupItems := grouped[key]
		if len(groupItems) == 0 {
			continue
		}
		group := vpnSourceTunnelGroup{
			SrcVPNTunnel:           groupItems[0].SrcVPNTunnel,
			SrcVPNGatewayInterface: groupItems[0].SrcVPNGatewayInterface,
			SrcVPNTunnelStatus:     groupItems[0].SrcVPNTunnelStatus,
			SrcRouters:             groupVPNSourceRouters(groupItems),
		}
		for _, item := range groupItems {
			if item.Mapped {
				group.Mapped = true
				break
			}
		}
		tunnelResult = append(tunnelResult, group)
	}
	return tunnelResult
}

func groupVPNSourceRouters(items []model.MappingItem) []vpnSourceRouterGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.SrcCloudRouter + "\x00" + item.SrcCloudRouterASN + "\x00" + item.SrcCloudRouterInterface + "\x00" + item.SrcCloudRouterInterfaceIP + "\x00" + item.SrcRoutes
		if _, ok := grouped[key]; !ok {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(keys)
	var result []vpnSourceRouterGroup
	for _, key := range keys {
		groupItems := grouped[key]
		if len(groupItems) == 0 {
			continue
		}
		result = append(result, vpnSourceRouterGroup{
			SrcCloudRouter:            groupItems[0].SrcCloudRouter,
			SrcCloudRouterASN:         groupItems[0].SrcCloudRouterASN,
			SrcCloudRouterInterface:   groupItems[0].SrcCloudRouterInterface,
			SrcCloudRouterInterfaceIP: groupItems[0].SrcCloudRouterInterfaceIP,
			SrcRoutes:                 groupItems[0].SrcRoutes,
			BGPStatuses:               groupVPNBGPStatuses(groupItems),
		})
	}
	return result
}

func groupVPNBGPStatuses(items []model.MappingItem) []vpnBGPStatusGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if !item.Mapped {
			continue
		}
		if _, ok := grouped[item.BGPPeeringStatus]; !ok {
			names = append(names, item.BGPPeeringStatus)
		}
		grouped[item.BGPPeeringStatus] = append(grouped[item.BGPPeeringStatus], item)
	}
	sort.Strings(names)
	var result []vpnBGPStatusGroup
	for _, name := range names {
		result = append(result, vpnBGPStatusGroup{
			BGPPeeringStatus: name,
			DstRouters:       groupVPNDestinationRouters(grouped[name]),
		})
	}
	return result
}

func groupVPNDestinationRouters(items []model.MappingItem) []vpnDestinationRouterGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		if strings.TrimSpace(item.DstCloudRouter) == "" {
			continue
		}
		key := item.DstCloudRouter + "\x00" + item.DstCloudRouterASN + "\x00" + item.DstCloudRouterInterface + "\x00" + item.DstCloudRouterInterfaceIP + "\x00" + item.DstRoutes
		if _, ok := grouped[key]; !ok {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(keys)
	var result []vpnDestinationRouterGroup
	for _, key := range keys {
		groupItems := grouped[key]
		if len(groupItems) == 0 {
			continue
		}
		result = append(result, vpnDestinationRouterGroup{
			DstCloudRouter:            groupItems[0].DstCloudRouter,
			DstCloudRouterASN:         groupItems[0].DstCloudRouterASN,
			DstCloudRouterInterface:   groupItems[0].DstCloudRouterInterface,
			DstCloudRouterInterfaceIP: groupItems[0].DstCloudRouterInterfaceIP,
			DstRoutes:                 groupItems[0].DstRoutes,
			DstTunnels:                groupVPNDestinationTunnels(groupItems),
		})
	}
	return result
}

func groupVPNDestinationTunnels(items []model.MappingItem) []vpnDestinationTunnelGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		if strings.TrimSpace(item.DstVPNTunnel) == "" {
			continue
		}
		key := item.DstVPNTunnel + "\x00" + item.DstVPNGatewayInterface + "\x00" + item.DstVPNTunnelStatus
		if _, ok := grouped[key]; !ok {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(keys)
	var result []vpnDestinationTunnelGroup
	for _, key := range keys {
		groupItems := grouped[key]
		if len(groupItems) == 0 {
			continue
		}
		result = append(result, vpnDestinationTunnelGroup{
			DstVPNTunnel:           groupItems[0].DstVPNTunnel,
			DstVPNGatewayInterface: groupItems[0].DstVPNGatewayInterface,
			DstVPNTunnelStatus:     groupItems[0].DstVPNTunnelStatus,
			DstGateways:            groupVPNDestinationGateways(groupItems),
		})
	}
	return result
}

func groupVPNDestinationGateways(items []model.MappingItem) []vpnDestinationGatewayGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.DstVPNGateway + "\x00" + item.DstVPNGatewayType + "\x00" + item.DstVPNGatewayInterface + "\x00" + item.DstVPNGatewayIP + "\x00" + item.DstRegion + "\x00" + item.DstVPC + "\x00" + item.DstProject
		if _, ok := grouped[key]; !ok {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(keys)
	var result []vpnDestinationGatewayGroup
	for _, key := range keys {
		groupItems := grouped[key]
		if len(groupItems) == 0 {
			continue
		}
		result = append(result, vpnDestinationGatewayGroup{
			DstVPNGateway:          groupItems[0].DstVPNGateway,
			DstVPNGatewayType:      groupItems[0].DstVPNGatewayType,
			DstVPNGatewayInterface: groupItems[0].DstVPNGatewayInterface,
			DstVPNGatewayIP:        groupItems[0].DstVPNGatewayIP,
			DstRegions:             groupVPNDestinationRegions(groupItems),
		})
	}
	return result
}

func groupVPNDestinationRegions(items []model.MappingItem) []vpnDestinationRegionGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		if strings.TrimSpace(item.DstRegion) == "" {
			continue
		}
		key := item.DstRegion + "\x00" + item.DstVPC
		if _, ok := grouped[key]; !ok {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], item)
	}
	sort.Strings(keys)
	var result []vpnDestinationRegionGroup
	for _, key := range keys {
		groupItems := grouped[key]
		result = append(result, vpnDestinationRegionGroup{
			DstRegion:   groupItems[0].DstRegion,
			DstVPC:      groupItems[0].DstVPC,
			DstProjects: groupVPNDestinationProjects(groupItems),
		})
	}
	return result
}

func groupVPNDestinationProjects(items []model.MappingItem) []vpnDestinationProjectGroup {
	seen := make(map[string]struct{})
	var names []string
	for _, item := range items {
		if strings.TrimSpace(item.DstProject) == "" {
			continue
		}
		if _, ok := seen[item.DstProject]; ok {
			continue
		}
		seen[item.DstProject] = struct{}{}
		names = append(names, item.DstProject)
	}
	sort.Strings(names)
	var result []vpnDestinationProjectGroup
	for _, name := range names {
		result = append(result, vpnDestinationProjectGroup{DstProject: name})
	}
	return result
}

func drawVPNTreeWorkload(b *strings.Builder, workload vpnWorkloadGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s workload: %s\n", indent, prefix, valueOrUnknown(workload.Name))
	for idx, environment := range workload.Environments {
		drawVPNTreeEnvironment(b, environment, childIndent, idx == len(workload.Environments)-1)
	}
}

func drawVPNTreeEnvironment(b *strings.Builder, environment vpnEnvironmentGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s environment: %s\n", indent, prefix, valueOrUnknown(environment.Name))
	for idx, srcProject := range environment.SrcProjects {
		drawVPNTreeSourceProject(b, srcProject, childIndent, idx == len(environment.SrcProjects)-1)
	}
}

func drawVPNTreeSourceProject(b *strings.Builder, srcProject vpnSourceProjectGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s project: %s\n", indent, prefix, valueOrUnknown(srcProject.SrcProject))
	for idx, srcRegion := range srcProject.SrcRegions {
		drawVPNTreeSourceRegion(b, srcRegion, childIndent, idx == len(srcProject.SrcRegions)-1)
	}
}

func drawVPNTreeSourceRegion(b *strings.Builder, srcRegion vpnSourceRegionGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s region: %s [vpc: %s]\n",
		indent,
		prefix,
		valueOrUnknown(srcRegion.SrcRegion),
		valueOrUnknown(srcRegion.SrcVPC),
	)
	for idx, gateway := range srcRegion.SrcGateways {
		drawVPNTreeSourceGateway(b, gateway, childIndent, idx == len(srcRegion.SrcGateways)-1)
	}
}

func drawVPNTreeSourceGateway(b *strings.Builder, gateway vpnSourceGatewayGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s vpn_gateway: %s [vpn_gateway_type: %s, vpn_gateway_interface: %s, vpn_gateway_ip: %s]\n",
		indent,
		prefix,
		valueOrUnknown(gateway.SrcVPNGateway),
		valueOrUnknown(gateway.SrcVPNGatewayType),
		valueOrUnknown(gateway.SrcVPNGatewayInterface),
		valueOrUnknown(gateway.SrcVPNGatewayIP),
	)
	for idx, tunnel := range gateway.SrcTunnels {
		drawVPNTreeSourceTunnel(b, tunnel, childIndent, idx == len(gateway.SrcTunnels)-1)
	}
}

func drawVPNTreeSourceTunnel(b *strings.Builder, tunnel vpnSourceTunnelGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s vpn_tunnel: %s [vpn_gateway_interface: %s, vpn_tunnel_status: %s]\n",
		indent,
		prefix,
		valueOrUnknown(tunnel.SrcVPNTunnel),
		valueOrUnknown(tunnel.SrcVPNGatewayInterface),
		valueOrUnknown(tunnel.SrcVPNTunnelStatus),
	)
	if !tunnel.Mapped {
		fmt.Fprintf(b, "%s`-- unmapped\n", childIndent)
		return
	}
	for idx, router := range tunnel.SrcRouters {
		drawVPNTreeSourceRouter(b, router, childIndent, idx == len(tunnel.SrcRouters)-1)
	}
}

func drawVPNTreeSourceRouter(b *strings.Builder, router vpnSourceRouterGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s cloud_router: %s [cloud_router_asn: %s, cloud_router_interface: %s, cloud_router_interface_ip: %s]\n",
		indent,
		prefix,
		valueOrUnknown(router.SrcCloudRouter),
		valueOrUnknown(router.SrcCloudRouterASN),
		valueOrUnknown(router.SrcCloudRouterInterface),
		valueOrUnknown(router.SrcCloudRouterInterfaceIP),
	)
	routeIsLast := len(router.BGPStatuses) == 0
	drawVPNTreeRoutes(b, router.SrcRoutes, childIndent, routeIsLast)
	for idx, status := range router.BGPStatuses {
		drawVPNTreeBGPStatus(b, status, childIndent, idx == len(router.BGPStatuses)-1)
	}
}

func drawVPNTreeBGPStatus(b *strings.Builder, status vpnBGPStatusGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(b, "%s%s bgp_peering_status: %s\n", indent, prefix, valueOrUnknown(status.BGPPeeringStatus))
	for idx, dstRouter := range status.DstRouters {
		drawVPNTreeDestinationRouter(b, dstRouter, childIndent, idx == len(status.DstRouters)-1)
	}
}

func drawVPNTreeDestinationRouter(b *strings.Builder, router vpnDestinationRouterGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s cloud_router: %s [cloud_router_asn: %s, cloud_router_interface: %s, cloud_router_interface_ip: %s]\n",
		indent,
		prefix,
		valueOrUnknown(router.DstCloudRouter),
		valueOrUnknown(router.DstCloudRouterASN),
		valueOrUnknown(router.DstCloudRouterInterface),
		valueOrUnknown(router.DstCloudRouterInterfaceIP),
	)
	routeIsLast := len(router.DstTunnels) == 0
	drawVPNTreeRoutes(b, router.DstRoutes, childIndent, routeIsLast)
	for idx, tunnel := range router.DstTunnels {
		drawVPNTreeDestinationTunnel(b, tunnel, childIndent, idx == len(router.DstTunnels)-1)
	}
}

func drawVPNTreeRoutes(b *strings.Builder, routes, indent string, isLast bool) {
	prefix := "|--"
	if isLast {
		prefix = "`--"
	}
	fmt.Fprintf(b, "%s%s routes: %s\n", indent, prefix, valueOrUnknown(routes))
}

func drawVPNTreeDestinationTunnel(b *strings.Builder, tunnel vpnDestinationTunnelGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s vpn_tunnel: %s [vpn_gateway_interface: %s, vpn_tunnel_status: %s]\n",
		indent,
		prefix,
		valueOrUnknown(tunnel.DstVPNTunnel),
		valueOrUnknown(tunnel.DstVPNGatewayInterface),
		valueOrUnknown(tunnel.DstVPNTunnelStatus),
	)
	for idx, gateway := range tunnel.DstGateways {
		drawVPNTreeDestinationGateway(b, gateway, childIndent, idx == len(tunnel.DstGateways)-1)
	}
}

func drawVPNTreeDestinationGateway(b *strings.Builder, gateway vpnDestinationGatewayGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s vpn_gateway: %s [vpn_gateway_type: %s, vpn_gateway_interface: %s, vpn_gateway_ip: %s]\n",
		indent,
		prefix,
		valueOrUnknown(gateway.DstVPNGateway),
		valueOrUnknown(gateway.DstVPNGatewayType),
		valueOrUnknown(gateway.DstVPNGatewayInterface),
		valueOrUnknown(gateway.DstVPNGatewayIP),
	)
	for idx, region := range gateway.DstRegions {
		drawVPNTreeDestinationRegion(b, region, childIndent, idx == len(gateway.DstRegions)-1)
	}
}

func drawVPNTreeDestinationRegion(b *strings.Builder, region vpnDestinationRegionGroup, indent string, isLast bool) {
	prefix := "|--"
	childIndent := indent + "|   "
	if isLast {
		prefix = "`--"
		childIndent = indent + "    "
	}
	fmt.Fprintf(
		b,
		"%s%s region: %s [vpc: %s]\n",
		indent,
		prefix,
		valueOrUnknown(region.DstRegion),
		valueOrUnknown(region.DstVPC),
	)
	for idx, project := range region.DstProjects {
		drawVPNTreeDestinationProject(b, project, childIndent, idx == len(region.DstProjects)-1)
	}
}

func drawVPNTreeDestinationProject(b *strings.Builder, project vpnDestinationProjectGroup, indent string, isLast bool) {
	prefix := "|--"
	if isLast {
		prefix = "`--"
	}
	fmt.Fprintf(b, "%s%s project: %s\n", indent, prefix, valueOrUnknown(project.DstProject))
}

func vpnSourceGatewayItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"vpn_gateway: %s<br>vpn_gateway_type: %s<br>vpn_gateway_interface: %s<br>vpn_gateway_ip: %s",
		valueOrUnknown(item.SrcVPNGateway),
		valueOrUnknown(item.SrcVPNGatewayType),
		valueOrUnknown(item.SrcVPNGatewayInterface),
		valueOrUnknown(item.SrcVPNGatewayIP),
	)
}

func vpnSourceRegionItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"region: %s<br>vpc: %s",
		valueOrUnknown(item.SrcRegion),
		valueOrUnknown(item.SrcVPC),
	)
}

func vpnSourceRouterItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"cloud_router: %s<br>cloud_router_asn: %s<br>cloud_router_interface: %s<br>cloud_router_interface_ip: %s%s",
		valueOrUnknown(item.SrcCloudRouter),
		valueOrUnknown(item.SrcCloudRouterASN),
		valueOrUnknown(item.SrcCloudRouterInterface),
		valueOrUnknown(item.SrcCloudRouterInterfaceIP),
		wrappedRoutesLabel(item.SrcRoutes, "<br>"),
	)
}

func vpnSourceTunnelItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"vpn_tunnel: %s<br>vpn_gateway_interface: %s<br>vpn_tunnel_status: %s",
		valueOrUnknown(item.SrcVPNTunnel),
		valueOrUnknown(item.SrcVPNGatewayInterface),
		valueOrUnknown(item.SrcVPNTunnelStatus),
	)
}

func vpnDestinationGatewayItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"vpn_gateway: %s<br>vpn_gateway_type: %s<br>vpn_gateway_interface: %s<br>vpn_gateway_ip: %s",
		valueOrUnknown(item.DstVPNGateway),
		valueOrUnknown(item.DstVPNGatewayType),
		valueOrUnknown(item.DstVPNGatewayInterface),
		valueOrUnknown(item.DstVPNGatewayIP),
	)
}

func vpnDestinationRegionItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"region: %s<br>vpc: %s",
		valueOrUnknown(item.DstRegion),
		valueOrUnknown(item.DstVPC),
	)
}

func vpnDestinationTunnelItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"vpn_tunnel: %s<br>vpn_gateway_interface: %s<br>vpn_tunnel_status: %s",
		valueOrUnknown(item.DstVPNTunnel),
		valueOrUnknown(item.DstVPNGatewayInterface),
		valueOrUnknown(item.DstVPNTunnelStatus),
	)
}

func vpnDestinationRouterItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"cloud_router: %s<br>cloud_router_asn: %s<br>cloud_router_interface: %s<br>cloud_router_interface_ip: %s%s",
		valueOrUnknown(item.DstCloudRouter),
		valueOrUnknown(item.DstCloudRouterASN),
		valueOrUnknown(item.DstCloudRouterInterface),
		valueOrUnknown(item.DstCloudRouterInterfaceIP),
		wrappedRoutesLabel(item.DstRoutes, "<br>"),
	)
}

func wrappedRoutesLabel(routes, lineBreak string) string {
	parts := splitRoutes(routes)
	if len(parts) == 0 {
		return lineBreak + "routes: unknown"
	}

	displayParts := make([]string, 0, len(parts))
	for idx, part := range parts {
		if idx < len(parts)-1 {
			displayParts = append(displayParts, part+",")
			continue
		}
		displayParts = append(displayParts, part)
	}

	rows := make([]string, 0, (len(displayParts)+1)/2)
	for idx := 0; idx < len(displayParts); idx += 2 {
		end := idx + 2
		if end > len(displayParts) {
			end = len(displayParts)
		}
		rows = append(rows, strings.Join(displayParts[idx:end], " "))
	}
	if len(rows) == 0 {
		return lineBreak + "routes: unknown"
	}

	label := lineBreak + "routes: " + rows[0]
	for _, row := range rows[1:] {
		label += lineBreak + row
	}
	return label
}

func splitRoutes(routes string) []string {
	if strings.TrimSpace(routes) == "" {
		return nil
	}
	parts := strings.Split(routes, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func defineMermaidNode(b *strings.Builder, seen map[string]struct{}, id, label string) {
	nodeKey := "node:" + id
	if _, ok := seen[nodeKey]; ok {
		return
	}
	fmt.Fprintf(b, "    %s[%q]\n", id, label)
	seen[nodeKey] = struct{}{}
}

func linkIfMissing(b *strings.Builder, seen map[string]struct{}, parentID, childID, childLabel string) {
	defineMermaidNode(b, seen, childID, childLabel)
	edgeKey := "edge:" + parentID + "->" + childID
	if _, ok := seen[edgeKey]; ok {
		return
	}
	fmt.Fprintf(b, "    %s --> %s\n", parentID, childID)
	seen[edgeKey] = struct{}{}
}

func mermaidID(value string) string {
	value = strings.ToLower(value)
	replacer := strings.NewReplacer(
		"-", "_",
		".", "_",
		"/", "_",
		":", "_",
		" ", "_",
	)
	return replacer.Replace(value)
}

func valueOrUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func soleKey(values map[string]struct{}) string {
	for value := range values {
		return value
	}
	return ""
}
