package dr

import "time"

// TenantDRStatus defines the CLI representation of a tenant's DR state.
type TenantDRStatus struct {
	TenantID                  string     `json:"tenant_id"`
	CurrentActiveRegion       string     `json:"current_active_region"`
	PrimaryRegion             string     `json:"primary_region"`
	SecondaryRegion           string     `json:"secondary_region"`
	State                     string     `json:"state"`
	CurrentRPOEstimateSeconds float64    `json:"current_rpo_estimate_seconds"`
	CurrentRTOEstimateSeconds float64    `json:"current_rto_estimate_seconds"`
	PrimaryHealthScore        float64    `json:"primary_health_score"`
	SecondaryHealthScore      float64    `json:"secondary_health_score"`
	WitnessConsensus          string     `json:"witness_consensus"`
	LastFailoverAt            *time.Time `json:"last_failover_at,omitempty"`
	LastFailoverReason        string     `json:"last_failover_reason,omitempty"`
	IsSovereigntyPreserved    bool       `json:"is_sovereignty_preserved"`
	UpdatedAt                 time.Time  `json:"updated_at"`
}

// TenantDRPolicy defines the disaster recovery policy for a tenant.
type TenantDRPolicy struct {
	TenantID                 string     `json:"tenant_id"`
	PrimaryRegion            string     `json:"primary_region"`
	SecondaryRegion          string     `json:"secondary_region"`
	TertiaryRegion           string     `json:"tertiary_region,omitempty"`
	FailoverMode             string     `json:"failover_mode"`
	TargetRPOSeconds         float64    `json:"target_rpo_seconds"`
	TargetRTOSeconds         float64    `json:"target_rto_seconds"`
	SovereigntyLock          bool       `json:"sovereignty_lock"`
	AutoFailoverEnabled      bool       `json:"auto_failover_enabled"`
	HealthGracePeriodSeconds int        `json:"health_grace_period_seconds"`
	MinWitnessQuorum         int        `json:"min_witness_quorum"`
	LastRehearsalAt          *time.Time `json:"last_rehearsal_at,omitempty"`
	RehearsalScore           float64    `json:"rehearsal_score"`
}

// DRRehearsalReport defines the outcome of an automated DR path simulation.
type DRRehearsalReport struct {
	RehearsalID                string    `json:"rehearsal_id"`
	TenantID                   string    `json:"tenant_id"`
	Timestamp                  time.Time `json:"timestamp"`
	SimulatedSourceRegion      string    `json:"simulated_source_region"`
	SimulatedTargetRegion      string    `json:"simulated_target_region"`
	ReplicationLagSeconds      float64   `json:"replication_lag_seconds"`
	SecondaryCapacityAvailable bool      `json:"secondary_capacity_available"`
	SovereigntyCompliant       bool      `json:"sovereignty_compliant"`
	BGPWithdrawalSimulated     bool      `json:"bgp_withdrawal_simulated"`
	SimulatedRTOSeconds        float64   `json:"simulated_rto_seconds"`
	Passed                     bool      `json:"passed"`
	DigitalSealHMAC            string    `json:"digital_seal_hmac"`
	Details                    string    `json:"details"`
}

// DREvent defines a journal entry for audit logging.
type DREvent struct {
	EventID      string    `json:"event_id"`
	TenantID     string    `json:"tenant_id"`
	EventType    string    `json:"event_type"`
	FromRegion   string    `json:"from_region"`
	ToRegion     string    `json:"to_region"`
	Trigger      string    `json:"trigger"`
	Timestamp    time.Time `json:"timestamp"`
	PreviousHMAC string    `json:"previous_hmac"`
	EventHMAC    string    `json:"event_hmac"`
	Actor        string    `json:"actor"`
	Details      string    `json:"details"`
}

// TenantIsolationProfile describes tenant isolation rules.
type TenantIsolationProfile struct {
	TenantID           string    `json:"tenant_id"`
	TenantName         string    `json:"tenant_name"`
	Jurisdiction       string    `json:"jurisdiction"`
	IsolationLevel     string    `json:"isolation_level"`
	VNetID             string    `json:"vnet_id"`
	WireGuardPeerGroup string    `json:"wireguard_peer_group"`
	DedicatedNodePool  string    `json:"dedicated_node_pool,omitempty"`
	AllowedRegions     []string  `json:"allowed_regions"`
	EgressRestricted   bool      `json:"egress_restricted"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
