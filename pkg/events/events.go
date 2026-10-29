package events

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

// Standard CloudEvents Spec Version
const CloudEventsSpecVersion = "1.0"

// Sovereign Jurisdictions
const (
	JurisdictionBangladesh = "DATA_JURISDICTION_BANGLADESH"
	JurisdictionEU         = "DATA_JURISDICTION_EU"
	JurisdictionSingapore  = "DATA_JURISDICTION_SINGAPORE"
	JurisdictionUS         = "DATA_JURISDICTION_US"
	JurisdictionAny        = "DATA_JURISDICTION_ANY"
)

// Sovereign Edge PoPs
const (
	PoPDhaka     = "bd-dac"
	PoPSingapore = "sg-sin"
	PoPFrankfurt = "de-fra"
	PoPIceland   = "is-north-1"
	PoPVirginia  = "us-iad"
)

var (
	validPoPs = map[string]bool{
		PoPDhaka:     true,
		PoPSingapore: true,
		PoPFrankfurt: true,
		PoPIceland:   true,
		PoPVirginia:  true,
	}

	validJurisdictions = map[string]bool{
		JurisdictionBangladesh: true,
		JurisdictionEU:         true,
		JurisdictionSingapore:  true,
		JurisdictionUS:         true,
		JurisdictionAny:        true,
	}

	uuidRegex = regexp.MustCompile(`^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`)
	typeRegex = regexp.MustCompile(`^io\.aeitron\.[a-z0-9_\.]+\.v[0-9]+$`)
	pqcRegex  = regexp.MustCompile(`^[a-f0-9]{64}$`)

	ErrSovereigntyViolation = errors.New("sovereign boundary violation: Bangladesh NDMA 2026 strictly confines sovereign events to bd-dac")
	ErrInvalidEnvelope      = errors.New("invalid cloudevent envelope")
)

// Envelope represents a Aeitron CloudEvents v1.0 standard envelope.
type Envelope struct {
	SpecVersion             string          `json:"specversion"`
	ID                      string          `json:"id"`
	Source                  string          `json:"source"`
	Type                    string          `json:"type"`
	Time                    string          `json:"time"`
	DataContentType         string          `json:"datacontenttype"`
	DataSchema              string          `json:"dataschema,omitempty"`
	Subject                 string          `json:"subject,omitempty"`
	SovereigntyJurisdiction string          `json:"sovereignty_jurisdiction"`
	StrictSovereignty       bool            `json:"strict_sovereignty"`
	OriginPoP               string          `json:"origin_pop"`
	PQCAttestation          string          `json:"pqc_attestation,omitempty"`
	Data                    json.RawMessage `json:"data"`
}

// Domain Payloads

type DNSModifiedRecord struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

type DNSZoneUpdatedPayload struct {
	ZoneID          string              `json:"zone_id"`
	Name            string              `json:"name"`
	Serial          int64               `json:"serial"`
	DNSSECEnabled   bool                `json:"dnssec_enabled"`
	RecordsCount    int                 `json:"records_count"`
	ModifiedRecords []DNSModifiedRecord `json:"modified_records"`
	SignerPoP       string              `json:"signer_pop"`
	UpdatedAt       string              `json:"updated_at"`
}

type SubseaCableDegradedPayload struct {
	CableID              string  `json:"cable_id"`
	Name                 string  `json:"name"`
	OriginPoP            string  `json:"origin_pop"`
	TargetPoP            string  `json:"target_pop"`
	MeasuredRTTMs        float64 `json:"measured_rtt_ms"`
	NominalRTTMs         float64 `json:"nominal_rtt_ms"`
	JitterMs             float64 `json:"jitter_ms"`
	PacketLossPercent    float64 `json:"packet_loss_percent"`
	Status               string  `json:"status"`
	BGPRerouted          bool    `json:"bgp_rerouted"`
	ActiveAlternateRoute string  `json:"active_alternate_route,omitempty"`
	TimestampNs          int64   `json:"timestamp_ns"`
}

type BGPPeerChangedPayload struct {
	PeerIP                 string `json:"peer_ip"`
	PeerASN                uint32 `json:"peer_asn"`
	LocalPoP               string `json:"local_pop"`
	State                  string `json:"state"`
	PreviousState          string `json:"previous_state"`
	AnnouncedPrefixesCount int    `json:"announced_prefixes_count"`
	SessionUptimeSeconds   int64  `json:"session_uptime_seconds"`
	HoldTimeSeconds        int    `json:"hold_time_seconds,omitempty"`
	ErrorSubcode           string `json:"error_subcode,omitempty"`
	TimestampNs            int64  `json:"timestamp_ns"`
}

type ThreatDetectedPayload struct {
	ThreatID            string `json:"threat_id"`
	SourceIP            string `json:"source_ip"`
	CountryCode         string `json:"country_code"`
	DetectionPoP        string `json:"detection_pop"`
	TargetService       string `json:"target_service"`
	WAFRuleID           string `json:"waf_rule_id"`
	Action              string `json:"action"`
	Severity            string `json:"severity"`
	NDMAViolation       bool   `json:"ndma_violation"`
	MitigationLatencyNs int64  `json:"mitigation_latency_ns"`
	SignatureHash       string `json:"signature_hash,omitempty"`
	TimestampNs         int64  `json:"timestamp_ns"`
}

type KMSKeyRotatedPayload struct {
	KeyID               string   `json:"key_id"`
	Algorithm           string   `json:"algorithm"`
	PreviousVersion     int      `json:"previous_version"`
	NewVersion          int      `json:"new_version"`
	HSMSlotID           int      `json:"hsm_slot_id"`
	HSMPoP              string   `json:"hsm_pop"`
	PhysicalTokenSerial string   `json:"physical_token_serial"`
	PQCEnabled          bool     `json:"pqc_enabled"`
	DualCustodySigners  []string `json:"dual_custody_signers,omitempty"`
	EpochTimestampS     int64    `json:"epoch_timestamp_s"`
}

type ComputeWorkloadScheduledPayload struct {
	WorkloadID      string  `json:"workload_id"`
	TenantID        string  `json:"tenant_id"`
	WorkloadType    string  `json:"workload_type"`
	TargetPoP       string  `json:"target_pop"`
	TargetNodeID    string  `json:"target_node_id"`
	CPUMillicores   int     `json:"cpu_millicores"`
	MemoryMB        int     `json:"memory_mb"`
	GPUCount        int     `json:"gpu_count"`
	GPUSlicePercent int     `json:"gpu_slice_percent,omitempty"`
	LatencyBudgetMs float64 `json:"latency_budget_ms"`
	SovereigntyRule string  `json:"sovereignty_rule"`
	TimestampNs     int64   `json:"timestamp_ns"`
}

type NodeThermalThrottleTriggeredPayload struct {
	NodeID                  string  `json:"node_id"`
	PoPID                   string  `json:"pop_id"`
	ComponentType           string  `json:"component_type"`
	TemperatureCelsius      float64 `json:"temperature_celsius"`
	ThermalThresholdCelsius float64 `json:"thermal_threshold_celsius"`
	ThrottleLevelPercent    int     `json:"throttle_level_percent"`
	FanSpeedRPM             int     `json:"fan_speed_rpm"`
	CarbonIntensityGCO2kWh  float64 `json:"carbon_intensity_gco2_kwh"`
	ActionTaken             string  `json:"action_taken"`
	TimestampNs             int64   `json:"timestamp_ns"`
}

type StorageReplicationDivergencePayload struct {
	IncidentID         string `json:"incident_id"`
	DatabaseName       string `json:"database_name"`
	TableName          string `json:"table_name"`
	PartitionID        string `json:"partition_id"`
	RecordPrimaryKey   string `json:"record_primary_key"`
	SourcePoP          string `json:"source_pop"`
	TargetPoP          string `json:"target_pop"`
	SourceMerkleHash   string `json:"source_merkle_hash"`
	TargetMerkleHash   string `json:"target_merkle_hash"`
	ResolutionStrategy string `json:"resolution_strategy"`
	AutoHealed         bool   `json:"auto_healed,omitempty"`
	TimestampNs        int64  `json:"timestamp_ns"`
}

type TenantDRFailoverTriggeredPayload struct {
	TenantID           string  `json:"tenant_id"`
	PreviousRegion     string  `json:"previous_region"`
	NewActiveRegion    string  `json:"new_active_region"`
	TriggerSource      string  `json:"trigger_source"`
	Actor              string  `json:"actor"`
	MeasuredRTOMs      float64 `json:"measured_rto_ms"`
	MeasuredRPOMs      float64 `json:"measured_rpo_ms"`
	QuorumWitnessCount int     `json:"quorum_witness_count"`
	SovereigntyIntact  bool    `json:"sovereignty_intact"`
	DigitalSealHMAC    string  `json:"digital_seal_hmac"`
	TimestampNs        int64   `json:"timestamp_ns"`
}

type TenantDRRehearsalCompletedPayload struct {
	RehearsalID        string  `json:"rehearsal_id"`
	TenantID           string  `json:"tenant_id"`
	SimulatedPrimary   string  `json:"simulated_primary"`
	SimulatedSecondary string  `json:"simulated_secondary"`
	EstimatedRTOMs     float64 `json:"estimated_rto_ms"`
	EstimatedRPOMs     float64 `json:"estimated_rpo_ms"`
	CapacityVerified   bool    `json:"capacity_verified"`
	SovereigntyIntact  bool    `json:"sovereignty_intact"`
	Status             string  `json:"status"`
	DigitalSealHMAC    string  `json:"digital_seal_hmac"`
	TimestampNs        int64   `json:"timestamp_ns"`
}

// GenerateUUIDv4 creates an RFC 4122 compliant UUIDv4.
func GenerateUUIDv4() (string, error) {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", fmt.Errorf("failed to read cryptorand: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// ComputePQCAttestation derives a 32-byte HMAC-SHA256 post-quantum attestation digest.
func ComputePQCAttestation(secret []byte, data []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// NewEnvelope creates, signs, and formats a CloudEvents v1.0 envelope.
func NewEnvelope(
	eventType string,
	source string,
	originPoP string,
	jurisdiction string,
	strictSovereignty bool,
	payload any,
	pqcKey []byte,
) (*Envelope, error) {
	if !validPoPs[originPoP] {
		return nil, fmt.Errorf("%w: invalid origin PoP '%s'", ErrInvalidEnvelope, originPoP)
	}
	if !validJurisdictions[jurisdiction] {
		return nil, fmt.Errorf("%w: invalid jurisdiction '%s'", ErrInvalidEnvelope, jurisdiction)
	}
	if !typeRegex.MatchString(eventType) {
		return nil, fmt.Errorf("%w: event type '%s' does not match pattern", ErrInvalidEnvelope, eventType)
	}

	id, err := GenerateUUIDv4()
	if err != nil {
		return nil, err
	}

	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	if len(pqcKey) == 0 {
		pqcKey = []byte("aeitron-pqc-default-ml-kem-1024-ratchet-key")
	}
	pqcDigest := ComputePQCAttestation(pqcKey, dataBytes)

	env := &Envelope{
		SpecVersion:             CloudEventsSpecVersion,
		ID:                      id,
		Source:                  source,
		Type:                    eventType,
		Time:                    time.Now().UTC().Format(time.RFC3339),
		DataContentType:         "application/json",
		SovereigntyJurisdiction: jurisdiction,
		StrictSovereignty:       strictSovereignty,
		OriginPoP:               originPoP,
		PQCAttestation:          pqcDigest,
		Data:                    dataBytes,
	}

	if err := env.Validate(); err != nil {
		return nil, err
	}

	return env, nil
}

// Validate checks the envelope against strict CloudEvents and Aeitron invariants.
func (e *Envelope) Validate() error {
	if e.SpecVersion != CloudEventsSpecVersion {
		return fmt.Errorf("%w: specversion must be '1.0', got '%s'", ErrInvalidEnvelope, e.SpecVersion)
	}
	if !uuidRegex.MatchString(e.ID) {
		return fmt.Errorf("%w: id '%s' is not valid UUIDv4", ErrInvalidEnvelope, e.ID)
	}
	if e.Source == "" {
		return fmt.Errorf("%w: source cannot be empty", ErrInvalidEnvelope)
	}
	if !typeRegex.MatchString(e.Type) {
		return fmt.Errorf("%w: type '%s' does not match reverse-DNS pattern", ErrInvalidEnvelope, e.Type)
	}
	if _, err := time.Parse(time.RFC3339, e.Time); err != nil {
		return fmt.Errorf("%w: time '%s' is not RFC 3339", ErrInvalidEnvelope, e.Time)
	}
	if e.DataContentType != "application/json" && e.DataContentType != "application/octet-stream" {
		return fmt.Errorf("%w: invalid datacontenttype '%s'", ErrInvalidEnvelope, e.DataContentType)
	}
	if !validJurisdictions[e.SovereigntyJurisdiction] {
		return fmt.Errorf("%w: invalid jurisdiction '%s'", ErrInvalidEnvelope, e.SovereigntyJurisdiction)
	}
	if !validPoPs[e.OriginPoP] {
		return fmt.Errorf("%w: invalid origin_pop '%s'", ErrInvalidEnvelope, e.OriginPoP)
	}
	if e.PQCAttestation != "" && !pqcRegex.MatchString(e.PQCAttestation) {
		return fmt.Errorf("%w: pqc_attestation '%s' must be 64-hex SHA-256", ErrInvalidEnvelope, e.PQCAttestation)
	}
	if len(e.Data) == 0 {
		return fmt.Errorf("%w: data cannot be empty", ErrInvalidEnvelope)
	}
	return nil
}

// EnforceSovereignEgress enforces Bangladesh NDMA 2026 data residency constraints.
// If strict sovereignty is active or the event is under Bangladesh jurisdiction,
// egress outside bd-dac is blocked fail-closed.
func (e *Envelope) EnforceSovereignEgress(destinationPoP string) error {
	isSovereign := e.StrictSovereignty || e.SovereigntyJurisdiction == JurisdictionBangladesh
	if isSovereign {
		if destinationPoP != PoPDhaka {
			return fmt.Errorf(
				"%w: event id '%s' (type: '%s') cannot egress to '%s'",
				ErrSovereigntyViolation,
				e.ID,
				e.Type,
				destinationPoP,
			)
		}
	}
	return nil
}
