package workload

import "time"

// WorkloadType defines supported execution types.
type WorkloadType string

const (
	TypeGPUInference       WorkloadType = "GPU_INFERENCE"
	TypeAPIGateway         WorkloadType = "API_GATEWAY"
	TypeContainerPod       WorkloadType = "CONTAINER_POD"
	TypeMicroVMFirecracker WorkloadType = "MICROVM_FIRECRACKER"
	TypeAgenticLoop        WorkloadType = "AGENTIC_LOOP"
	TypeStaticContent      WorkloadType = "STATIC_CONTENT"
	TypeDatabaseVolume     WorkloadType = "DATABASE_VOLUME"
)

// DataJurisdiction defines regulatory residency requirements.
type DataJurisdiction string

const (
	JurisdictionAny        DataJurisdiction = "ANY"
	JurisdictionBangladesh DataJurisdiction = "BANGLADESH" // NDMA 2026 / CII Compliant
	JurisdictionEU         DataJurisdiction = "EU"         // GDPR
	JurisdictionUS         DataJurisdiction = "US"
	JurisdictionSingapore  DataJurisdiction = "SINGAPORE"
)

// OptimizationObjective defines primary optimization weighting.
type OptimizationObjective string

const (
	ObjectiveBalanced   OptimizationObjective = "BALANCED"
	ObjectiveLowLatency OptimizationObjective = "LOW_LATENCY"
	ObjectiveMinCost    OptimizationObjective = "MIN_COST"
	ObjectiveZeroCarbon OptimizationObjective = "ZERO_CARBON"
)

// Manifest represents user-declared workload specification in YAML or JSON.
type Manifest struct {
	APIVersion string       `json:"apiVersion" yaml:"apiVersion"`
	Kind       string       `json:"kind" yaml:"kind"`
	Metadata   MetadataSpec `json:"metadata" yaml:"metadata"`
	Spec       WorkloadSpec `json:"spec" yaml:"spec"`
}

type MetadataSpec struct {
	Name      string            `json:"name" yaml:"name"`
	TenantID  string            `json:"tenant_id" yaml:"tenant_id"`
	ProjectID string            `json:"project_id" yaml:"project_id"`
	Labels    map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}

type WorkloadSpec struct {
	WorkloadID         string                `json:"workload_id" yaml:"workload_id"`
	Type               WorkloadType          `json:"type" yaml:"type"`
	Jurisdiction       DataJurisdiction      `json:"jurisdiction" yaml:"jurisdiction"`
	StrictSovereignty  bool                  `json:"strict_sovereignty" yaml:"strict_sovereignty"`
	MaxLatencyP95Ms    uint32                `json:"max_latency_p95_ms" yaml:"max_latency_p95_ms"`
	MaxCostBudgetUSD   float64               `json:"max_cost_budget_usd" yaml:"max_cost_budget_usd"`
	CPUMillicores      uint32                `json:"cpu_millicores" yaml:"cpu_millicores"`
	MemoryBytes        uint64                `json:"memory_bytes" yaml:"memory_bytes"`
	GPUCount           uint32                `json:"gpu_count" yaml:"gpu_count"`
	RequiredGPUModel   string                `json:"required_gpu_model" yaml:"required_gpu_model"`
	Objective          OptimizationObjective `json:"objective" yaml:"objective"`
	ClientOriginRegion string                `json:"client_origin_region" yaml:"client_origin_region"`
	IdempotencyKey     string                `json:"idempotency_key,omitempty" yaml:"idempotency_key,omitempty"`
}

// WorkloadStatus represents live placement and scheduling state.
type WorkloadStatus struct {
	Phase              string    `json:"phase"`
	AssignedRegion     string    `json:"assigned_region"`
	AssignedNode       string    `json:"assigned_node"`
	AssignedProvider   string    `json:"assigned_provider"`
	MeasuredLatencyMs  uint32    `json:"measured_latency_ms"`
	CurrentCostRateUSD float64   `json:"current_cost_rate_usd"`
	RoutingReason      string    `json:"routing_reason"`
	ReasonCodes        []string  `json:"reason_codes,omitempty"`
	ScheduledAt        time.Time `json:"scheduled_at"`
	LeaseExpiresAt     time.Time `json:"lease_expires_at"`
}

// ScheduleResponse represents the server response for a scheduled workload.
type ScheduleResponse struct {
	WorkloadID      string         `json:"workload_id"`
	Status          WorkloadStatus `json:"status"`
	IngressEndpoint string         `json:"ingress_endpoint"`
}

// NodeSummary provides high-level health and capacity of fabric nodes.
type NodeSummary struct {
	ID              string           `json:"id"`
	Provider        string           `json:"provider"`
	Region          string           `json:"region"`
	Jurisdiction    DataJurisdiction `json:"jurisdiction"`
	GPUModel        string           `json:"gpu_model"`
	TotalGPUs       int              `json:"total_gpus"`
	AvailableGPUs   int              `json:"available_gpus"`
	CostPerGPUHour  float64          `json:"cost_per_gpu_hour"`
	CarbonIntensity float64          `json:"carbon_intensity"`
	Healthy         bool             `json:"healthy"`
	IngressEndpoint string           `json:"ingress_endpoint"`
}

// PlacementEvent represents a cryptographically chained scheduler placement or preemption event.
type PlacementEvent struct {
	Sequence    uint64    `json:"sequence"`
	Timestamp   time.Time `json:"timestamp"`
	PrevHash    string    `json:"prev_hash"`
	CurrentHash string    `json:"current_hash"`
	EventType   string    `json:"event_type"`
	WorkloadID  string    `json:"workload_id"`
	TenantID    string    `json:"tenant_id"`
	NodeID      string    `json:"node_id"`
	Region      string    `json:"region"`
	Priority    uint32    `json:"priority"`
	Reason      string    `json:"reason"`
}

// MerkleProofStep represents a single sibling hash node in a Merkle audit path.
type MerkleProofStep struct {
	SiblingHash string `json:"sibling_hash"`
	IsRight     bool   `json:"is_right"`
}

// MerkleInclusionProof encapsulates a Zero-Egress proof proving a workload event was placed.
type MerkleInclusionProof struct {
	Sequence   uint64            `json:"sequence"`
	LeafHash   string            `json:"leaf_hash"`
	MerkleRoot string            `json:"merkle_root"`
	Path       []MerkleProofStep `json:"path"`
}

// JournalVerifyResponse encapsulates the status and Merkle root of the placement journal.
type JournalVerifyResponse struct {
	Status       string `json:"status"`
	EntriesCount int    `json:"entries_count"`
	MerkleRoot   string `json:"merkle_root"`
	Algorithm    string `json:"algorithm"`
}

// JournalProofResponse is returned by GET /v1/journal/proof.
type JournalProofResponse struct {
	Proof    *MerkleInclusionProof `json:"proof"`
	Verified bool                  `json:"verified"`
}

// RaftPeerDTO represents a remote PoP peer in the 5-region consensus cluster.
type RaftPeerDTO struct {
	ID            string    `json:"id"`
	Region        string    `json:"region"`
	Jurisdiction  string    `json:"jurisdiction"`
	SubseaCable   string    `json:"subsea_cable"`
	RTTMs         uint32    `json:"rtt_ms"`
	NextIndex     uint64    `json:"next_index"`
	MatchIndex    uint64    `json:"match_index"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	Healthy       bool      `json:"healthy"`
}

// RaftClusterStatusDTO represents high-level consensus state for the 5-PoP Raft cluster.
type RaftClusterStatusDTO struct {
	ClusterID              string        `json:"cluster_id"`
	NodeID                 string        `json:"node_id"`
	Region                 string        `json:"region"`
	Role                   string        `json:"role"`
	CurrentTerm            uint64        `json:"current_term"`
	LeaderID               string        `json:"leader_id"`
	CommitIndex            uint64        `json:"commit_index"`
	LastApplied            uint64        `json:"last_applied"`
	LogEntriesCount        int           `json:"log_entries_count"`
	LatestLogHash          string        `json:"latest_log_hash"`
	PqcAttestationVerified bool          `json:"pqc_attestation_verified"`
	TotalPeersCount        int           `json:"total_peers_count"`
	HealthyPeersCount      int           `json:"healthy_peers_count"`
	QuorumSatisfied        bool          `json:"quorum_satisfied"`
	Peers                  []RaftPeerDTO `json:"peers"`
}
