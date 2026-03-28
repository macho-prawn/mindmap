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
	FormatCSV     = "csv"
	FormatTSV     = "tsv"
	FormatJSON    = "json"
	FormatTree    = "tree"
)

var separatedHeader = []string{
	"org",
	"workload",
	"environment",
	"src_project",
	"src_interconnect",
	"mapped",
	"src_region",
	"src_state",
	"dst_project",
	"dst_region",
	"dst_vlan_attachment",
	"dst_vlan_attachment_state",
	"dst_vlan_attachment_vlanid",
	"dst_cloud_router",
	"dst_cloud_router_interface",
	"dst_cloud_router_interface_ip",
	"remote_bgp_peer",
	"remote_bgp_peer_ip",
	"bgp_peering_status",
}

func Render(report model.Report, format string) ([]byte, string, error) {
	switch format {
	case "", FormatMermaid:
		return renderMermaid(report), "mmd", nil
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
	if err := writer.Write(separatedHeader); err != nil {
		return nil, err
	}
	for _, item := range normalizedItems(report) {
		record := []string{
			item.Org,
			item.Workload,
			item.Environment,
			item.SrcProject,
			item.SrcInterconnect,
			fmt.Sprintf("%t", item.Mapped),
			item.SrcRegion,
			item.SrcState,
			item.DstProject,
			item.DstRegion,
			item.DstVLANAttachment,
			item.DstVLANAttachmentState,
			item.DstVLANAttachmentVLANID,
			item.DstCloudRouter,
			item.DstCloudRouterInterface,
			item.DstCloudRouterInterfaceIP,
			item.RemoteBGPPeer,
			item.RemoteBGPPeerIP,
			item.BGPPeeringStatus,
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return buf.Bytes(), writer.Error()
}

func renderJSON(report model.Report) ([]byte, error) {
	return json.MarshalIndent(buildJSONReport(report), "", "  ")
}

func renderTree(report model.Report) []byte {
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
	hierarchy := buildHierarchy(report)
	var b strings.Builder
	b.WriteString("flowchart LR\n")

	seen := make(map[string]struct{})
	for _, org := range hierarchy.Orgs {
		orgID := mermaidID("org-" + org.Name)
		defineMermaidNode(&b, seen, orgID, "org: "+valueOrUnknown(org.Name))
		for _, workload := range org.Workloads {
			workloadID := mermaidID("workload-" + org.Name + "-" + workload.Name)
			linkIfMissing(&b, seen, orgID, workloadID, "workload: "+valueOrUnknown(workload.Name))
			for _, environment := range workload.Environments {
				environmentID := mermaidID("environment-" + org.Name + "-" + workload.Name + "-" + environment.Name)
				linkIfMissing(&b, seen, workloadID, environmentID, "environment: "+valueOrUnknown(environment.Name))
				for _, srcProject := range environment.SrcProjects {
					srcID := mermaidID("src-" + org.Name + "-" + workload.Name + "-" + environment.Name + "-" + srcProject.SrcProject)
					linkIfMissing(&b, seen, environmentID, srcID, "src_project: "+valueOrUnknown(srcProject.SrcProject))
					scope := strings.Join([]string{org.Name, workload.Name, environment.Name, srcProject.SrcProject}, "/")
					renderMermaidSourceProject(&b, seen, scope, srcID, srcProject)
				}
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
	SrcInterconnect string                `json:"src_interconnect"`
	Mapped          bool                  `json:"mapped"`
	SrcRegion       string                `json:"src_region"`
	SrcState        string                `json:"src_state"`
	DstProjects     []jsonDestinationNode `json:"dst_projects,omitempty"`
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
	DstVLANAttachment         string `json:"dst_vlan_attachment"`
	DstVLANAttachmentState    string `json:"dst_vlan_attachment_state"`
	DstVLANAttachmentVLANID   string `json:"dst_vlan_attachment_vlanid"`
	DstCloudRouter            string `json:"dst_cloud_router"`
	DstCloudRouterInterface   string `json:"dst_cloud_router_interface"`
	DstCloudRouterInterfaceIP string `json:"dst_cloud_router_interface_ip"`
	RemoteBGPPeer             string `json:"remote_bgp_peer"`
	RemoteBGPPeerIP           string `json:"remote_bgp_peer_ip"`
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
	SrcInterconnect string
	Mapped          bool
	SrcRegion       string
	SrcState        string
	DstProjects     []destinationGroup
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
	DstVLANAttachment         string
	DstVLANAttachmentState    string
	DstVLANAttachmentVLANID   string
	DstCloudRouter            string
	DstCloudRouterInterface   string
	DstCloudRouterInterfaceIP string
	RemoteBGPPeer             string
	RemoteBGPPeerIP           string
	BGPPeeringStatus          string
}

type mermaidCollapse struct {
	SharedProject string
	SharedRegion  string
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
			SrcInterconnect: valueOrUnknown(interconnect.SrcInterconnect),
			Mapped:          interconnect.Mapped,
			SrcRegion:       valueOrUnknown(interconnect.SrcRegion),
			SrcState:        valueOrUnknown(interconnect.SrcState),
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
						DstVLANAttachment:         valueOrUnknown(attachment.DstVLANAttachment),
						DstVLANAttachmentState:    valueOrUnknown(attachment.DstVLANAttachmentState),
						DstVLANAttachmentVLANID:   valueOrUnknown(attachment.DstVLANAttachmentVLANID),
						DstCloudRouter:            valueOrUnknown(attachment.DstCloudRouter),
						DstCloudRouterInterface:   valueOrUnknown(attachment.DstCloudRouterInterface),
						DstCloudRouterInterfaceIP: valueOrUnknown(attachment.DstCloudRouterInterfaceIP),
						RemoteBGPPeer:             valueOrUnknown(attachment.RemoteBGPPeer),
						RemoteBGPPeerIP:           valueOrUnknown(attachment.RemoteBGPPeerIP),
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
			SrcInterconnect: name,
			SrcRegion:       groupItems[0].SrcRegion,
			SrcState:        groupItems[0].SrcState,
			DstProjects:     groupDestinations(groupItems),
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
				DstVLANAttachment:         item.DstVLANAttachment,
				DstVLANAttachmentState:    item.DstVLANAttachmentState,
				DstVLANAttachmentVLANID:   item.DstVLANAttachmentVLANID,
				DstCloudRouter:            item.DstCloudRouter,
				DstCloudRouterInterface:   item.DstCloudRouterInterface,
				DstCloudRouterInterfaceIP: item.DstCloudRouterInterfaceIP,
				RemoteBGPPeer:             item.RemoteBGPPeer,
				RemoteBGPPeerIP:           item.RemoteBGPPeerIP,
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
		"%s%s src_interconnect: %s [mapped: %t, src_region: %s, src_state: %s]\n",
		indent,
		prefix,
		valueOrUnknown(interconnect.SrcInterconnect),
		interconnect.Mapped,
		valueOrUnknown(interconnect.SrcRegion),
		valueOrUnknown(interconnect.SrcState),
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
		"%s%s dst_vlan_attachment: %s [dst_vlan_attachment_state: %s, dst_vlan_attachment_vlanid: %s]\n",
		indent,
		prefix,
		valueOrUnknown(attachment.DstVLANAttachment),
		valueOrUnknown(attachment.DstVLANAttachmentState),
		valueOrUnknown(attachment.DstVLANAttachmentVLANID),
	)
	fmt.Fprintf(
		b,
		"%s`-- dst_cloud_router: %s\n",
		childIndent,
		valueOrUnknown(attachment.DstCloudRouter),
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
		"%s        `-- remote_bgp_peer: %s [remote_bgp_peer_ip: %s, bgp_peering_status: %s]\n",
		childIndent,
		valueOrUnknown(attachment.RemoteBGPPeer),
		valueOrUnknown(attachment.RemoteBGPPeerIP),
		valueOrUnknown(attachment.BGPPeeringStatus),
	)
}

func renderMermaidSourceProject(b *strings.Builder, seen map[string]struct{}, scope, srcID string, srcProject sourceGroup) {
	collapse := sharedMermaidNodes(srcProject.Interconnects)
	for _, interconnect := range srcProject.Interconnects {
		interconnectID := mermaidID("ic-" + scope + "-" + interconnect.SrcInterconnect)
		linkIfMissing(b, seen, srcID, interconnectID, interconnectNodeLabel(interconnect))
		renderMermaidInterconnect(b, seen, scope, interconnectID, interconnect, collapse)
	}
}

func renderMermaidInterconnect(b *strings.Builder, seen map[string]struct{}, scope, interconnectID string, interconnect interconnectGroup, collapse mermaidCollapse) {
	for _, dst := range interconnect.DstProjects {
		dstProjectID := mermaidID("dst-project-" + scope + "-" + interconnect.SrcInterconnect + "-" + dst.DstProject)
		if collapse.SharedProject != "" && dst.Mapped {
			dstProjectID = mermaidID("dst-project-" + scope + "-" + dst.DstProject)
		}
		linkIfMissing(b, seen, interconnectID, dstProjectID, destinationProjectNodeLabel(dst))
		if !dst.Mapped {
			unmappedID := mermaidID("unmapped-" + scope + "-" + interconnect.SrcInterconnect + "-" + dst.DstProject)
			linkIfMissing(b, seen, dstProjectID, unmappedID, "unmapped")
			continue
		}

		for _, region := range dst.DstRegions {
			regionID := mermaidID("dst-region-" + scope + "-" + interconnect.SrcInterconnect + "-" + dst.DstProject + "-" + region.DstRegion)
			if collapse.SharedRegion != "" {
				regionID = mermaidID("dst-region-" + scope + "-" + region.DstRegion)
			}
			linkIfMissing(b, seen, dstProjectID, regionID, destinationRegionNodeLabel(region))
			for _, attachment := range region.DstVLANAttachments {
				attachmentID := mermaidID("attachment-" + scope + "-" + interconnect.SrcInterconnect + "-" + dst.DstProject + "-" + region.DstRegion + "-" + attachment.DstVLANAttachment)
				routerID := mermaidID("router-" + scope + "-" + interconnect.SrcInterconnect + "-" + dst.DstProject + "-" + region.DstRegion + "-" + attachment.DstVLANAttachment + "-" + attachment.DstCloudRouter)
				interfaceID := mermaidID("interface-" + scope + "-" + interconnect.SrcInterconnect + "-" + dst.DstProject + "-" + region.DstRegion + "-" + attachment.DstVLANAttachment + "-" + attachment.DstCloudRouter + "-" + attachment.DstCloudRouterInterface)
				peerID := mermaidID("peer-" + scope + "-" + interconnect.SrcInterconnect + "-" + dst.DstProject + "-" + region.DstRegion + "-" + attachment.DstVLANAttachment + "-" + attachment.DstCloudRouter + "-" + attachment.DstCloudRouterInterface + "-" + attachment.RemoteBGPPeer + "-" + attachment.RemoteBGPPeerIP)

				linkIfMissing(b, seen, regionID, attachmentID, attachmentNodeLabel(attachment))
				linkIfMissing(b, seen, attachmentID, routerID, routerNodeLabel(attachment))
				if hasInterface(attachment) {
					linkIfMissing(b, seen, routerID, interfaceID, interfaceNodeLabel(attachment))
				}
				if hasPeer(attachment) {
					parentID := routerID
					if hasInterface(attachment) {
						parentID = interfaceID
					}
					linkIfMissing(b, seen, parentID, peerID, peerNodeLabel(attachment))
				}
			}
		}
	}
}

func sharedMermaidNodes(interconnects []interconnectGroup) mermaidCollapse {
	projects := make(map[string]struct{})
	regions := make(map[string]struct{})
	mappedProjects := 0
	for _, interconnect := range interconnects {
		for _, dst := range interconnect.DstProjects {
			if !dst.Mapped {
				continue
			}
			projects[dst.DstProject] = struct{}{}
			mappedProjects++
			for _, region := range dst.DstRegions {
				regions[region.DstRegion] = struct{}{}
			}
		}
	}

	var collapse mermaidCollapse
	if mappedProjects > 0 && len(projects) == 1 {
		collapse.SharedProject = soleKey(projects)
	}
	if mappedProjects > 0 && len(regions) == 1 {
		collapse.SharedRegion = soleKey(regions)
	}
	return collapse
}

func interconnectNodeLabel(interconnect interconnectGroup) string {
	return fmt.Sprintf(
		"src_interconnect: %s\\nsrc_region: %s\\nsrc_state: %s",
		valueOrUnknown(interconnect.SrcInterconnect),
		valueOrUnknown(interconnect.SrcRegion),
		valueOrUnknown(interconnect.SrcState),
	)
}

func destinationProjectNodeLabel(dst destinationGroup) string {
	return fmt.Sprintf(
		"dst_project: %s\\nmapped: %t",
		valueOrUnknown(dst.DstProject),
		dst.Mapped,
	)
}

func destinationRegionNodeLabel(region regionGroup) string {
	return "dst_region: " + valueOrUnknown(region.DstRegion)
}

func attachmentNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"dst_vlan_attachment: %s\\ndst_vlan_attachment_state: %s\\ndst_vlan_attachment_vlanid: %s",
		valueOrUnknown(attachment.DstVLANAttachment),
		valueOrUnknown(attachment.DstVLANAttachmentState),
		valueOrUnknown(attachment.DstVLANAttachmentVLANID),
	)
}

func routerNodeLabel(attachment attachmentGroup) string {
	return "dst_cloud_router: " + valueOrUnknown(attachment.DstCloudRouter)
}

func interfaceNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"dst_cloud_router_interface: %s\\ndst_cloud_router_interface_ip: %s",
		valueOrUnknown(attachment.DstCloudRouterInterface),
		valueOrUnknown(attachment.DstCloudRouterInterfaceIP),
	)
}

func peerNodeLabel(attachment attachmentGroup) string {
	return fmt.Sprintf(
		"remote_bgp_peer: %s\\nremote_bgp_peer_ip: %s\\nbgp_peering_status: %s",
		valueOrUnknown(attachment.RemoteBGPPeer),
		valueOrUnknown(attachment.RemoteBGPPeerIP),
		valueOrUnknown(attachment.BGPPeeringStatus),
	)
}

func hasInterface(attachment attachmentGroup) bool {
	return strings.TrimSpace(attachment.DstCloudRouterInterface) != "" || strings.TrimSpace(attachment.DstCloudRouterInterfaceIP) != ""
}

func hasPeer(attachment attachmentGroup) bool {
	return strings.TrimSpace(attachment.RemoteBGPPeer) != "" || strings.TrimSpace(attachment.RemoteBGPPeerIP) != "" || strings.TrimSpace(attachment.BGPPeeringStatus) != ""
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
