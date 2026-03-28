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
	SrcProject      string `json:"src_project"`
	SrcInterconnect string `json:"src_interconnect"`
	SrcRegion       string `json:"src_region"`
	SrcState        string `json:"src_state"`
	DstProject      string `json:"dst_project"`
	Region          string `json:"region"`
	Attachment      string `json:"attachment"`
	AttachmentState string `json:"attachment_state"`
	Router          string `json:"router"`
	Interface       string `json:"interface"`
	BGPPeerName     string `json:"bgp_peer_name"`
	LocalIP         string `json:"local_ip"`
	RemoteIP        string `json:"remote_ip"`
	BGPStatus       string `json:"bgp_status"`
	Mapped          bool   `json:"mapped"`
}

type Report struct {
	Type               string        `json:"type"`
	SourceProject      string        `json:"source_project,omitempty"`
	DestinationProject string        `json:"destination_project,omitempty"`
	Selectors          Selectors     `json:"selectors"`
	Items              []MappingItem `json:"items"`
}
