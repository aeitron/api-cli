package security

import "time"

// DataClassification reflects sovereign sensitivity under national data protection acts.
type DataClassification string

const (
	ClassificationCII        DataClassification = "CII_CRITICAL"
	ClassificationBanking    DataClassification = "FINANCIAL_BANKING"
	ClassificationHealthcare DataClassification = "HEALTHCARE_RECORDS"
	ClassificationGeneral    DataClassification = "GENERAL_ENTERPRISE"
)

// SovereigntyEvaluationRequest represents an audit request.
type SovereigntyEvaluationRequest struct {
	TenantID            string             `json:"tenant_id"`
	ProjectID           string             `json:"project_id"`
	WorkloadID          string             `json:"workload_id"`
	SourceRegion        string             `json:"source_region"`
	TargetRegion        string             `json:"target_region"`
	Classification      DataClassification `json:"classification"`
	StrictSovereignty   bool               `json:"strict_sovereignty"`
	HasLocalDhakaMirror bool               `json:"has_local_dhaka_mirror"`
}

// SovereigntyEvaluationResult represents the audit verdict with cryptographic proof.
type SovereigntyEvaluationResult struct {
	Verdict     string    `json:"verdict"`
	Compliant   bool      `json:"compliant"`
	Standard    string    `json:"standard"`
	Reason      string    `json:"reason"`
	EvaluatedAt time.Time `json:"evaluated_at"`
	AuditHash   string    `json:"audit_hash"`
	AuditIndex  uint64    `json:"audit_index"`
}

// AuditEntry represents a tamper-proof cryptographic audit log entry.
type AuditEntry struct {
	Index       uint64    `json:"index"`
	Timestamp   time.Time `json:"timestamp"`
	PrevHash    string    `json:"prev_hash"`
	CurrentHash string    `json:"current_hash"`
	Action      string    `json:"action"`
	Subject     string    `json:"subject"`
	Actor       string    `json:"actor"`
	Verdict     string    `json:"verdict"`
	Details     string    `json:"details"`
}

// BlockedIPEntry represents an L4 firewall drop entry.
type BlockedIPEntry struct {
	IPAddress      string    `json:"ip_address"`
	Reason         string    `json:"reason"`
	DroppedPackets uint64    `json:"dropped_packets"`
	BlockedAt      time.Time `json:"blocked_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	FilterMode     string    `json:"filter_mode"`
}

// SecurityPolicy represents edge domain security posture.
type SecurityPolicy struct {
	PolicyID             string    `json:"policy_id"`
	DomainID             string    `json:"domain_id"`
	WAFEnabled           bool      `json:"waf_enabled"`
	WAFMode              string    `json:"waf_mode"`
	OWASPRulesetEnabled  bool      `json:"owasp_ruleset_enabled"`
	RateLimitEnabled     bool      `json:"rate_limit_enabled"`
	RateLimitRPM         uint32    `json:"rate_limit_rpm"`
	BurstCapacity        uint32    `json:"burst_capacity"`
	BotProtectionEnabled bool      `json:"bot_protection_enabled"`
	UpdatedAt            time.Time `json:"updated_at"`
}
