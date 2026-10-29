package network

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupMockServer() (*Client, *httptest.Server) {
	mux := http.NewServeMux()

	// 1. Prefix Validate
	mux.HandleFunc("/api/v1/network/bgp/validate-prefix", func(w http.ResponseWriter, r *http.Request) {
		var req ValidatePrefixRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(req.Prefix, "10.") {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "REJECTED",
				"prefix": req.Prefix,
				"error":  "bogon prefix rejected",
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "VALID",
			"prefix": req.Prefix,
		})
	})

	// 2. RPKI Validate
	mux.HandleFunc("/api/v1/network/bgp/validate-rpki", func(w http.ResponseWriter, r *http.Request) {
		var req ValidateRPKIRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		status := "VALID"
		if req.ASN != 202456 {
			status = "INVALID"
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": status,
			"prefix": req.Prefix,
			"asn":    req.ASN,
		})
	})

	// 3. Flap Record
	mux.HandleFunc("/api/v1/network/bgp/record-flap", func(w http.ResponseWriter, r *http.Request) {
		var req FlapRecordRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"prefix":  req.Prefix,
			"state":   "ACTIVE",
			"penalty": 1000.0,
		})
	})

	// 4. Damping Status
	mux.HandleFunc("/api/v1/network/bgp/damping-status", func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("prefix")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"prefix":  prefix,
			"state":   "ACTIVE",
			"penalty": 450.0,
		})
	})

	// 5. Drain Plan
	mux.HandleFunc("/api/v1/network/bgp/drain-plan", func(w http.ResponseWriter, r *http.Request) {
		var req DrainPlanRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DrainPlanResponse{
			PeerID:            req.PeerID,
			CommunityTag:      "65535:0",
			LocalPreference:   0,
			MED:               65535,
			DwellDuration:     time.Duration(req.DwellSeconds) * time.Second,
			TargetAlternative: req.TargetAlternative,
		})
	})

	// 6. Flow Dispatch
	mux.HandleFunc("/api/v1/network/maglev/dispatch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"backend_id":   "pop-sin-gw-01",
			"backend_ip":   "172.30.0.3",
			"backend_port": 8080,
			"weight":       100,
		})
	})

	// 7. Maglev Stats
	mux.HandleFunc("/api/v1/network/maglev/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"table_size":        65537,
			"backend_count":     5,
			"conntrack_entries": 12,
		})
	})

	// 8. PoP Health
	mux.HandleFunc("/v1/edge/pops/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_pops": 4,
			"pops": []map[string]interface{}{
				{"pop_id": "dhaka", "region": "asia-south1", "target_endpoint": "198.51.100.1:443", "status": "POP_ACTIVE", "healthy": true, "last_rtt_ms": 4.2, "ewma_latency_ms": 4.2},
				{"pop_id": "singapore", "region": "asia-southeast1", "target_endpoint": "80.225.241.129:443", "status": "POP_ACTIVE", "healthy": true, "last_rtt_ms": 8.1, "ewma_latency_ms": 8.1},
				{"pop_id": "frankfurt", "region": "europe-west3", "target_endpoint": "198.51.100.21:443", "status": "POP_ACTIVE", "healthy": true, "last_rtt_ms": 14.3, "ewma_latency_ms": 14.3},
				{"pop_id": "virginia", "region": "us-east1", "target_endpoint": "198.51.100.31:443", "status": "POP_ACTIVE", "healthy": true, "last_rtt_ms": 12.0, "ewma_latency_ms": 12.0},
			},
		})
	})

	// 9. PoP Routes
	mux.HandleFunc("/v1/edge/pops/routes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_routes": 4,
			"routes": []map[string]interface{}{
				{
					"pop_id":         "dhaka",
					"region":         "asia-south1",
					"primary_target": "bdix-dhaka-direct",
					"primary_rtt_ms": 4.0,
					"active_target":  "bdix-dhaka-direct",
					"status":         "OPTIMAL",
					"failover_targets": []map[string]interface{}{
						{"target_pop": "singapore", "subsea_cable": "SMW6", "sla_rtt_ms": 32.0, "sovereignty_restricted": true, "status": "STANDBY"},
					},
				},
			},
		})
	})

	// 10. PoP Probe Run
	mux.HandleFunc("/v1/edge/pops/health/probe", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total_probed": 4,
			"results": map[string]interface{}{
				"dhaka":     map[string]interface{}{"pop_id": "dhaka", "target": "198.51.100.1:443", "success": true, "latency_ms": 4.2, "status_code": 200},
				"singapore": map[string]interface{}{"pop_id": "singapore", "target": "80.225.241.129:443", "success": true, "latency_ms": 8.1, "status_code": 200},
			},
		})
	})

	// 11. BGP Peers
	mux.HandleFunc("/v1/bgp/peers", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": 1,
			"peers": []map[string]interface{}{
				{
					"config": map[string]interface{}{
						"peer_asn":   24115,
						"peer_addr":  "103.108.140.1",
						"hold_time":  90000000000,
						"enable_bfd": true,
						"region":     "bd-dac",
					},
					"state":         "ESTABLISHED",
					"prefixes_held": 1,
					"updates_sent":  14,
				},
			},
		})
	})

	// 12. BGP Advertise
	mux.HandleFunc("/v1/bgp/advertise", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		prefix, _ := req["prefix"].(string)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ADVERTISED",
			"prefix":    prefix,
			"local_asn": 64512,
			"router_id": "198.51.100.1",
		})
	})

	// 13. BGP Withdraw
	mux.HandleFunc("/v1/bgp/withdraw", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		prefix, _ := req["prefix"].(string)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "WITHDRAWN",
			"prefix":  prefix,
			"message": "Prefix withdrawn from Anycast mesh",
		})
	})

	// 14. XDP Stats
	mux.HandleFunc("/v1/edge/xdp/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(XDPStatsDTO{
			Interface:      "anycast-pop-bd",
			Mode:           "ZeroCopy (XSK / AF_XDP)",
			LineRateMpps:   14.88,
			TotalRxPackets: 148800000,
			TotalTxPackets: 147500000,
			DDoSDrops:      1250000,
			PassedPackets:  147500000,
		})
	})

	server := httptest.NewServer(mux)
	client := &Client{
		RouterURL:           server.URL,
		ConfigControllerURL: server.URL,
		BGPControllerURL:    server.URL,
		AuthToken:           "test-token",
		HTTPClient:          server.Client(),
	}
	return client, server
}

func TestNetworkCommands_EndToEnd(t *testing.T) {
	client, server := setupMockServer()
	defer server.Close()

	// 1. Help
	if err := HandleNetworkCommand(client, []string{}); err != nil {
		t.Fatalf("expected nil for empty args help, got: %v", err)
	}
	if err := HandleNetworkCommand(client, []string{"help"}); err != nil {
		t.Fatalf("expected nil for help, got: %v", err)
	}

	// 2. BGP Validate - Valid
	if err := HandleNetworkCommand(client, []string{"bgp", "validate", "103.144.180.0/24"}); err != nil {
		t.Fatalf("expected nil for valid prefix, got: %v", err)
	}

	// 3. BGP Validate - Bogon Rejected (should print rejected without crashing)
	if err := HandleNetworkCommand(client, []string{"bgp", "validate", "10.0.0.0/8"}); err != nil {
		t.Fatalf("expected nil for handled bogon rejection, got: %v", err)
	}

	// 4. BGP RPKI
	if err := HandleNetworkCommand(client, []string{"bgp", "rpki", "202456", "103.144.180.0/24"}); err != nil {
		t.Fatalf("expected nil for rpki command, got: %v", err)
	}

	// 5. BGP Flap
	if err := HandleNetworkCommand(client, []string{"bgp", "flap", "198.51.100.0/24"}); err != nil {
		t.Fatalf("expected nil for flap command, got: %v", err)
	}

	// 6. BGP Damping Status
	if err := HandleNetworkCommand(client, []string{"bgp", "damping-status", "198.51.100.0/24"}); err != nil {
		t.Fatalf("expected nil for damping-status command, got: %v", err)
	}

	// 7. BGP Drain
	if err := HandleNetworkCommand(client, []string{"bgp", "drain", "peer-gw-01", "--alternative", "pop-sin-gw-02", "--dwell", "45"}); err != nil {
		t.Fatalf("expected nil for drain command, got: %v", err)
	}

	// 8. Flow Dispatch
	if err := HandleNetworkCommand(client, []string{"flow", "dispatch", "203.0.113.10", "54321", "103.144.180.1", "443", "6"}); err != nil {
		t.Fatalf("expected nil for flow dispatch command, got: %v", err)
	}

	// 9. Maglev Stats
	if err := HandleNetworkCommand(client, []string{"maglev", "stats"}); err != nil {
		t.Fatalf("expected nil for maglev stats command, got: %v", err)
	}

	// 10. BGP Peers
	if err := HandleNetworkCommand(client, []string{"bgp", "peers"}); err != nil {
		t.Fatalf("expected nil for bgp peers via network, got: %v", err)
	}
	if err := HandleBGPCommand(client, []string{"peers"}); err != nil {
		t.Fatalf("expected nil for bgp peers direct, got: %v", err)
	}

	// 11. BGP Advertise
	if err := HandleBGPCommand(client, []string{"advertise", "103.108.140.0/24", "--community", "64512:1001", "--med", "50"}); err != nil {
		t.Fatalf("expected nil for bgp advertise, got: %v", err)
	}

	// 12. BGP Withdraw
	if err := HandleBGPCommand(client, []string{"withdraw", "103.108.140.0/24"}); err != nil {
		t.Fatalf("expected nil for bgp withdraw, got: %v", err)
	}

	// 13. XDP Stats
	if err := HandleNetworkCommand(client, []string{"xdp", "stats"}); err != nil {
		t.Fatalf("expected nil for xdp stats via network, got: %v", err)
	}
	if err := HandleXDPCommand(client, []string{"stats"}); err != nil {
		t.Fatalf("expected nil for xdp stats direct, got: %v", err)
	}

	// 13b. XDP WireGuard Offload Stats
	if err := HandleNetworkCommand(client, []string{"xdp", "wireguard"}); err != nil {
		t.Fatalf("expected nil for xdp wireguard via network, got: %v", err)
	}
	if err := HandleXDPCommand(client, []string{"wireguard"}); err != nil {
		t.Fatalf("expected nil for xdp wireguard direct, got: %v", err)
	}

	// 14. BGP and XDP help
	if err := HandleBGPCommand(client, []string{}); err != nil {
		t.Fatalf("expected nil for bgp empty help, got: %v", err)
	}
	if err := HandleXDPCommand(client, []string{}); err != nil {
		t.Fatalf("expected nil for xdp empty help, got: %v", err)
	}
}

func TestNetworkCommands_Errors(t *testing.T) {
	client, server := setupMockServer()
	defer server.Close()

	// Unknown subcommand
	if err := HandleNetworkCommand(client, []string{"unknown"}); err == nil {
		t.Fatal("expected error for unknown subcommand, got nil")
	}

	// Unknown BGP action
	if err := HandleNetworkCommand(client, []string{"bgp", "unknown"}); err == nil {
		t.Fatal("expected error for unknown bgp action, got nil")
	}

	// Missing prefix in validate
	if err := HandleNetworkCommand(client, []string{"bgp", "validate"}); err == nil {
		t.Fatal("expected error for missing prefix in validate, got nil")
	}

	// Missing prefix in advertise
	if err := HandleBGPCommand(client, []string{"advertise"}); err == nil {
		t.Fatal("expected error for missing prefix in advertise, got nil")
	}

	// Missing prefix in withdraw
	if err := HandleBGPCommand(client, []string{"withdraw"}); err == nil {
		t.Fatal("expected error for missing prefix in withdraw, got nil")
	}

	// Unknown XDP action
	if err := HandleXDPCommand(client, []string{"unknown"}); err == nil {
		t.Fatal("expected error for unknown xdp action, got nil")
	}

	// Missing args in RPKI
	if err := HandleNetworkCommand(client, []string{"bgp", "rpki", "202456"}); err == nil {
		t.Fatal("expected error for missing prefix in rpki, got nil")
	}

	// Invalid ASN
	if err := HandleNetworkCommand(client, []string{"bgp", "rpki", "not-an-asn", "103.144.180.0/24"}); err == nil {
		t.Fatal("expected error for invalid ASN, got nil")
	}

	// Insufficient flow args
	if err := HandleNetworkCommand(client, []string{"flow", "dispatch", "10.0.0.1"}); err == nil {
		t.Fatal("expected error for insufficient flow args, got nil")
	}

	// 15. PoP List
	if err := HandleNetworkCommand(client, []string{"pop", "list"}); err != nil {
		t.Fatalf("expected nil for pop list, got: %v", err)
	}

	// 16. PoP Routes
	if err := HandleNetworkCommand(client, []string{"pop", "routes"}); err != nil {
		t.Fatalf("expected nil for pop routes, got: %v", err)
	}

	// 17. PoP Probe
	if err := HandleNetworkCommand(client, []string{"pop", "probe"}); err != nil {
		t.Fatalf("expected nil for pop probe, got: %v", err)
	}

	// 18. PoP Unknown action
	if err := HandleNetworkCommand(client, []string{"pop", "unknown"}); err == nil {
		t.Fatal("expected error for unknown pop action, got nil")
	}
}
