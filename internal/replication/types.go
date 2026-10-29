package replication

// RegionPeerInfo represents peer sync telemetry.
type RegionPeerInfo struct {
	RegionID          string  `json:"region_id"`
	Name              string  `json:"name"`
	Online            bool    `json:"online"`
	LatencyMs         float64 `json:"latency_ms"`
	LagMs             int64   `json:"lag_ms"`
	RecordsSynced     uint64  `json:"records_synced"`
	ViolationsBlocked uint64  `json:"violations_blocked"`
	InSync            bool    `json:"in_sync"`
	MerkleParity      bool    `json:"merkle_parity"`
	SovereignBoundary string  `json:"sovereign_boundary"`
}

// ReplicationStatusSummary provides HUD telemetry.
type ReplicationStatusSummary struct {
	LocalRegion     string           `json:"local_region"`
	TotalRecords    int              `json:"total_records"`
	SovereignCount  int              `json:"sovereign_count"`
	GlobalCount     int              `json:"global_count"`
	MerkleRootHash  string           `json:"merkle_root_hash"`
	TotalViolations uint64           `json:"total_violations_blocked"`
	Peers           []RegionPeerInfo `json:"peers"`
}

// DatabasePeerDTO represents peer sync telemetry for the multi-region database engine.
type DatabasePeerDTO struct {
	RegionID          string  `json:"region_id"`
	PoPName           string  `json:"pop_name"`
	SubseaCable       string  `json:"subsea_cable"`
	LatencyMs         float64 `json:"latency_ms"`
	ReplicationLagMs  int64   `json:"replication_lag_ms"`
	AnalyticalSynced  uint64  `json:"analytical_synced"`
	TxSynced          uint64  `json:"tx_synced"`
	ViolationsBlocked uint64  `json:"violations_blocked"`
	PQCAttestationOK  bool    `json:"pqc_attestation_ok"`
	InSync            bool    `json:"in_sync"`
	MerkleParity      bool    `json:"merkle_parity"`
}

// DatabaseStatusDTO represents the complete HUD state of the sovereign database engine.
type DatabaseStatusDTO struct {
	LocalRegion         string                     `json:"local_region"`
	PoPName             string                     `json:"pop_name"`
	UptimeSeconds       int64                      `json:"uptime_seconds"`
	AnalyticalTotal     uint64                     `json:"analytical_total_records"`
	AnalyticalSovereign uint64                     `json:"analytical_sovereign_records"`
	AnalyticalGlobal    uint64                     `json:"analytical_global_records"`
	AnalyticalBlocks    uint64                     `json:"analytical_blocks_count"`
	TxCommitted         uint64                     `json:"tx_committed_count"`
	TxAborted           uint64                     `json:"tx_aborted_count"`
	LatestWALSeq        uint64                     `json:"latest_wal_seq"`
	LatestWALHash       string                     `json:"latest_wal_hash"`
	PQCAttestation      string                     `json:"pqc_attestation"`
	TotalViolations     uint64                     `json:"total_violations_blocked"`
	Peers               map[string]DatabasePeerDTO `json:"peers"`
}
