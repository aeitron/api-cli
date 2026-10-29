package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSecurityCLICommands_EndToEnd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v1/security/sovereignty/evaluate" && r.Method == http.MethodPost:
			var req SovereigntyEvaluationRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			res := SovereigntyEvaluationResult{
				Verdict:     "RESIDENCY_COMPLIANT_LOCAL_PROCESSING",
				Compliant:   true,
				Standard:    "BANGLADESH_NDMA_2026",
				Reason:      "Workload processed locally within Bangladesh territory",
				EvaluatedAt: time.Now(),
			}
			_ = json.NewEncoder(w).Encode(res)

		case r.URL.Path == "/v1/security/firewall/block" && r.Method == http.MethodPost:
			entry := BlockedIPEntry{
				IPAddress:      "198.51.100.42",
				Reason:         "test-block",
				DroppedPackets: 0,
				BlockedAt:      time.Now(),
				ExpiresAt:      time.Now().Add(1 * time.Hour),
				FilterMode:     "XDP_DROP",
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(entry)

		case r.URL.Path == "/v1/security/firewall/blocked" && r.Method == http.MethodGet:
			ips := []BlockedIPEntry{
				{
					IPAddress:      "198.51.100.42",
					Reason:         "test-block",
					DroppedPackets: 154,
					BlockedAt:      time.Now(),
					ExpiresAt:      time.Now().Add(1 * time.Hour),
					FilterMode:     "XDP_DROP",
				},
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"count": 1, "ips": ips})

		case r.URL.Path == "/v1/security/firewall/unblock" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "UNBLOCKED", "ip_address": "198.51.100.42"})

		case r.URL.Path == "/v1/security/policies/domain-alpha" && r.Method == http.MethodGet:
			p := SecurityPolicy{
				PolicyID:             "pol-123",
				DomainID:             "domain-alpha",
				WAFEnabled:           true,
				WAFMode:              "BLOCK",
				OWASPRulesetEnabled:  true,
				RateLimitEnabled:     true,
				RateLimitRPM:         12000,
				BurstCapacity:        500,
				BotProtectionEnabled: true,
				UpdatedAt:            time.Now(),
			}
			_ = json.NewEncoder(w).Encode(p)

		case r.URL.Path == "/v1/security/policies/domain-alpha" && r.Method == http.MethodPut:
			var p SecurityPolicy
			_ = json.NewDecoder(r.Body).Decode(&p)
			p.PolicyID = "pol-saved"
			p.UpdatedAt = time.Now()
			_ = json.NewEncoder(w).Encode(p)

		case r.URL.Path == "/v1/security/audit/chain" && r.Method == http.MethodGet:
			entries := []AuditEntry{
				{
					Index:       0,
					Timestamp:   time.Now(),
					PrevHash:    "0000000000000000000000000000000000000000000000000000000000000000",
					CurrentHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					Action:      "SOVEREIGNTY_EVALUATE",
					Subject:     "app-core",
					Actor:       "tenant-gov",
					Verdict:     "PERMITTED",
					Details:     "Domestic execution in Dhaka",
				},
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"verified": true,
				"count":    1,
				"entries":  entries,
			})

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := &Client{
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	}

	// 1. Audit sovereignty (passing)
	if err := HandleSecurityCommand(client, []string{
		"audit",
		"--workload", "app-core",
		"--source", "dhaka",
		"--target", "dhaka",
		"--classification", "CII_CRITICAL",
	}); err != nil {
		t.Fatalf("security audit failed: %v", err)
	}

	// 2. Audit Log (Merkle Chain)
	if err := HandleSecurityCommand(client, []string{"audit-log"}); err != nil {
		t.Fatalf("security audit-log failed: %v", err)
	}

	// 3. Firewall Block
	if err := HandleSecurityCommand(client, []string{
		"firewall", "block", "198.51.100.42",
		"--reason", "ddos-mitigation",
		"--duration", "1800",
	}); err != nil {
		t.Fatalf("firewall block failed: %v", err)
	}

	// 4. Firewall List
	if err := HandleSecurityCommand(client, []string{"firewall", "list"}); err != nil {
		t.Fatalf("firewall list failed: %v", err)
	}

	// 5. Firewall Unblock
	if err := HandleSecurityCommand(client, []string{"firewall", "unblock", "198.51.100.42"}); err != nil {
		t.Fatalf("firewall unblock failed: %v", err)
	}

	// 6. Policy Get
	if err := HandleSecurityCommand(client, []string{"policy", "get", "domain-alpha"}); err != nil {
		t.Fatalf("policy get failed: %v", err)
	}

	// 7. Policy Set
	if err := HandleSecurityCommand(client, []string{
		"policy", "set", "domain-alpha",
		"--waf",
		"--waf-mode", "BLOCK",
		"--owasp",
		"--rate-limit",
		"--rpm", "15000",
		"--burst", "600",
		"--bot",
	}); err != nil {
		t.Fatalf("policy set failed: %v", err)
	}
}
