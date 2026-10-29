package dns

import "time"

// RecordType defines standard DNS resource record types.
type RecordType string

const (
	TypeA     RecordType = "A"
	TypeAAAA  RecordType = "AAAA"
	TypeCNAME RecordType = "CNAME"
	TypeTXT   RecordType = "TXT"
	TypeMX    RecordType = "MX"
	TypeCAA   RecordType = "CAA"
	TypeSRV   RecordType = "SRV"
	TypeNS    RecordType = "NS"
	TypeSOA   RecordType = "SOA"
)

// RoutingPolicy defines Anycast traffic steering algorithms.
type RoutingPolicy string

const (
	PolicySimple         RoutingPolicy = "SIMPLE"
	PolicyAnycastGeo     RoutingPolicy = "ANYCAST_GEO"
	PolicyLatencyOptimal RoutingPolicy = "LATENCY_OPTIMAL"
	PolicyFailover       RoutingPolicy = "FAILOVER"
)

// DNSRecord encapsulates an authoritative DNS resource record with Anycast metadata.
type DNSRecord struct {
	ID             string            `json:"id"`
	ZoneID         string            `json:"zone_id"`
	Name           string            `json:"name"`
	Type           RecordType        `json:"type"`
	TTL            uint32            `json:"ttl"`
	Content        string            `json:"content"`
	Priority       int               `json:"priority,omitempty"`
	Weight         int               `json:"weight,omitempty"`
	RoutingPolicy  RoutingPolicy     `json:"routing_policy"`
	GeoTargets     map[string]string `json:"geo_targets,omitempty"`
	HealthCheckURL string            `json:"health_check_url,omitempty"`
	Healthy        bool              `json:"healthy"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// DNSZone represents an authoritative forward or reverse DNS domain.
type DNSZone struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Serial    uint32    `json:"serial"`
	Status    string    `json:"status"`
	DNSSEC    bool      `json:"dnssec_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ResolvedRecord represents an answer returned to a DNS client.
type ResolvedRecord struct {
	Name    string     `json:"name"`
	Type    RecordType `json:"type"`
	TTL     uint32     `json:"ttl"`
	Content string     `json:"content"`
}

// ResolveRequest represents a query to the Anycast DNS resolution engine.
type ResolveRequest struct {
	QueryName    string     `json:"query_name"`
	RecordType   RecordType `json:"record_type"`
	ClientIP     string     `json:"client_ip,omitempty"`
	ClientRegion string     `json:"client_region,omitempty"`
}

// ResolveResponse represents the explainable Anycast routed answer.
type ResolveResponse struct {
	QueryName      string           `json:"query_name"`
	RecordType     RecordType       `json:"record_type"`
	Records        []ResolvedRecord `json:"records"`
	MatchedPolicy  RoutingPolicy    `json:"matched_policy"`
	SelectedRegion string           `json:"selected_region"`
	LatencyMs      float64          `json:"latency_ms"`
}

// PoPHealthState represents health assessment of a PoP.
type PoPHealthState string

const (
	PoPHealthy   PoPHealthState = "POP_HEALTHY"
	PoPDegraded  PoPHealthState = "POP_DEGRADED"
	PoPUnhealthy PoPHealthState = "POP_UNHEALTHY"
	PoPDraining  PoPHealthState = "POP_DRAINING"
)

// BGPAnnouncementState represents current BGP Route Health Injection posture.
type BGPAnnouncementState string

const (
	BGPAdvertised  BGPAnnouncementState = "BGP_ADVERTISED"
	BGPPrepended3X BGPAnnouncementState = "BGP_PREPENDED_3X"
	BGPWithdrawn   BGPAnnouncementState = "BGP_WITHDRAWN"
)

// PoPHealthStatus details live health, BGP state, and flap metrics for a PoP.
type PoPHealthStatus struct {
	RegionID              string               `json:"region_id"`
	Name                  string               `json:"name"`
	AnycastIPv4           string               `json:"anycast_ipv4"`
	AnycastIPv6           string               `json:"anycast_ipv6"`
	FailoverPoPID         string               `json:"failover_pop_id"`
	State                 PoPHealthState       `json:"state"`
	BGPState              BGPAnnouncementState `json:"bgp_state"`
	ConsecutiveSuccesses  int                  `json:"consecutive_successes"`
	ConsecutiveFailures   int                  `json:"consecutive_failures"`
	FlapDampingPenalty    float64              `json:"flap_damping_penalty"`
	Damped                bool                 `json:"damped"`
	LastProbeLatencyMs    float64              `json:"last_probe_latency_ms"`
	LastStateTransition   time.Time            `json:"last_state_transition"`
	SovereigntyRestricted bool                 `json:"sovereignty_restricted"`
	DrainReason           string               `json:"drain_reason,omitempty"`
}

// FailoverEvent records an automated or manual BGP route state transition.
type FailoverEvent struct {
	Sequence     int64                `json:"sequence"`
	Timestamp    time.Time            `json:"timestamp"`
	RegionID     string               `json:"region_id"`
	EventType    string               `json:"event_type"`
	OldState     PoPHealthState       `json:"old_state"`
	NewState     PoPHealthState       `json:"new_state"`
	OldBGPState  BGPAnnouncementState `json:"old_bgp_state"`
	NewBGPState  BGPAnnouncementState `json:"new_bgp_state"`
	Reason       string               `json:"reason"`
	TriggeredBy  string               `json:"triggered_by"`
	BGPWithdrawn bool                 `json:"bgp_withdrawn"`
}

// DnsDeletionRequest represents an initiated record deletion under human approval policy.
type DnsDeletionRequest struct {
	RequestID      string    `json:"requestId"`
	Domain         string    `json:"domain"`
	RecordID       string    `json:"recordId"`
	Requester      string    `json:"requester"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expiresAt"`
	ChallengeToken string    `json:"challengeToken"`
}

// DnsApprovalResult represents the outcome of an independent operator approval.
type DnsApprovalResult struct {
	Status        string    `json:"status"`
	RequestID     string    `json:"requestId"`
	ApprovedBy    string    `json:"approvedBy"`
	ApprovalToken string    `json:"approvalToken"`
	ValidUntil    time.Time `json:"validUntil"`
	Message       string    `json:"message"`
}
