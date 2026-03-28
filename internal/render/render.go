package render

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"mindmap/internal/model"
)

const (
	FormatMermaid = "mermaid"
	FormatCSV     = "csv"
	FormatTSV     = "tsv"
	FormatJSON    = "json"
	FormatTree    = "tree"
)

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
	header := []string{
		"org",
		"src_project",
		"src_interconnect",
		"dst_project",
		"region",
		"src_region",
		"src_state",
		"attachment",
		"attachment_state",
		"router",
		"interface",
		"local_ip",
		"bgp_peer_name",
		"remote_ip",
		"bgp_status",
		"mapped",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, item := range report.Items {
		record := []string{
			report.Selectors.Org,
			item.SrcProject,
			item.SrcInterconnect,
			item.DstProject,
			item.Region,
			item.SrcRegion,
			item.SrcState,
			item.Attachment,
			item.AttachmentState,
			item.Router,
			item.Interface,
			item.LocalIP,
			item.BGPPeerName,
			item.RemoteIP,
			item.BGPStatus,
			fmt.Sprintf("%t", item.Mapped),
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
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", report.Selectors.Org)
	fmt.Fprintf(&b, "`-- %s\n", report.SourceProject)
	interconnects := groupInterconnects(report)
	for idx, interconnect := range interconnects {
		drawTreeInterconnect(&b, interconnect, idx == len(interconnects)-1)
	}
	return []byte(b.String())
}

func renderMermaid(report model.Report) []byte {
	var b strings.Builder
	b.WriteString("flowchart LR\n")

	orgID := mermaidID("org-" + report.Selectors.Org)
	srcID := mermaidID("src-" + report.SourceProject)
	fmt.Fprintf(&b, "    %s[%q]\n", orgID, report.Selectors.Org)
	fmt.Fprintf(&b, "    %s[%q]\n", srcID, report.SourceProject)
	fmt.Fprintf(&b, "    %s --> %s\n", orgID, srcID)

	seen := make(map[string]struct{})
	for _, item := range report.Items {
		interconnectID := mermaidID("ic-" + item.SrcInterconnect)
		linkIfMissing(&b, seen, srcID, interconnectID, interconnectLabel(item))
		if !item.Mapped {
			unmappedID := mermaidID("unmapped-" + item.SrcInterconnect + "-" + item.DstProject)
			linkIfMissing(&b, seen, interconnectID, unmappedID, fmt.Sprintf("%s\\nunmapped", valueOrUnknown(item.DstProject)))
			continue
		}

		dstID := mermaidID("dst-" + item.SrcInterconnect + "-" + item.DstProject)
		regionID := mermaidID("region-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.Region)
		attachmentID := mermaidID("attachment-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.Region + "-" + item.Attachment)
		endpointID := mermaidID("endpoint-" + item.SrcInterconnect + "-" + item.DstProject + "-" + item.Region + "-" + item.Attachment + "-" + item.Interface + "-" + item.BGPPeerName + "-" + item.RemoteIP)

		linkIfMissing(&b, seen, interconnectID, dstID, valueOrUnknown(item.DstProject))
		linkIfMissing(&b, seen, dstID, regionID, valueOrUnknown(item.Region))
		linkIfMissing(&b, seen, regionID, attachmentID, attachmentLabel(item))
		linkIfMissing(&b, seen, attachmentID, endpointID, endpointLabel(item))
	}
	return []byte(b.String())
}

type jsonReport struct {
	Type string      `json:"type"`
	Org  jsonOrgNode `json:"org"`
}

type jsonOrgNode struct {
	Name        string           `json:"name"`
	SrcProjects []jsonSourceNode `json:"src_projects"`
}

type jsonSourceNode struct {
	Name                   string                 `json:"name"`
	DedicatedInterconnects []jsonInterconnectNode `json:"dedicated_interconnects"`
}

type jsonInterconnectNode struct {
	Name                string                `json:"name"`
	SrcRegion           string                `json:"src_region"`
	SrcState            string                `json:"src_state"`
	DestinationProjects []jsonDestinationNode `json:"dst_projects"`
}

type jsonDestinationNode struct {
	Name    string           `json:"name"`
	Mapped  bool             `json:"mapped"`
	Regions []jsonRegionNode `json:"regions,omitempty"`
}

type jsonRegionNode struct {
	Name        string               `json:"name"`
	Attachments []jsonAttachmentNode `json:"attachments,omitempty"`
}

type jsonAttachmentNode struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	Router    string `json:"router"`
	Interface string `json:"interface"`
	LocalIP   string `json:"local_ip"`
	BGPPeer   string `json:"bgp_peer_name"`
	RemoteIP  string `json:"remote_ip"`
	BGPStatus string `json:"bgp_status"`
}

type interconnectGroup struct {
	Name             string
	SrcRegion        string
	SrcState         string
	DestinationNodes []destinationGroup
}

type destinationGroup struct {
	Name    string
	Mapped  bool
	Regions []regionGroup
}

type regionGroup struct {
	Name        string
	Attachments []attachmentGroup
}

type attachmentGroup struct {
	Name      string
	State     string
	Router    string
	Interface string
	LocalIP   string
	BGPPeer   string
	RemoteIP  string
	BGPStatus string
}

func buildJSONReport(report model.Report) jsonReport {
	interconnects := groupInterconnects(report)
	result := jsonReport{
		Type: report.Type,
		Org: jsonOrgNode{
			Name: report.Selectors.Org,
			SrcProjects: []jsonSourceNode{{
				Name:                   report.SourceProject,
				DedicatedInterconnects: make([]jsonInterconnectNode, 0, len(interconnects)),
			}},
		},
	}
	for _, interconnect := range interconnects {
		node := jsonInterconnectNode{
			Name:      interconnect.Name,
			SrcRegion: interconnect.SrcRegion,
			SrcState:  interconnect.SrcState,
		}
		for _, dst := range interconnect.DestinationNodes {
			dstNode := jsonDestinationNode{
				Name:   dst.Name,
				Mapped: dst.Mapped,
			}
			for _, region := range dst.Regions {
				regionNode := jsonRegionNode{Name: region.Name}
				for _, attachment := range region.Attachments {
					regionNode.Attachments = append(regionNode.Attachments, jsonAttachmentNode{
						Name:      attachment.Name,
						State:     attachment.State,
						Router:    attachment.Router,
						Interface: attachment.Interface,
						LocalIP:   attachment.LocalIP,
						BGPPeer:   attachment.BGPPeer,
						RemoteIP:  attachment.RemoteIP,
						BGPStatus: attachment.BGPStatus,
					})
				}
				dstNode.Regions = append(dstNode.Regions, regionNode)
			}
			node.DestinationProjects = append(node.DestinationProjects, dstNode)
		}
		result.Org.SrcProjects[0].DedicatedInterconnects = append(result.Org.SrcProjects[0].DedicatedInterconnects, node)
	}
	return result
}

func groupInterconnects(report model.Report) []interconnectGroup {
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range report.Items {
		if _, ok := grouped[item.SrcInterconnect]; !ok {
			names = append(names, item.SrcInterconnect)
		}
		grouped[item.SrcInterconnect] = append(grouped[item.SrcInterconnect], item)
	}
	sort.Strings(names)

	result := make([]interconnectGroup, 0, len(names))
	for _, name := range names {
		items := grouped[name]
		if len(items) == 0 {
			continue
		}
		result = append(result, interconnectGroup{
			Name:             name,
			SrcRegion:        items[0].SrcRegion,
			SrcState:         items[0].SrcState,
			DestinationNodes: groupDestinations(items),
		})
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
		dst := destinationGroup{Name: name}
		for _, item := range dstItems {
			if item.Mapped {
				dst.Mapped = true
				break
			}
		}
		if dst.Mapped {
			dst.Regions = groupRegions(dstItems)
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
		if _, ok := grouped[item.Region]; !ok {
			names = append(names, item.Region)
		}
		grouped[item.Region] = append(grouped[item.Region], item)
	}
	sort.Strings(names)

	result := make([]regionGroup, 0, len(names))
	for _, name := range names {
		region := regionGroup{Name: name}
		for _, item := range grouped[name] {
			region.Attachments = append(region.Attachments, attachmentGroup{
				Name:      item.Attachment,
				State:     item.AttachmentState,
				Router:    item.Router,
				Interface: item.Interface,
				LocalIP:   item.LocalIP,
				BGPPeer:   item.BGPPeerName,
				RemoteIP:  item.RemoteIP,
				BGPStatus: item.BGPStatus,
			})
		}
		result = append(result, region)
	}
	return result
}

func drawTreeInterconnect(b *strings.Builder, interconnect interconnectGroup, isLast bool) {
	prefix := "|--"
	childPrefix := "|   "
	if isLast {
		prefix = "`--"
		childPrefix = "    "
	}
	fmt.Fprintf(b, "    %s %s [%s]\n", prefix, interconnect.Name, valueOrUnknown(interconnect.SrcState))
	for idx, dst := range interconnect.DestinationNodes {
		lastDestination := idx == len(interconnect.DestinationNodes)-1
		dstPrefix := childPrefix + "|--"
		regionIndent := childPrefix + "|   "
		if lastDestination {
			dstPrefix = childPrefix + "`--"
			regionIndent = childPrefix + "    "
		}
		fmt.Fprintf(b, "%s %s\n", dstPrefix, valueOrUnknown(dst.Name))
		if !dst.Mapped {
			fmt.Fprintf(b, "%s `-- unmapped\n", regionIndent)
			continue
		}
		for regionIdx, region := range dst.Regions {
			lastRegion := regionIdx == len(dst.Regions)-1
			regionPrefix := regionIndent + "|--"
			attachmentIndent := regionIndent + "|   "
			if lastRegion {
				regionPrefix = regionIndent + "`--"
				attachmentIndent = regionIndent + "    "
			}
			fmt.Fprintf(b, "%s %s\n", regionPrefix, valueOrUnknown(region.Name))
			for attachmentIdx, attachment := range region.Attachments {
				lastAttachment := attachmentIdx == len(region.Attachments)-1
				attachmentPrefix := attachmentIndent + "|--"
				detailIndent := attachmentIndent + "|   "
				if lastAttachment {
					attachmentPrefix = attachmentIndent + "`--"
					detailIndent = attachmentIndent + "    "
				}
				fmt.Fprintf(b, "%s attachment: %s [%s]\n", attachmentPrefix, valueOrUnknown(attachment.Name), valueOrUnknown(attachment.State))
				fmt.Fprintf(b, "%s router: %s\n", detailIndent, valueOrUnknown(attachment.Router))
				fmt.Fprintf(b, "%s interface/local_ip: %s / %s\n", detailIndent, valueOrUnknown(attachment.Interface), valueOrUnknown(attachment.LocalIP))
				fmt.Fprintf(b, "%s bgp_peer/remote_ip/status: %s / %s / %s\n", detailIndent, valueOrUnknown(attachment.BGPPeer), valueOrUnknown(attachment.RemoteIP), valueOrUnknown(attachment.BGPStatus))
			}
		}
	}
}

func interconnectLabel(item model.MappingItem) string {
	return fmt.Sprintf("%s\\n%s", item.SrcInterconnect, valueOrUnknown(item.SrcState))
}

func attachmentLabel(item model.MappingItem) string {
	return fmt.Sprintf("%s\\nrouter: %s\\n%s", valueOrUnknown(item.Attachment), valueOrUnknown(item.Router), valueOrUnknown(item.AttachmentState))
}

func endpointLabel(item model.MappingItem) string {
	return fmt.Sprintf("if: %s | local: %s\\nbgp: %s | remote: %s | %s",
		valueOrUnknown(item.Interface),
		valueOrUnknown(item.LocalIP),
		valueOrUnknown(item.BGPPeerName),
		valueOrUnknown(item.RemoteIP),
		valueOrUnknown(item.BGPStatus),
	)
}

func linkIfMissing(b *strings.Builder, seen map[string]struct{}, parentID, childID, childLabel string) {
	nodeKey := "node:" + childID
	if _, ok := seen[nodeKey]; !ok {
		fmt.Fprintf(b, "    %s[%q]\n", childID, childLabel)
		seen[nodeKey] = struct{}{}
	}
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
