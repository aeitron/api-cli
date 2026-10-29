package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewEnvelopeSuccess(t *testing.T) {
	payload := ThreatDetectedPayload{
		ThreatID:            "a1b2c3d4-e5f6-4000-8000-123456789abc",
		SourceIP:            "198.51.100.44",
		CountryCode:         "SG",
		DetectionPoP:        PoPDhaka,
		TargetService:       "srv-edge-gateway",
		WAFRuleID:           "CRS-942100-SQLI",
		Action:              "BLOCK",
		Severity:            "CRITICAL",
		NDMAViolation:       true,
		MitigationLatencyNs: 4200,
		TimestampNs:         1792656000000000000,
	}

	env, err := NewEnvelope(
		"io.aeitron.security.threat_detected.v1",
		"urn:aeitron:pop:bd-dac:service:security",
		PoPDhaka,
		JurisdictionBangladesh,
		true,
		payload,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected NewEnvelope error: %v", err)
	}

	if env.SpecVersion != "1.0" {
		t.Errorf("expected SpecVersion 1.0, got %s", env.SpecVersion)
	}
	if len(env.ID) != 36 {
		t.Errorf("expected UUIDv4 ID length 36, got %d (%s)", len(env.ID), env.ID)
	}
	if len(env.PQCAttestation) != 64 {
		t.Errorf("expected 64-char hex PQC attestation, got %d (%s)", len(env.PQCAttestation), env.PQCAttestation)
	}
	if !env.StrictSovereignty {
		t.Errorf("expected StrictSovereignty to be true")
	}

	// Verify unmarshaling payload from raw JSON
	var decoded ThreatDetectedPayload
	if err := json.Unmarshal(env.Data, &decoded); err != nil {
		t.Fatalf("failed to decode event data: %v", err)
	}
	if decoded.ThreatID != payload.ThreatID {
		t.Errorf("expected ThreatID %s, got %s", payload.ThreatID, decoded.ThreatID)
	}
}

func TestNewEnvelopeInvalidPoP(t *testing.T) {
	_, err := NewEnvelope(
		"io.aeitron.dns.zone_updated.v1",
		"urn:aeitron:pop:invalid:service:dns",
		"invalid-pop",
		JurisdictionAny,
		false,
		map[string]string{"foo": "bar"},
		nil,
	)
	if err == nil {
		t.Fatalf("expected error for invalid PoP, got nil")
	}
}

func TestNewEnvelopeInvalidType(t *testing.T) {
	_, err := NewEnvelope(
		"invalid_event_type",
		"urn:aeitron:pop:bd-dac:service:dns",
		PoPDhaka,
		JurisdictionAny,
		false,
		map[string]string{"foo": "bar"},
		nil,
	)
	if err == nil {
		t.Fatalf("expected error for invalid event type pattern, got nil")
	}
}

func TestSovereignEgressBangladeshStrict_BlocksForeignPoP(t *testing.T) {
	env, err := NewEnvelope(
		"io.aeitron.security.threat_detected.v1",
		"urn:aeitron:pop:bd-dac:service:security",
		PoPDhaka,
		JurisdictionBangladesh,
		true,
		map[string]string{"confidential": "citizen-data"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected NewEnvelope error: %v", err)
	}

	foreignPoPs := []string{PoPSingapore, PoPFrankfurt, PoPIceland, PoPVirginia}
	for _, fPop := range foreignPoPs {
		err := env.EnforceSovereignEgress(fPop)
		if err == nil {
			t.Errorf("expected sovereign egress violation when routing to %s, got nil", fPop)
		}
		if !strings.Contains(err.Error(), "NDMA 2026 strictly confines sovereign events to bd-dac") {
			t.Errorf("expected NDMA 2026 confinement error, got: %v", err)
		}
	}
}

func TestSovereignEgressBangladeshStrict_AllowsLocalDhaka(t *testing.T) {
	env, err := NewEnvelope(
		"io.aeitron.dns.zone_updated.v1",
		"urn:aeitron:pop:bd-dac:service:dns",
		PoPDhaka,
		JurisdictionBangladesh,
		true,
		map[string]string{"zone": "internal.gov.bd"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected NewEnvelope error: %v", err)
	}

	if err := env.EnforceSovereignEgress(PoPDhaka); err != nil {
		t.Errorf("expected local bd-dac egress to be permitted, got error: %v", err)
	}
}

func TestSovereignEgressGlobalAnycast_AllowsForeignPoP(t *testing.T) {
	env, err := NewEnvelope(
		"io.aeitron.network.cable_degraded.v1",
		"urn:aeitron:pop:sg-sin:service:network",
		PoPSingapore,
		JurisdictionAny,
		false,
		map[string]string{"cable": "SMW6"},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected NewEnvelope error: %v", err)
	}

	if err := env.EnforceSovereignEgress(PoPFrankfurt); err != nil {
		t.Errorf("expected global non-sovereign event to egress freely, got error: %v", err)
	}
}

func TestDomainPayloadSerialization(t *testing.T) {
	cablePayload := SubseaCableDegradedPayload{
		CableID:           "SMW6",
		Name:              "South East Asia-Middle East-Western Europe 6",
		OriginPoP:         PoPDhaka,
		TargetPoP:         PoPSingapore,
		MeasuredRTTMs:     184.2,
		NominalRTTMs:      42.0,
		JitterMs:          38.5,
		PacketLossPercent: 12.8,
		Status:            "DEGRADED",
		BGPRerouted:       true,
		TimestampNs:       1792656000000000000,
	}

	env, err := NewEnvelope(
		"io.aeitron.network.cable_degraded.v1",
		"urn:aeitron:pop:bd-dac:service:network",
		PoPDhaka,
		JurisdictionAny,
		false,
		cablePayload,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create cable degraded envelope: %v", err)
	}

	var roundtrip SubseaCableDegradedPayload
	if err := json.Unmarshal(env.Data, &roundtrip); err != nil {
		t.Fatalf("failed to unmarshal cable payload: %v", err)
	}
	if roundtrip.CableID != "SMW6" {
		t.Errorf("expected CableID SMW6, got %s", roundtrip.CableID)
	}
	if roundtrip.MeasuredRTTMs != 184.2 {
		t.Errorf("expected MeasuredRTTMs 184.2, got %f", roundtrip.MeasuredRTTMs)
	}
}

func TestTenantDRPayloadSerialization(t *testing.T) {
	failoverPayload := TenantDRFailoverTriggeredPayload{
		TenantID:           "tenant-dhaka-bank-01",
		PreviousRegion:     "dhaka",
		NewActiveRegion:    "singapore",
		TriggerSource:      "DUAL_WITNESS_CONSENSUS",
		Actor:              "dr-automated-controller",
		MeasuredRTOMs:      1850.5,
		MeasuredRPOMs:      45.0,
		QuorumWitnessCount: 2,
		SovereigntyIntact:  true,
		DigitalSealHMAC:    "9f8a4b12c7d8e9f0112233445566778899aabbccddeeff001122334455667788",
		TimestampNs:        time.Now().UnixNano(),
	}

	env, err := NewEnvelope(
		"io.aeitron.tenant.dr.failover.v1",
		"urn:aeitron:pop:bd-dac:service:tenant-dr",
		PoPDhaka,
		JurisdictionBangladesh,
		true,
		failoverPayload,
		[]byte("attestation-secret-key-32bytes!!"),
	)
	if err != nil {
		t.Fatalf("failed to create DR failover envelope: %v", err)
	}

	var roundtrip TenantDRFailoverTriggeredPayload
	if err := json.Unmarshal(env.Data, &roundtrip); err != nil {
		t.Fatalf("failed to unmarshal DR failover payload: %v", err)
	}
	if roundtrip.TenantID != "tenant-dhaka-bank-01" {
		t.Errorf("expected TenantID tenant-dhaka-bank-01, got %s", roundtrip.TenantID)
	}
	if roundtrip.MeasuredRTOMs != 1850.5 {
		t.Errorf("expected MeasuredRTOMs 1850.5, got %f", roundtrip.MeasuredRTOMs)
	}
	if env.PQCAttestation == "" {
		t.Errorf("expected non-empty PQC attestation")
	}

	rehearsalPayload := TenantDRRehearsalCompletedPayload{
		RehearsalID:        "reh-001",
		TenantID:           "tenant-dhaka-bank-01",
		SimulatedPrimary:   "dhaka",
		SimulatedSecondary: "singapore",
		EstimatedRTOMs:     1420.0,
		EstimatedRPOMs:     38.0,
		CapacityVerified:   true,
		SovereigntyIntact:  true,
		Status:             "SUCCESS",
		DigitalSealHMAC:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		TimestampNs:        time.Now().UnixNano(),
	}

	rehEnv, err := NewEnvelope(
		"io.aeitron.tenant.dr.rehearsal.v1",
		"urn:aeitron:pop:bd-dac:service:tenant-dr",
		PoPDhaka,
		JurisdictionBangladesh,
		true,
		rehearsalPayload,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create DR rehearsal envelope: %v", err)
	}
	var rehRoundtrip TenantDRRehearsalCompletedPayload
	if err := json.Unmarshal(rehEnv.Data, &rehRoundtrip); err != nil {
		t.Fatalf("failed to unmarshal DR rehearsal payload: %v", err)
	}
	if rehRoundtrip.RehearsalID != "reh-001" {
		t.Errorf("expected RehearsalID reh-001, got %s", rehRoundtrip.RehearsalID)
	}
}
