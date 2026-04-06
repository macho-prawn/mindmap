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
	"src_vpn_tunnel",
	"src_vpn_tunnel_status",
	"bgp_peering_status",
	"dst_vpn_tunnel",
	"dst_vpn_tunnel_status",
	"dst_cloud_router",
	"dst_cloud_router_asn",
	"dst_cloud_router_interface",
	"dst_cloud_router_interface_ip",
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
		item.SrcVPNTunnel,
		item.SrcVPNTunnelStatus,
		item.BGPPeeringStatus,
		item.DstVPNTunnel,
		item.DstVPNTunnelStatus,
		item.DstCloudRouter,
		item.DstCloudRouterASN,
		item.DstCloudRouterInterface,
		item.DstCloudRouterInterfaceIP,
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
	SrcProject      string                 `json:"src_project"`
	SrcInterconnect []jsonInterconnectNode `json:"src_interconnects,omitempty"`
}

type jsonInterconnectNode struct {
	SrcInterconnect  string                `json:"src_interconnect"`
	Mapped           bool                  `json:"mapped"`
	SrcRegion        string                `json:"src_region"`
	SrcState         string                `json:"src_state"`
	SrcMacsecEnabled bool                  `json:"src_macsec_enabled"`
	SrcMacsecKeyName string                `json:"src_macsec_keyname"`
	DstProjects      []jsonDestinationNode `json:"dst_projects,omitempty"`
}

type jsonDestinationNode struct {
	DstProject string           `json:"dst_project"`
	Mapped     bool             `json:"mapped"`
	DstRegions []jsonRegionNode `json:"dst_regions,omitempty"`
}

type jsonRegionNode struct {
	DstRegion          string               `json:"dst_region"`
	DstVLANAttachments []jsonAttachmentNode `json:"dst_vlan_attachments,omitempty"`
}

type jsonAttachmentNode struct {
	DstVPC                    string `json:"dst_vpc"`
	DstVLANAttachment         string `json:"dst_vlan_attachment"`
	DstVLANAttachmentState    string `json:"dst_vlan_attachment_state"`
	DstVLANAttachmentVLANID   string `json:"dst_vlan_attachment_vlanid"`
	DstCloudRouter            string `json:"dst_cloud_router"`
	DstCloudRouterASN         string `json:"dst_cloud_router_asn"`
	DstCloudRouterInterface   string `json:"dst_cloud_router_interface"`
	DstCloudRouterInterfaceIP string `json:"dst_cloud_router_interface_ip"`
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
	fmt.Fprintf(b, "%s%s src_project: %s\n", indent, prefix, valueOrUnknown(srcProject.SrcProject))
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
		"%s%s src_interconnect: %s [mapped: %t, src_region: %s, src_state: %s, src_macsec_enabled: %t, src_macsec_keyname: %s]\n",
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
	fmt.Fprintf(b, "%s%s dst_project: %s [mapped: %t]\n", indent, prefix, valueOrUnknown(dst.DstProject), dst.Mapped)
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
	fmt.Fprintf(b, "%s%s dst_region: %s\n", indent, prefix, valueOrUnknown(region.DstRegion))
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
		"%s%s dst_vlan_attachment: %s [dst_vpc: %s, dst_vlan_attachment_state: %s, dst_vlan_attachment_vlanid: %s]\n",
		indent,
		prefix,
		valueOrUnknown(attachment.DstVLANAttachment),
		valueOrUnknown(attachment.DstVPC),
		valueOrUnknown(attachment.DstVLANAttachmentState),
		valueOrUnknown(attachment.DstVLANAttachmentVLANID),
	)
	fmt.Fprintf(
		b,
		"%s`-- dst_cloud_router: %s [dst_cloud_router_asn: %s]\n",
		childIndent,
		valueOrUnknown(attachment.DstCloudRouter),
		valueOrUnknown(attachment.DstCloudRouterASN),
	)
	fmt.Fprintf(
		b,
		"%s    `-- dst_cloud_router_interface: %s [dst_cloud_router_interface_ip: %s]\n",
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
		"src_interconnect: %s<br>src_region: %s<br>src_state: %s<br>src_macsec_enabled: %t<br>src_macsec_keyname: %s",
		valueOrUnknown(interconnect.SrcInterconnect),
		valueOrUnknown(interconnect.SrcRegion),
		valueOrUnknown(interconnect.SrcState),
		interconnect.SrcMacsecEnabled,
		valueOrUnknown(interconnect.SrcMacsecKeyName),
	)
}

func interconnectItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"src_interconnect: %s<br>src_region: %s<br>src_state: %s<br>src_macsec_enabled: %t<br>src_macsec_keyname: %s",
		valueOrUnknown(item.SrcInterconnect),
		valueOrUnknown(item.SrcRegion),
		valueOrUnknown(item.SrcState),
		item.SrcMacsecEnabled,
		valueOrUnknown(item.SrcMacsecKeyName),
	)
}

func destinationProjectItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_project: %s<br>mapped: %t",
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
			"dst_region: %s<br>dst_vpc: %s",
			valueOrUnknown(item.DstRegion),
			valueOrUnknown(summary.Value),
		)
	}
	return "dst_region: " + valueOrUnknown(item.DstRegion)
}

func destinationVPCItemLabel(item model.MappingItem) string {
	return "dst_vpc: " + valueOrUnknown(item.DstVPC)
}

func attachmentItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_vlan_attachment: %s<br>dst_vlan_attachment_state: %s<br>dst_vlan_attachment_vlanid: %s",
		valueOrUnknown(item.DstVLANAttachment),
		valueOrUnknown(item.DstVLANAttachmentState),
		valueOrUnknown(item.DstVLANAttachmentVLANID),
	)
}

func routerItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_cloud_router: %s<br>dst_cloud_router_asn: %s",
		valueOrUnknown(item.DstCloudRouter),
		valueOrUnknown(item.DstCloudRouterASN),
	)
}

func interfaceItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_cloud_router_interface: %s<br>dst_cloud_router_interface_ip: %s",
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
		"dst_project: %s<br>mapped: %t",
		valueOrUnknown(dst.DstProject),
		dst.Mapped,
	)
}

func destinationRegionNodeLabel(region regionGroup) string {
	return "dst_region: " + valueOrUnknown(region.DstRegion)
}

func attachmentNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"dst_vlan_attachment: %s<br>dst_vlan_attachment_state: %s<br>dst_vlan_attachment_vlanid: %s",
		valueOrUnknown(attachment.DstVLANAttachment),
		valueOrUnknown(attachment.DstVLANAttachmentState),
		valueOrUnknown(attachment.DstVLANAttachmentVLANID),
	)
}

func routerNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"dst_cloud_router: %s<br>dst_cloud_router_asn: %s",
		valueOrUnknown(attachment.DstCloudRouter),
		valueOrUnknown(attachment.DstCloudRouterASN),
	)
}

func interfaceNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"dst_cloud_router_interface: %s<br>dst_cloud_router_interface_ip: %s",
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
	SrcGateways []vpnSourceGatewayGroup
}

type vpnSourceGatewayGroup struct {
	SrcVPNGateway     string
	SrcVPNGatewayType string
	SrcRouters        []vpnSourceRouterGroup
}

type vpnSourceRouterGroup struct {
	SrcCloudRouter            string
	SrcCloudRouterASN         string
	SrcCloudRouterInterface   string
	SrcCloudRouterInterfaceIP string
	SrcTunnels                []vpnSourceTunnelGroup
}

type vpnSourceTunnelGroup struct {
	SrcVPNTunnel       string
	SrcVPNTunnelStatus string
	Mapped             bool
	BGPStatuses        []vpnBGPStatusGroup
}

type vpnBGPStatusGroup struct {
	BGPPeeringStatus string
	DstTunnels       []vpnDestinationTunnelGroup
}

type vpnDestinationTunnelGroup struct {
	DstVPNTunnel       string
	DstVPNTunnelStatus string
	DstRouters         []vpnDestinationRouterGroup
}

type vpnDestinationRouterGroup struct {
	DstCloudRouter            string
	DstCloudRouterASN         string
	DstCloudRouterInterface   string
	DstCloudRouterInterfaceIP string
	DstGateways               []vpnDestinationGatewayGroup
}

type vpnDestinationGatewayGroup struct {
	DstVPNGateway     string
	DstVPNGatewayType string
	DstRegions        []vpnDestinationRegionGroup
}

type vpnDestinationRegionGroup struct {
	DstRegion   string
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
	SrcProject string                `json:"src_project"`
	SrcRegions []vpnJSONSourceRegion `json:"src_regions,omitempty"`
}

type vpnJSONSourceRegion struct {
	SrcRegion   string                 `json:"src_region"`
	SrcGateways []vpnJSONSourceGateway `json:"src_vpn_gateways,omitempty"`
}

type vpnJSONSourceGateway struct {
	SrcVPNGateway     string                `json:"src_vpn_gateway"`
	SrcVPNGatewayType string                `json:"src_vpn_gateway_type"`
	SrcRouters        []vpnJSONSourceRouter `json:"src_cloud_routers,omitempty"`
}

type vpnJSONSourceRouter struct {
	SrcCloudRouter            string                `json:"src_cloud_router"`
	SrcCloudRouterASN         string                `json:"src_cloud_router_asn"`
	SrcCloudRouterInterface   string                `json:"src_cloud_router_interface"`
	SrcCloudRouterInterfaceIP string                `json:"src_cloud_router_interface_ip"`
	SrcTunnels                []vpnJSONSourceTunnel `json:"src_vpn_tunnels,omitempty"`
}

type vpnJSONSourceTunnel struct {
	SrcVPNTunnel       string             `json:"src_vpn_tunnel"`
	SrcVPNTunnelStatus string             `json:"src_vpn_tunnel_status"`
	BGPStatuses        []vpnJSONBGPStatus `json:"bgp_peering_statuses,omitempty"`
}

type vpnJSONBGPStatus struct {
	BGPPeeringStatus string                     `json:"bgp_peering_status"`
	DstTunnels       []vpnJSONDestinationTunnel `json:"dst_vpn_tunnels,omitempty"`
}

type vpnJSONDestinationTunnel struct {
	DstVPNTunnel       string                     `json:"dst_vpn_tunnel"`
	DstVPNTunnelStatus string                     `json:"dst_vpn_tunnel_status"`
	DstRouters         []vpnJSONDestinationRouter `json:"dst_cloud_routers,omitempty"`
}

type vpnJSONDestinationRouter struct {
	DstCloudRouter            string                      `json:"dst_cloud_router"`
	DstCloudRouterASN         string                      `json:"dst_cloud_router_asn"`
	DstCloudRouterInterface   string                      `json:"dst_cloud_router_interface"`
	DstCloudRouterInterfaceIP string                      `json:"dst_cloud_router_interface_ip"`
	DstGateways               []vpnJSONDestinationGateway `json:"dst_vpn_gateways,omitempty"`
}

type vpnJSONDestinationGateway struct {
	DstVPNGateway     string                     `json:"dst_vpn_gateway"`
	DstVPNGatewayType string                     `json:"dst_vpn_gateway_type"`
	DstRegions        []vpnJSONDestinationRegion `json:"dst_regions,omitempty"`
}

type vpnJSONDestinationRegion struct {
	DstRegion   string                      `json:"dst_region"`
	DstProjects []vpnJSONDestinationProject `json:"dst_projects,omitempty"`
}

type vpnJSONDestinationProject struct {
	DstProject string `json:"dst_project"`
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
					regionNode := vpnJSONSourceRegion{SrcRegion: valueOrUnknown(srcRegion.SrcRegion)}
					for _, srcGateway := range srcRegion.SrcGateways {
						gatewayNode := vpnJSONSourceGateway{
							SrcVPNGateway:     valueOrUnknown(srcGateway.SrcVPNGateway),
							SrcVPNGatewayType: valueOrUnknown(srcGateway.SrcVPNGatewayType),
						}
						for _, srcRouter := range srcGateway.SrcRouters {
							routerNode := vpnJSONSourceRouter{
								SrcCloudRouter:            valueOrUnknown(srcRouter.SrcCloudRouter),
								SrcCloudRouterASN:         valueOrUnknown(srcRouter.SrcCloudRouterASN),
								SrcCloudRouterInterface:   valueOrUnknown(srcRouter.SrcCloudRouterInterface),
								SrcCloudRouterInterfaceIP: valueOrUnknown(srcRouter.SrcCloudRouterInterfaceIP),
							}
							for _, srcTunnel := range srcRouter.SrcTunnels {
								tunnelNode := vpnJSONSourceTunnel{
									SrcVPNTunnel:       valueOrUnknown(srcTunnel.SrcVPNTunnel),
									SrcVPNTunnelStatus: valueOrUnknown(srcTunnel.SrcVPNTunnelStatus),
								}
								for _, status := range srcTunnel.BGPStatuses {
									statusNode := vpnJSONBGPStatus{
										BGPPeeringStatus: valueOrUnknown(status.BGPPeeringStatus),
									}
									for _, dstTunnel := range status.DstTunnels {
										dstTunnelNode := vpnJSONDestinationTunnel{
											DstVPNTunnel:       valueOrUnknown(dstTunnel.DstVPNTunnel),
											DstVPNTunnelStatus: valueOrUnknown(dstTunnel.DstVPNTunnelStatus),
										}
										for _, dstRouter := range dstTunnel.DstRouters {
											dstRouterNode := vpnJSONDestinationRouter{
												DstCloudRouter:            valueOrUnknown(dstRouter.DstCloudRouter),
												DstCloudRouterASN:         valueOrUnknown(dstRouter.DstCloudRouterASN),
												DstCloudRouterInterface:   valueOrUnknown(dstRouter.DstCloudRouterInterface),
												DstCloudRouterInterfaceIP: valueOrUnknown(dstRouter.DstCloudRouterInterfaceIP),
											}
											for _, dstGateway := range dstRouter.DstGateways {
												dstGatewayNode := vpnJSONDestinationGateway{
													DstVPNGateway:     valueOrUnknown(dstGateway.DstVPNGateway),
													DstVPNGatewayType: valueOrUnknown(dstGateway.DstVPNGatewayType),
												}
												for _, dstRegion := range dstGateway.DstRegions {
													dstRegionNode := vpnJSONDestinationRegion{
														DstRegion: valueOrUnknown(dstRegion.DstRegion),
													}
													for _, dstProject := range dstRegion.DstProjects {
														dstRegionNode.DstProjects = append(dstRegionNode.DstProjects, vpnJSONDestinationProject{
															DstProject: valueOrUnknown(dstProject.DstProject),
														})
													}
													dstGatewayNode.DstRegions = append(dstGatewayNode.DstRegions, dstRegionNode)
												}
												dstRouterNode.DstGateways = append(dstRouterNode.DstGateways, dstGatewayNode)
											}
											dstTunnelNode.DstRouters = append(dstTunnelNode.DstRouters, dstRouterNode)
										}
										statusNode.DstTunnels = append(statusNode.DstTunnels, dstTunnelNode)
									}
									tunnelNode.BGPStatuses = append(tunnelNode.BGPStatuses, statusNode)
								}
								routerNode.SrcTunnels = append(routerNode.SrcTunnels, tunnelNode)
							}
							gatewayNode.SrcRouters = append(gatewayNode.SrcRouters, routerNode)
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
		srcRegionID := mermaidID("vpn-src-region-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion)
		srcGatewayID := mermaidID("vpn-src-gateway-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion + "-" + item.SrcVPNGateway + "-" + item.SrcVPNGatewayType)
		srcRouterID := mermaidID("vpn-src-router-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion + "-" + item.SrcVPNGateway + "-" + item.SrcCloudRouter + "-" + item.SrcCloudRouterInterface)
		srcTunnelID := mermaidID("vpn-src-tunnel-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion + "-" + item.SrcVPNGateway + "-" + item.SrcCloudRouter + "-" + item.SrcVPNTunnel)

		defineMermaidNode(&b, seen, orgID, "org: "+valueOrUnknown(item.Org))
		linkIfMissing(&b, seen, orgID, workloadID, "workload: "+valueOrUnknown(item.Workload))
		linkIfMissing(&b, seen, workloadID, environmentID, "environment: "+valueOrUnknown(item.Environment))
		linkIfMissing(&b, seen, environmentID, srcProjectID, "src_project: "+valueOrUnknown(item.SrcProject))
		linkIfMissing(&b, seen, srcProjectID, srcRegionID, "src_region: "+valueOrUnknown(item.SrcRegion))
		linkIfMissing(&b, seen, srcRegionID, srcGatewayID, vpnSourceGatewayItemLabel(item))
		linkIfMissing(&b, seen, srcGatewayID, srcRouterID, vpnSourceRouterItemLabel(item))
		linkIfMissing(&b, seen, srcRouterID, srcTunnelID, vpnSourceTunnelItemLabel(item))

		if !item.Mapped || strings.TrimSpace(item.DstProject) == "" {
			unmappedID := mermaidID("vpn-unmapped-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion + "-" + item.SrcVPNGateway + "-" + item.SrcCloudRouter + "-" + item.SrcVPNTunnel)
			linkIfMissing(&b, seen, srcTunnelID, unmappedID, "unmapped")
			continue
		}

		statusID := mermaidID("vpn-bgp-status-" + item.Org + "-" + item.SrcProject + "-" + item.SrcRegion + "-" + item.SrcVPNGateway + "-" + item.SrcCloudRouter + "-" + item.SrcVPNTunnel + "-" + item.BGPPeeringStatus)
		linkIfMissing(&b, seen, srcTunnelID, statusID, peeringStatusItemLabel(item))

		if strings.TrimSpace(item.DstVPNTunnel) == "" {
			continue
		}

		// Keep the destination branch scoped to the current source-tunnel/status path
		// so distinct tunnel pairs never collapse into one shared Mermaid subtree.
		dstTunnelID := mermaidID("vpn-dst-tunnel-" + statusID + "-" + item.DstVPNTunnel)
		dstRouterID := mermaidID("vpn-dst-router-" + dstTunnelID + "-" + item.DstCloudRouter + "-" + item.DstCloudRouterInterface)
		dstGatewayID := mermaidID("vpn-dst-gateway-" + dstRouterID + "-" + item.DstVPNGateway + "-" + item.DstVPNGatewayType)
		dstRegionID := mermaidID("vpn-dst-region-" + dstGatewayID + "-" + item.DstRegion)
		dstProjectID := mermaidID("vpn-dst-project-" + dstRegionID + "-" + item.DstProject)

		linkIfMissing(&b, seen, statusID, dstTunnelID, vpnDestinationTunnelItemLabel(item))
		linkIfMissing(&b, seen, dstTunnelID, dstRouterID, vpnDestinationRouterItemLabel(item))
		linkIfMissing(&b, seen, dstRouterID, dstGatewayID, vpnDestinationGatewayItemLabel(item))
		linkIfMissing(&b, seen, dstGatewayID, dstRegionID, "dst_region: "+valueOrUnknown(item.DstRegion))
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
	for _, item := range items {
		grouped[item.SrcRegion] = append(grouped[item.SrcRegion], item)
	}
	var result []vpnSourceRegionGroup
	for _, region := range sortedKeys(grouped) {
		result = append(result, vpnSourceRegionGroup{
			SrcRegion:   region,
			SrcGateways: groupVPNSourceGateways(grouped[region]),
		})
	}
	return result
}

func groupVPNSourceGateways(items []model.MappingItem) []vpnSourceGatewayGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.SrcVPNGateway + "\x00" + item.SrcVPNGatewayType
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
			SrcVPNGateway:     groupItems[0].SrcVPNGateway,
			SrcVPNGatewayType: groupItems[0].SrcVPNGatewayType,
			SrcRouters:        groupVPNSourceRouters(groupItems),
		})
	}
	return result
}

func groupVPNSourceRouters(items []model.MappingItem) []vpnSourceRouterGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.SrcCloudRouter + "\x00" + item.SrcCloudRouterASN + "\x00" + item.SrcCloudRouterInterface + "\x00" + item.SrcCloudRouterInterfaceIP
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
			SrcTunnels:                groupVPNSrcTunnels(groupItems),
		})
	}
	return result
}

func groupVPNSrcTunnels(items []model.MappingItem) []vpnSourceTunnelGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if _, ok := grouped[item.SrcVPNTunnel]; !ok {
			names = append(names, item.SrcVPNTunnel)
		}
		grouped[item.SrcVPNTunnel] = append(grouped[item.SrcVPNTunnel], item)
	}
	sort.Strings(names)
	var result []vpnSourceTunnelGroup
	for _, name := range names {
		groupItems := grouped[name]
		if len(groupItems) == 0 {
			continue
		}
		group := vpnSourceTunnelGroup{
			SrcVPNTunnel:       name,
			SrcVPNTunnelStatus: groupItems[0].SrcVPNTunnelStatus,
			BGPStatuses:        groupVPNBGPStatuses(groupItems),
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
			DstTunnels:       groupVPNDestinationTunnels(grouped[name]),
		})
	}
	return result
}

func groupVPNDestinationTunnels(items []model.MappingItem) []vpnDestinationTunnelGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if strings.TrimSpace(item.DstVPNTunnel) == "" {
			continue
		}
		if _, ok := grouped[item.DstVPNTunnel]; !ok {
			names = append(names, item.DstVPNTunnel)
		}
		grouped[item.DstVPNTunnel] = append(grouped[item.DstVPNTunnel], item)
	}
	sort.Strings(names)
	var result []vpnDestinationTunnelGroup
	for _, name := range names {
		result = append(result, vpnDestinationTunnelGroup{
			DstVPNTunnel:       name,
			DstVPNTunnelStatus: grouped[name][0].DstVPNTunnelStatus,
			DstRouters:         groupVPNDestinationRouters(grouped[name]),
		})
	}
	return result
}

func groupVPNDestinationRouters(items []model.MappingItem) []vpnDestinationRouterGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.DstCloudRouter + "\x00" + item.DstCloudRouterASN + "\x00" + item.DstCloudRouterInterface + "\x00" + item.DstCloudRouterInterfaceIP
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
			DstGateways:               groupVPNDestinationGateways(groupItems),
		})
	}
	return result
}

func groupVPNDestinationGateways(items []model.MappingItem) []vpnDestinationGatewayGroup {
	grouped := make(map[string][]model.MappingItem)
	var keys []string
	for _, item := range items {
		key := item.DstVPNGateway + "\x00" + item.DstVPNGatewayType
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
			DstVPNGateway:     groupItems[0].DstVPNGateway,
			DstVPNGatewayType: groupItems[0].DstVPNGatewayType,
			DstRegions:        groupVPNDestinationRegions(groupItems),
		})
	}
	return result
}

func groupVPNDestinationRegions(items []model.MappingItem) []vpnDestinationRegionGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range items {
		if strings.TrimSpace(item.DstRegion) == "" {
			continue
		}
		if _, ok := grouped[item.DstRegion]; !ok {
			names = append(names, item.DstRegion)
		}
		grouped[item.DstRegion] = append(grouped[item.DstRegion], item)
	}
	sort.Strings(names)
	var result []vpnDestinationRegionGroup
	for _, name := range names {
		result = append(result, vpnDestinationRegionGroup{
			DstRegion:   name,
			DstProjects: groupVPNDestinationProjects(grouped[name]),
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
	fmt.Fprintf(b, "%s%s src_project: %s\n", indent, prefix, valueOrUnknown(srcProject.SrcProject))
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
	fmt.Fprintf(b, "%s%s src_region: %s\n", indent, prefix, valueOrUnknown(srcRegion.SrcRegion))
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
		"%s%s src_vpn_gateway: %s [src_vpn_gateway_type: %s]\n",
		indent,
		prefix,
		valueOrUnknown(gateway.SrcVPNGateway),
		valueOrUnknown(gateway.SrcVPNGatewayType),
	)
	for idx, router := range gateway.SrcRouters {
		drawVPNTreeSourceRouter(b, router, childIndent, idx == len(gateway.SrcRouters)-1)
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
		"%s%s src_cloud_router: %s [src_cloud_router_asn: %s, src_cloud_router_interface: %s, src_cloud_router_interface_ip: %s]\n",
		indent,
		prefix,
		valueOrUnknown(router.SrcCloudRouter),
		valueOrUnknown(router.SrcCloudRouterASN),
		valueOrUnknown(router.SrcCloudRouterInterface),
		valueOrUnknown(router.SrcCloudRouterInterfaceIP),
	)
	for idx, tunnel := range router.SrcTunnels {
		drawVPNTreeSourceTunnel(b, tunnel, childIndent, idx == len(router.SrcTunnels)-1)
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
		"%s%s src_vpn_tunnel: %s [src_vpn_tunnel_status: %s]\n",
		indent,
		prefix,
		valueOrUnknown(tunnel.SrcVPNTunnel),
		valueOrUnknown(tunnel.SrcVPNTunnelStatus),
	)
	if !tunnel.Mapped {
		fmt.Fprintf(b, "%s`-- unmapped\n", childIndent)
		return
	}
	for idx, status := range tunnel.BGPStatuses {
		drawVPNTreeBGPStatus(b, status, childIndent, idx == len(tunnel.BGPStatuses)-1)
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
	for idx, dstTunnel := range status.DstTunnels {
		drawVPNTreeDestinationTunnel(b, dstTunnel, childIndent, idx == len(status.DstTunnels)-1)
	}
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
		"%s%s dst_vpn_tunnel: %s [dst_vpn_tunnel_status: %s]\n",
		indent,
		prefix,
		valueOrUnknown(tunnel.DstVPNTunnel),
		valueOrUnknown(tunnel.DstVPNTunnelStatus),
	)
	for idx, router := range tunnel.DstRouters {
		drawVPNTreeDestinationRouter(b, router, childIndent, idx == len(tunnel.DstRouters)-1)
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
		"%s%s dst_cloud_router: %s [dst_cloud_router_asn: %s, dst_cloud_router_interface: %s, dst_cloud_router_interface_ip: %s]\n",
		indent,
		prefix,
		valueOrUnknown(router.DstCloudRouter),
		valueOrUnknown(router.DstCloudRouterASN),
		valueOrUnknown(router.DstCloudRouterInterface),
		valueOrUnknown(router.DstCloudRouterInterfaceIP),
	)
	for idx, gateway := range router.DstGateways {
		drawVPNTreeDestinationGateway(b, gateway, childIndent, idx == len(router.DstGateways)-1)
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
		"%s%s dst_vpn_gateway: %s [dst_vpn_gateway_type: %s]\n",
		indent,
		prefix,
		valueOrUnknown(gateway.DstVPNGateway),
		valueOrUnknown(gateway.DstVPNGatewayType),
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
	fmt.Fprintf(b, "%s%s dst_region: %s\n", indent, prefix, valueOrUnknown(region.DstRegion))
	for idx, project := range region.DstProjects {
		drawVPNTreeDestinationProject(b, project, childIndent, idx == len(region.DstProjects)-1)
	}
}

func drawVPNTreeDestinationProject(b *strings.Builder, project vpnDestinationProjectGroup, indent string, isLast bool) {
	prefix := "|--"
	if isLast {
		prefix = "`--"
	}
	fmt.Fprintf(b, "%s%s dst_project: %s\n", indent, prefix, valueOrUnknown(project.DstProject))
}

func vpnSourceGatewayItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"src_vpn_gateway: %s<br>src_vpn_gateway_type: %s",
		valueOrUnknown(item.SrcVPNGateway),
		valueOrUnknown(item.SrcVPNGatewayType),
	)
}

func vpnSourceRouterItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"src_cloud_router: %s<br>src_cloud_router_asn: %s<br>src_cloud_router_interface: %s<br>src_cloud_router_interface_ip: %s",
		valueOrUnknown(item.SrcCloudRouter),
		valueOrUnknown(item.SrcCloudRouterASN),
		valueOrUnknown(item.SrcCloudRouterInterface),
		valueOrUnknown(item.SrcCloudRouterInterfaceIP),
	)
}

func vpnSourceTunnelItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"src_vpn_tunnel: %s<br>src_vpn_tunnel_status: %s",
		valueOrUnknown(item.SrcVPNTunnel),
		valueOrUnknown(item.SrcVPNTunnelStatus),
	)
}

func vpnDestinationGatewayItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_vpn_gateway: %s<br>dst_vpn_gateway_type: %s",
		valueOrUnknown(item.DstVPNGateway),
		valueOrUnknown(item.DstVPNGatewayType),
	)
}

func vpnDestinationTunnelItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_vpn_tunnel: %s<br>dst_vpn_tunnel_status: %s",
		valueOrUnknown(item.DstVPNTunnel),
		valueOrUnknown(item.DstVPNTunnelStatus),
	)
}

func vpnDestinationRouterItemLabel(item model.MappingItem) string {
	return fmt.Sprintf(
		"dst_cloud_router: %s<br>dst_cloud_router_asn: %s<br>dst_cloud_router_interface: %s<br>dst_cloud_router_interface_ip: %s",
		valueOrUnknown(item.DstCloudRouter),
		valueOrUnknown(item.DstCloudRouterASN),
		valueOrUnknown(item.DstCloudRouterInterface),
		valueOrUnknown(item.DstCloudRouterInterfaceIP),
	)
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
