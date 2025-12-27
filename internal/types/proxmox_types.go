package types

// Below are generated from pvesh get /cluster/status --output-format json | jq
type ProxmoxCluster struct {
	ID          string `json:"id" db:"id"`                       // PocketBase generated UUID
	ClusterName string `json:"cluster_name" db:"cluster_name"`   // name of the proxmox cluster
	Nodes       int    `json:"nodes" db:"nodes"`                 // number of nodes
	Quorate     int    `json:"quorate" db:"quorate"`             // Quorate
	Version     int    `json:"version" db:"version"`             // Version of cluster config
	ClusterID   string `json:"px_cluster_id" db:"px_cluster_id"` // id of the proxmox cluster
}

type ProxmoxNode struct {
	ID         string `json:"id" db:"id"`                   // PocketBase generated UUID
	AgentID    string `json:"agent_id" db:"agent_id"`       // Target agent
	ClusterID  string `json:"cluster_id" db:"cluster_id"`   // ID of the proxmox cluster
	IP         string `json:"ip" db:"ip"`                   // IP of the node
	Level      int    `json:"level" db:"level"`             // Level
	Local      int    `json:"local" db:"local"`             // Local
	Name       string `json:"name" db:"name"`               // name of the proxmox node
	NodeID     int    `json:"px_node_id" db:"px_node_id"`   // Proxmox id of the node
	Online     int    `json:"online" db:"online"`           // whether node is online
	PVEVersion int    `json:"pve_version" db:"pve_version"` // Version of proxmox node
}

// Below are generated from pvesh get /cluster/resources --output-format json | jq
// and pvesh get /nodes/<node-id>/lxc/<lxc-id>/config --output-format json | jq
type ProxmoxLXC struct {
	ID        string `json:"id" db:"id"`                 // PocketBase generated UUID
	AgentID   string `json:"agent_id" db:"agent_id"`     // Target agent
	ClusterID string `json:"cluster_id" db:"cluster_id"` // ID of the proxmox cluster
	NodeID    string `json:"node_id" db:"node_id"`       // ID of the proxmox node
	Name      string `json:"name" db:"name"`             // name of the proxmox LXC
	LXCID     string `json:"lxc_id" db:"lxc_id"`         // ID of the proxmox LXC
	Status    string `json:"status" db:"status"`         // Status of the LXC
	OSType    string `json:"ostype" db:"ostype"`         // OSTYpe of the LXC
	Uptime    string `json:"uptime" db:"uptime"`         // uptime
	VmID      int    `json:"vmid" db:"vmid"`             // VMID
}

// Below are generated from pvesh get /cluster/resources --output-format json | jq
// and pvesh get /nodes/<node-id>/qemu/<qemu-id>/config --output-format json | jq
type ProxmoxQemu struct {
	ID        string `json:"id" db:"id"`                 // PocketBase generated UUID
	AgentID   string `json:"agent_id" db:"agent_id"`     // Target agent
	ClusterID string `json:"cluster_id" db:"cluster_id"` // ID of the proxmox cluster
	NodeID    string `json:"node_id" db:"node_id"`       // ID of the proxmox node
	Name      string `json:"name" db:"name"`             // name of the proxmox VM
	QemuID    string `json:"qemu_id" db:"qemu_id"`       // ID of the proxmox VM
	Status    string `json:"status" db:"status"`         // Status of the VM
	OSType    string `json:"ostype" db:"ostype"`         // OSTYpe of the VM
	Uptime    string `json:"uptime" db:"uptime"`         // uptime
	VmID      int    `json:"vmid" db:"vmid"`             // VMID
	VmGenID   string `json:"vmgenid" db:"vmgenid"`       // VMGenID
	KVM       int    `json:"kvm" db:"kvm"`               // KVM
	Boot      string `json:"boot" db:"boot"`             // boot order
	HostCPU   string `json:"hostcpu" db:"hostcpu"`       // Host CPU Type, ex ; host,x86-64-v2-AES
}
