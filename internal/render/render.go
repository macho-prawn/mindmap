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
		"src_project",
		"src_interconnect",
		"src_region",
		"src_state",
		"dst_project",
		"region",
		"attachment",
		"attachment_state",
		"router",
		"interface",
		"bgp_peer_name",
		"local_ip",
		"remote_ip",
		"bgp_status",
		"mapped",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	for _, item := range report.Items {
		record := []string{
			item.SrcProject,
			item.SrcInterconnect,
			item.SrcRegion,
			item.SrcState,
			item.DstProject,
			item.Region,
			item.Attachment,
			item.AttachmentState,
			item.Router,
			item.Interface,
			item.BGPPeerName,
			item.LocalIP,
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
	return json.MarshalIndent(report, "", "  ")
}

func renderTree(report model.Report) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", report.DestinationProject)
	grouped := make(map[string][]model.MappingItem)
	var names []string
	for _, item := range report.Items {
		if _, ok := grouped[item.SrcInterconnect]; !ok {
			names = append(names, item.SrcInterconnect)
		}
		grouped[item.SrcInterconnect] = append(grouped[item.SrcInterconnect], item)
	}
	sort.Strings(names)
	for _, name := range names {
		items := grouped[name]
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "|-- %s [%s]\n", name, items[0].SrcState)
		for _, item := range items {
			if !item.Mapped {
				fmt.Fprintf(&b, "|   `-- unmapped\n")
				continue
			}
			fmt.Fprintf(&b, "|   |-- region: %s\n", valueOrUnknown(item.Region))
			fmt.Fprintf(&b, "|   |-- attachment: %s [%s]\n", valueOrUnknown(item.Attachment), valueOrUnknown(item.AttachmentState))
			fmt.Fprintf(&b, "|   |-- router: %s\n", valueOrUnknown(item.Router))
			fmt.Fprintf(&b, "|   |-- interface: %s\n", valueOrUnknown(item.Interface))
			fmt.Fprintf(&b, "|   |-- bgp_peer: %s\n", valueOrUnknown(item.BGPPeerName))
			fmt.Fprintf(&b, "|   |-- local_ip: %s\n", valueOrUnknown(item.LocalIP))
			fmt.Fprintf(&b, "|   `-- remote_ip/status: %s / %s\n", valueOrUnknown(item.RemoteIP), valueOrUnknown(item.BGPStatus))
		}
	}
	return []byte(b.String())
}

func renderMermaid(report model.Report) []byte {
	var b strings.Builder
	b.WriteString("mindmap\n")
	fmt.Fprintf(&b, "  root((GCP %s %s -> %s))\n", report.Type, report.SourceProject, report.DestinationProject)
	for _, item := range report.Items {
		fmt.Fprintf(&b, "    %s[\"%s (%s)\"]\n", mermaidID("src-"+item.SrcInterconnect), item.SrcInterconnect, item.SrcRegion)
		if !item.Mapped {
			fmt.Fprintf(&b, "      %s[\"unmapped\"]\n", mermaidID("unmapped-"+item.SrcInterconnect))
			continue
		}
		fmt.Fprintf(&b, "      %s[\"%s (%s)\"]\n", mermaidID(item.Attachment+"-"+item.Region), item.Attachment, item.Region)
		fmt.Fprintf(&b, "        %s[\"router: %s\"]\n", mermaidID(item.Router+"-"+item.Region), item.Router)
		fmt.Fprintf(&b, "          %s[\"iface: %s\"]\n", mermaidID(item.Interface+"-"+item.Region), valueOrUnknown(item.Interface))
		fmt.Fprintf(&b, "            %s[\"bgp: %s | %s | %s -> %s\"]\n",
			mermaidID(item.BGPPeerName+"-"+item.Interface+"-"+item.Region),
			valueOrUnknown(item.BGPPeerName),
			valueOrUnknown(item.BGPStatus),
			valueOrUnknown(item.LocalIP),
			valueOrUnknown(item.RemoteIP),
		)
	}
	return []byte(b.String())
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
