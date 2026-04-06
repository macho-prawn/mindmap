package model

type Selectors struct {
	Org         string `json:"org"`
	Workload    string `json:"workload"`
	Environment string `json:"environment"`
}

type DedicatedInterconnect struct {
	Name          string
	State         string
	MacsecEnabled bool
	MacsecKeyName string
}

type VLANAttachment struct {
	Name         string
	Region       string
	Network      string
	State        string
	Interconnect string
	Router       string
	VLANID       string
}

type RouterInterface struct {
	Name                     string
	LinkedInterconnectAttach string
	LinkedVPNTunnel          string
	IPRange                  string
}

type BGPPeer struct {
	Name         string
	Interface    string
	LocalIP      string
	RemoteIP     string
	PeerASN      string
	SessionState string
}

type CloudRouter struct {
	Name       string
	Region     string
	Network    string
	ASN        string
	Interfaces []RouterInterface
	BGPPeers   []BGPPeer
}

type VPNGateway struct {
	Name     string
	Region   string
	Network  string
	Type     string
	Status   string
	SelfLink string
}

type VPNTunnel struct {
	Name                string
	Region              string
	Status              string
	SelfLink            string
	Router              string
	VPNGateway          string
	TargetVPNGateway    string
	PeerGCPGateway      string
	PeerExternalGateway string
	PeerIP              string
	VPNGatewayInterface string
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
	SrcVPNGateway             string `json:"src_vpn_gateway"`
	SrcVPNGatewayType         string `json:"src_vpn_gateway_type"`
	SrcVPNGatewayStatus       string `json:"src_vpn_gateway_status"`
	SrcVPNTunnel              string `json:"src_vpn_tunnel"`
	SrcVPNTunnelStatus        string `json:"src_vpn_tunnel_status"`
	Mapped                    bool   `json:"mapped"`
	SrcRegion                 string `json:"src_region"`
	SrcState                  string `json:"src_state"`
	SrcMacsecEnabled          bool   `json:"src_macsec_enabled"`
	SrcMacsecKeyName          string `json:"src_macsec_keyname"`
	DstProject                string `json:"dst_project"`
	DstRegion                 string `json:"dst_region"`
	DstVPC                    string `json:"dst_vpc"`
	DstVLANAttachment         string `json:"dst_vlan_attachment"`
	DstVLANAttachmentState    string `json:"dst_vlan_attachment_state"`
	DstVLANAttachmentVLANID   string `json:"dst_vlan_attachment_vlanid"`
	DstVPNGateway             string `json:"dst_vpn_gateway"`
	DstVPNGatewayType         string `json:"dst_vpn_gateway_type"`
	DstVPNGatewayStatus       string `json:"dst_vpn_gateway_status"`
	DstVPNTunnel              string `json:"dst_vpn_tunnel"`
	DstVPNTunnelStatus        string `json:"dst_vpn_tunnel_status"`
	DstCloudRouter            string `json:"dst_cloud_router"`
	DstCloudRouterASN         string `json:"dst_cloud_router_asn"`
	DstCloudRouterInterface   string `json:"dst_cloud_router_interface"`
	DstCloudRouterInterfaceIP string `json:"dst_cloud_router_interface_ip"`
	RemoteBGPPeer             string `json:"remote_bgp_peer"`
	RemoteBGPPeerIP           string `json:"remote_bgp_peer_ip"`
	RemoteBGPPeerASN          string `json:"remote_bgp_peer_asn"`
	BGPPeeringStatus          string `json:"bgp_peering_status"`
}

type Report struct {
	Type               string        `json:"type"`
	SourceProject      string        `json:"source_project,omitempty"`
	DestinationProject string        `json:"destination_project,omitempty"`
	Selectors          Selectors     `json:"selectors"`
	Items              []MappingItem `json:"items"`
}
