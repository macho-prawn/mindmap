package model

type Selectors struct {
	Org         string `json:"org"`
	Workload    string `json:"workload"`
	Environment string `json:"environment"`
}

type DedicatedInterconnect struct {
	Name  string
	State string
}

type VLANAttachment struct {
	Name         string
	Region       string
	State        string
	Interconnect string
	Router       string
	VLANID       string
}

type RouterInterface struct {
	Name                     string
	LinkedInterconnectAttach string
	IPRange                  string
}

type BGPPeer struct {
	Name         string
	Interface    string
	LocalIP      string
	RemoteIP     string
	SessionState string
}

type CloudRouter struct {
	Name       string
	Region     string
	Interfaces []RouterInterface
	BGPPeers   []BGPPeer
}

type BGPPeerStatus struct {
	Name         string
	LocalIP      string
	RemoteIP     string
	SessionState string
}

type RouterStatus struct {
	RouterName string
	Region     string
	Peers      []BGPPeerStatus
}

type MappingItem struct {
	Org                       string `json:"org"`
	Workload                  string `json:"workload"`
	Environment               string `json:"environment"`
	SrcProject                string `json:"src_project"`
	SrcInterconnect           string `json:"src_interconnect"`
	Mapped                    bool   `json:"mapped"`
	SrcRegion                 string `json:"src_region"`
	SrcState                  string `json:"src_state"`
	DstProject                string `json:"dst_project"`
	DstRegion                 string `json:"dst_region"`
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

type Report struct {
	Type               string        `json:"type"`
	SourceProject      string        `json:"source_project,omitempty"`
	DestinationProject string        `json:"destination_project,omitempty"`
	Selectors          Selectors     `json:"selectors"`
	Items              []MappingItem `json:"items"`
}
