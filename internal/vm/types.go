package vm

// VMInstance represents an active or pre-warmed virtual machine.
type VMInstance struct {
	ID                      string   `json:"id"`
	NodeID                  string   `json:"node_id"`
	Name                    string   `json:"name"`
	TenantID                string   `json:"tenant_id"`
	WorkloadID              string   `json:"workload_id"`
	Region                  string   `json:"region"`
	Tier                    string   `json:"tier"`
	Category                string   `json:"category"`
	Hypervisor              string   `json:"hypervisor"`
	VCPUs                   uint32   `json:"vcpus"`
	MemoryMB                uint64   `json:"memory_mb"`
	HugepagesMB             uint64   `json:"hugepages_mb"`
	NUMANode                uint32   `json:"numa_node"`
	PinnedCPUs              []uint32 `json:"pinned_cpus"`
	IPAddress               string   `json:"ip_address"`
	MACAddress              string   `json:"mac_address"`
	TapDevice               string   `json:"tap_device"`
	Status                  string   `json:"status"`
	KernelVersion           string   `json:"kernel_version"`
	RootfsPath              string   `json:"rootfs_path"`
	BootTimeMs              float64  `json:"boot_time_ms"`
	CreatedAtEpoch          int64    `json:"created_at_epoch"`
	UpdatedAtEpoch          int64    `json:"updated_at_epoch"`
	SovereignClassification string   `json:"sovereign_classification"`
	AttestationToken        string   `json:"attestation_token"`
}

// NodeHost represents an edge compute host capable of hosting VMs.
type NodeHost struct {
	NodeID            string `json:"node_id"`
	Region            string `json:"region"`
	TotalVCPUs        uint32 `json:"total_vcpus"`
	AllocatedVCPUs    uint32 `json:"allocated_vcpus"`
	TotalMemoryMB     uint64 `json:"total_memory_mb"`
	AllocatedMemoryMB uint64 `json:"allocated_memory_mb"`
	NumaSockets       uint32 `json:"numa_sockets"`
	ActiveVMCount     uint32 `json:"active_vm_count"`
}

// PrewarmedPool represents a standby pool of pre-warmed snapshots for fast boot.
type PrewarmedPool struct {
	Region             string `json:"region"`
	Tier               string `json:"tier"`
	ReadyCount         uint32 `json:"ready_count"`
	TargetCount        uint32 `json:"target_count"`
	LastReplenishedSec int64  `json:"last_replenished_sec"`
}

// CreateVMRequest payload for provisioning a new VM.
type CreateVMRequest struct {
	Name                    string `json:"name"`
	TenantID                string `json:"tenant_id"`
	WorkloadID              string `json:"workload_id"`
	Region                  string `json:"region"`
	Tier                    string `json:"tier"`
	VCPUs                   uint32 `json:"vcpus,omitempty"`
	MemoryMB                uint64 `json:"memory_mb,omitempty"`
	RootfsImage             string `json:"rootfs_image,omitempty"`
	SovereignClassification string `json:"sovereign_classification,omitempty"`
}

// VMActionRequest payload for executing actions on a VM.
type VMActionRequest struct {
	Action string `json:"action"` // start, pause, resume, stop, reboot
}

// ClusterVMOverview summarizes cluster-wide virtualization metrics.
type ClusterVMOverview struct {
	TotalVMs                int    `json:"total_vms"`
	ActiveMicroVMs          int    `json:"active_micro_vms"`
	ActiveLargeVMs          int    `json:"active_large_vms"`
	TotalAllocatedVCPUs     uint32 `json:"total_allocated_vcpus"`
	TotalAllocatedMemoryMB  uint64 `json:"total_allocated_memory_mb"`
	PrewarmedPoolReady      int    `json:"prewarmed_pool_ready"`
	SovereignInstances      int    `json:"sovereign_instances"`
	PQCSovereignAttestation string `json:"pqc_sovereign_attestation"`
}
