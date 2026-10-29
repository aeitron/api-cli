package dns

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDNSCLICommands_EndToEnd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/v1/zones" && r.Method == http.MethodPost:
			zone := DNSZone{
				ID:        "zone-test-123",
				TenantID:  "tenant-corp",
				Name:      "testdomain.com.",
				Serial:    123456,
				Status:    "ACTIVE",
				DNSSEC:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(zone)

		case r.URL.Path == "/v1/zones" && r.Method == http.MethodGet:
			zones := []DNSZone{
				{
					ID:       "zone-test-123",
					TenantID: "tenant-corp",
					Name:     "testdomain.com.",
					Serial:   123456,
					Status:   "ACTIVE",
					DNSSEC:   true,
				},
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"zones": zones})

		case r.URL.Path == "/v1/zones/zone-test-123/records" && r.Method == http.MethodPost:
			var rec DNSRecord
			_ = json.NewDecoder(r.Body).Decode(&rec)
			rec.ID = "rec-test-001"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(rec)

		case r.URL.Path == "/v1/zones/zone-test-123/records" && r.Method == http.MethodGet:
			records := []DNSRecord{
				{
					ID:            "rec-test-001",
					ZoneID:        "zone-test-123",
					Name:          "api.testdomain.com.",
					Type:          TypeA,
					TTL:           300,
					Content:       "103.144.180.1",
					RoutingPolicy: PolicyAnycastGeo,
					GeoTargets:    map[string]string{"dhaka": "103.144.180.1", "singapore": "80.225.241.129"},
				},
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"records": records})

		case r.URL.Path == "/v1/dns/resolve" && r.Method == http.MethodPost:
			resp := ResolveResponse{
				QueryName:  "api.testdomain.com.",
				RecordType: TypeA,
				Records: []ResolvedRecord{
					{Name: "api.testdomain.com.", Type: TypeA, TTL: 300, Content: "103.144.180.1"},
				},
				MatchedPolicy:  PolicyAnycastGeo,
				SelectedRegion: "dhaka",
				LatencyMs:      4.2,
			}
			_ = json.NewEncoder(w).Encode(resp)

		case r.URL.Path == "/v1/zones/zone-test-123/export" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("$ORIGIN testdomain.com.\n@ IN SOA ns1.aeitron.net. hostmaster 1 10800 3600 604800 300\n"))

		case r.URL.Path == "/v1/dns/request-deletion" && r.Method == http.MethodPost:
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)
			delReq := DnsDeletionRequest{
				RequestID:      "del_req_mock_456",
				Domain:         payload["domain"],
				RecordID:       payload["record_id"],
				Requester:      "test_operator",
				Status:         "PENDING_APPROVAL",
				ExpiresAt:      time.Now().Add(10 * time.Minute),
				ChallengeToken: "tok_approval_789",
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(delReq)

		case r.URL.Path == "/v1/dns/approve-deletion" && r.Method == http.MethodPost:
			var payload map[string]string
			_ = json.NewDecoder(r.Body).Decode(&payload)
			approval := DnsApprovalResult{
				Status:        "APPROVED",
				RequestID:     payload["requestId"],
				ApprovedBy:    payload["actor"],
				ApprovalToken: "tok_approval_789",
				ValidUntil:    time.Now().Add(10 * time.Minute),
				Message:       "Operator authorization granted",
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(approval)

		case r.URL.Path == "/v1/zones/zone-test-123/records/rec-protected-001" && r.Method == http.MethodDelete:
			q := r.URL.Query()
			if q.Get("confirm") != "true" || q.Get("approval_token") != "tok_approval_789" {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error":     "APPROVAL_REQUIRED: Record deletion requires challenge approval",
					"requestId": "del_req_mock_456",
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "DELETED"})

		case r.URL.Path == "/v1/zones/zone-test-123/records/rec-test-001" && r.Method == http.MethodDelete:
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "DELETED"})

		case r.URL.Path == "/v1/zones/zone-test-123" && r.Method == http.MethodDelete:
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "DELETED"})

		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := &Client{
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	}

	// 1. Create Zone
	if err := HandleDNSCommand(client, []string{"zones", "create", "testdomain.com", "--tenant", "tenant-corp"}); err != nil {
		t.Fatalf("zones create failed: %v", err)
	}

	// 2. List Zones
	if err := HandleDNSCommand(client, []string{"zones", "list"}); err != nil {
		t.Fatalf("zones list failed: %v", err)
	}

	// 3. Add Record
	if err := HandleDNSCommand(client, []string{
		"records", "add", "zone-test-123",
		"--name", "api.testdomain.com",
		"--type", "A",
		"--content", "103.144.180.1",
		"--policy", "ANYCAST_GEO",
		"--geo-targets", "dhaka=103.144.180.1,singapore=80.225.241.129",
	}); err != nil {
		t.Fatalf("records add failed: %v", err)
	}

	// 4. List Records
	if err := HandleDNSCommand(client, []string{"records", "list", "zone-test-123"}); err != nil {
		t.Fatalf("records list failed: %v", err)
	}

	// 5. Resolve
	if err := HandleDNSCommand(client, []string{"resolve", "api.testdomain.com", "--region", "dhaka"}); err != nil {
		t.Fatalf("resolve failed: %v", err)
	}

	// 6. Export BIND
	if err := HandleDNSCommand(client, []string{"export", "zone-test-123"}); err != nil {
		t.Fatalf("export failed: %v", err)
	}

	// 7. Request deletion of protected record (initiates challenge)
	if err := HandleDNSCommand(client, []string{
		"records", "request-delete", "testdomain.com", "rec-protected-001",
		"--reason", "Decommissioning legacy CNAME",
	}); err != nil {
		t.Fatalf("records request-delete failed: %v", err)
	}

	// 8. Attempt deletion without token / confirm (must fail with challenge guidance)
	if err := HandleDNSCommand(client, []string{"records", "delete", "zone-test-123", "rec-protected-001"}); err == nil {
		t.Fatalf("records delete on protected record without approval expected challenge error, got nil")
	}

	// 9. Independent operator approves deletion
	if err := HandleDNSCommand(client, []string{"approve", "del_req_mock_456", "--actor", "operator_security_lead"}); err != nil {
		t.Fatalf("approve failed: %v", err)
	}

	// 10. Execute deletion with approval token and confirm flag
	if err := HandleDNSCommand(client, []string{
		"records", "delete", "zone-test-123", "rec-protected-001",
		"--confirm", "--approval-token", "tok_approval_789",
	}); err != nil {
		t.Fatalf("records delete with approval token failed: %v", err)
	}

	// 11. Delete Unprotected Record
	if err := HandleDNSCommand(client, []string{"records", "delete", "zone-test-123", "rec-test-001"}); err != nil {
		t.Fatalf("records delete failed: %v", err)
	}

	// 12. Delete Zone
	if err := HandleDNSCommand(client, []string{"zones", "delete", "zone-test-123"}); err != nil {
		t.Fatalf("zones delete failed: %v", err)
	}
}

func TestDNSCLICommands_FailClosed(t *testing.T) {
	// Client configured in fail-closed mode pointing to unreachable endpoint
	client := &Client{
		BaseURL:    "http://127.0.0.1:59999",
		HealthURL:  "http://127.0.0.1:59999",
		FailClosed: true,
		HTTPClient: &http.Client{Timeout: 500 * time.Millisecond},
	}

	// Health status must fail closed
	if err := HandleDNSCommand(client, []string{"health"}); err == nil {
		t.Fatalf("health in fail-closed mode expected error on unreachable backend, got nil")
	}

	// Drain PoP must fail closed
	if err := HandleDNSCommand(client, []string{"drain", "--reason", "emergency"}); err == nil {
		t.Fatalf("drain in fail-closed mode expected error on unreachable backend, got nil")
	}

	// Restore PoP must fail closed
	if err := HandleDNSCommand(client, []string{"restore"}); err == nil {
		t.Fatalf("restore in fail-closed mode expected error on unreachable backend, got nil")
	}

	// Failover events must fail closed
	if err := HandleDNSCommand(client, []string{"failover"}); err == nil {
		t.Fatalf("failover in fail-closed mode expected error on unreachable backend, got nil")
	}
}

func TestDNSCLICommands_HealthAndFailover(t *testing.T) {
	client := NewClient()

	// 1. Help commands
	if err := HandleDNSCommand(client, []string{}); err != nil {
		t.Fatalf("HandleDNSCommand() empty failed: %v", err)
	}
	if err := HandleDNSCommand(client, []string{"help"}); err != nil {
		t.Fatalf("HandleDNSCommand(help) failed: %v", err)
	}

	// 2. Health HUD
	if err := HandleDNSCommand(client, []string{"health"}); err != nil {
		t.Fatalf("HandleDNSCommand(health) failed: %v", err)
	}

	// 3. Drain PoP
	if err := HandleDNSCommand(client, []string{"drain", "--reason", "scheduled fiber maintenance"}); err != nil {
		t.Fatalf("HandleDNSCommand(drain) failed: %v", err)
	}

	// 4. Restore PoP
	if err := HandleDNSCommand(client, []string{"restore"}); err != nil {
		t.Fatalf("HandleDNSCommand(restore) failed: %v", err)
	}

	// 5. Failover events
	if err := HandleDNSCommand(client, []string{"failover"}); err != nil {
		t.Fatalf("HandleDNSCommand(failover) failed: %v", err)
	}

	// 6. Unknown command returns error
	if err := HandleDNSCommand(client, []string{"invalid_subcommand"}); err == nil {
		t.Fatalf("HandleDNSCommand(invalid_subcommand) expected error, got nil")
	}
}
