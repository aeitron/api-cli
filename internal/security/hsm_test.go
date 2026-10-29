package security

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHSMCommands_EndToEnd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1/kms/hsm/status":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":           "ONLINE",
				"mode":             "PHYSICAL_PKCS11_HARDWARE",
				"slot_id":          0,
				"label":            "SovereignHSM-Dhaka-Slot0",
				"manufacturer":     "SovereignSilicon Corp",
				"model":            "SEC-HSM-2026",
				"serial_number":    "BD-DHAKA-HSM-001",
				"fips_level":       "FIPS 140-3 Level 3",
				"compliance":       "Bangladesh NDMA 2026 Chapter 4",
				"hardware_locked":  true,
				"active_sessions":  1,
				"max_sessions":     64,
				"free_memory_kb":   2048,
				"firmware_version": "v3.2.1-fips",
				"hardware_version": "v1.0",
			})
		case "/v1/kms/hsm/slots":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"count": 2,
				"slots": []map[string]interface{}{
					{
						"slot_id":         0,
						"region":          "dhaka",
						"role":            "PRIMARY_ACTIVE",
						"token_label":     "SovereignHSM-Dhaka-Slot0",
						"manufacturer_id": "SovereignSilicon Corp",
						"token_present":   true,
					},
					{
						"slot_id":         1,
						"region":          "singapore",
						"role":            "STANDBY_REPLICATED",
						"token_label":     "SovereignHSM-Singapore-Slot1",
						"manufacturer_id": "SovereignSilicon Corp",
						"token_present":   true,
					},
				},
			})
		case "/v1/kms/hsm/test-wrap":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":      "PASSED",
				"kek_id":      "kek-dhaka-sovereign",
				"operation":   "CKM_AES_KEY_GEN_AND_WRAP",
				"latency_us":  145,
				"verified_at": "2026-10-21T22:30:00Z",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client := &Client{
		BaseURL:    ts.URL,
		AuthToken:  "test-api-key",
		HTTPClient: ts.Client(),
	}

	// 1. Help
	PrintHSMHelp()

	// 2. Empty args
	if err := HandleHSMCommand(client, []string{}); err != nil {
		t.Fatalf("expected nil on empty args, got: %v", err)
	}

	// 3. Subcommand "help"
	if err := HandleHSMCommand(client, []string{"help"}); err != nil {
		t.Fatalf("expected nil on help, got: %v", err)
	}

	// 4. Subcommand "status"
	if err := HandleHSMCommand(client, []string{"status"}); err != nil {
		t.Fatalf("handleHSMStatus failed: %v", err)
	}

	// 5. Subcommand "slots"
	if err := HandleHSMCommand(client, []string{"slots"}); err != nil {
		t.Fatalf("handleHSMSlots failed: %v", err)
	}

	// 6. Subcommand "test-wrap"
	if err := HandleHSMCommand(client, []string{"test-wrap"}); err != nil {
		t.Fatalf("handleHSMTestWrap failed: %v", err)
	}

	// 7. Unknown subcommand
	if err := HandleHSMCommand(client, []string{"unknown-cmd"}); err == nil {
		t.Fatalf("expected error on unknown subcommand, got nil")
	}
}
