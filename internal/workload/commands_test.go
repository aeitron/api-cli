package workload

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseManifestBytes_YAML(t *testing.T) {
	yamlData := `
apiVersion: aeitron.io/v1
kind: Workload
metadata:
  name: test-gpu-inference
  tenant_id: tenant-ai-01
  project_id: proj-nlp
spec:
  workload_id: wl-gpu-999
  type: GPU_INFERENCE
  jurisdiction: BANGLADESH
  strict_sovereignty: true
  max_latency_p95_ms: 25
  max_cost_budget_usd: 4.50
  cpu_millicores: 8000
  memory_bytes: 34359738368
  gpu_count: 2
  required_gpu_model: H100
  objective: LOW_LATENCY
  client_origin_region: dhaka
`
	m, err := ParseManifestBytes([]byte(yamlData))
	if err != nil {
		t.Fatalf("failed to parse YAML manifest: %v", err)
	}

	if m.Metadata.Name != "test-gpu-inference" {
		t.Errorf("expected name 'test-gpu-inference', got %q", m.Metadata.Name)
	}
	if m.Metadata.TenantID != "tenant-ai-01" {
		t.Errorf("expected tenant 'tenant-ai-01', got %q", m.Metadata.TenantID)
	}
	if m.Spec.WorkloadID != "wl-gpu-999" {
		t.Errorf("expected workload_id 'wl-gpu-999', got %q", m.Spec.WorkloadID)
	}
	if m.Spec.Type != TypeGPUInference {
		t.Errorf("expected type GPU_INFERENCE, got %s", m.Spec.Type)
	}
	if m.Spec.Jurisdiction != JurisdictionBangladesh {
		t.Errorf("expected jurisdiction BANGLADESH, got %s", m.Spec.Jurisdiction)
	}
	if !m.Spec.StrictSovereignty {
		t.Errorf("expected strict_sovereignty true")
	}
	if m.Spec.GPUCount != 2 {
		t.Errorf("expected gpu_count 2, got %d", m.Spec.GPUCount)
	}
}

func TestParseManifestBytes_JSON(t *testing.T) {
	jsonData := `{
		"apiVersion": "aeitron.io/v1",
		"kind": "Workload",
		"metadata": {
			"name": "json-workload",
			"tenant_id": "tenant-007"
		},
		"spec": {
			"workload_id": "wl-json-001",
			"type": "API_GATEWAY",
			"gpu_count": 0
		}
	}`
	m, err := ParseManifestBytes([]byte(jsonData))
	if err != nil {
		t.Fatalf("failed to parse JSON manifest: %v", err)
	}
	if m.Spec.WorkloadID != "wl-json-001" {
		t.Errorf("expected wl-json-001, got %s", m.Spec.WorkloadID)
	}
}

func TestValidateManifest_MissingTenant(t *testing.T) {
	m := &Manifest{
		Kind: "Workload",
		Metadata: MetadataSpec{
			Name: "test-invalid",
		},
		Spec: WorkloadSpec{
			WorkloadID: "test-invalid",
		},
	}
	err := ValidateManifest(m)
	if err == nil {
		t.Fatal("expected error for missing tenant_id")
	}
	if !strings.Contains(err.Error(), "tenant_id is required") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestValidateManifest_ExcessiveGPUs(t *testing.T) {
	m := &Manifest{
		Kind: "Workload",
		Metadata: MetadataSpec{
			Name:     "test-excessive",
			TenantID: "tenant-valid",
		},
		Spec: WorkloadSpec{
			WorkloadID: "test-excessive",
			GPUCount:   128, // Exceeds 64 limit
		},
	}
	err := ValidateManifest(m)
	if err == nil {
		t.Fatal("expected error for excessive GPU request")
	}
	if !strings.Contains(err.Error(), "exceeds maximum limit") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerateTemplate(t *testing.T) {
	tmplGPU := GenerateTemplate("gpu")
	if !strings.Contains(tmplGPU, "GPU_INFERENCE") {
		t.Errorf("expected GPU_INFERENCE in GPU template, got:\n%s", tmplGPU)
	}

	tmplBD := GenerateTemplate("sovereign")
	if !strings.Contains(tmplBD, "BANGLADESH") || !strings.Contains(tmplBD, "strict_sovereignty: true") {
		t.Errorf("expected BANGLADESH strict sovereignty in sovereign template, got:\n%s", tmplBD)
	}
}

func TestClient_ScheduleAndRelease(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/workloads/schedule" && r.Method == http.MethodPost:
			resp := ScheduleResponse{
				WorkloadID:      "wl-mock-01",
				IngressEndpoint: "https://dha.edge.aeitron.net",
				Status: WorkloadStatus{
					Phase:             "PHASE_SCHEDULED",
					AssignedRegion:    "dhaka",
					AssignedNode:      "node-dha-01",
					AssignedProvider:  "baremetal-bd",
					MeasuredLatencyMs: 6,
					RoutingReason:     "Optimal RTT & NDMA 2026 match",
					ScheduledAt:       time.Now(),
					LeaseExpiresAt:    time.Now().Add(30 * time.Minute),
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case r.URL.Path == "/v1/workloads/wl-mock-01" && r.Method == http.MethodGet:
			resp := ScheduleResponse{
				WorkloadID: "wl-mock-01",
				Status: WorkloadStatus{
					Phase:          "PHASE_RUNNING",
					AssignedRegion: "dhaka",
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		case r.URL.Path == "/v1/workloads/wl-mock-01/release" && r.Method == http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "RELEASED"})
		case r.URL.Path == "/v1/nodes" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"nodes": []NodeSummary{
					{ID: "node-dha-01", Region: "dhaka", AvailableGPUs: 8, Healthy: true},
				},
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

	manifest := &Manifest{
		Metadata: MetadataSpec{
			Name:     "wl-mock-01",
			TenantID: "tenant-mock",
		},
		Spec: WorkloadSpec{
			WorkloadID: "wl-mock-01",
			Type:       TypeGPUInference,
		},
	}

	// 1. Schedule
	resp, err := client.ScheduleWorkload(context.Background(), manifest)
	if err != nil {
		t.Fatalf("failed to schedule: %v", err)
	}
	if resp.WorkloadID != "wl-mock-01" {
		t.Errorf("expected wl-mock-01, got %s", resp.WorkloadID)
	}

	// 2. Get
	got, err := client.GetWorkload(context.Background(), "wl-mock-01")
	if err != nil {
		t.Fatalf("failed to get workload: %v", err)
	}
	if got.Status.Phase != "PHASE_RUNNING" {
		t.Errorf("expected PHASE_RUNNING, got %s", got.Status.Phase)
	}

	// 3. List Nodes
	nodes, err := client.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("failed to list nodes: %v", err)
	}
	if len(nodes) != 1 || nodes[0].ID != "node-dha-01" {
		t.Errorf("unexpected nodes returned: %+v", nodes)
	}

	// 4. Release
	if err := client.ReleaseWorkload(context.Background(), "wl-mock-01"); err != nil {
		t.Fatalf("failed to release: %v", err)
	}
}

func TestHandleDeployCommands(t *testing.T) {
	// Create temporary manifest file
	tmpDir := t.TempDir()
	manifestFile := filepath.Join(tmpDir, "workload.yaml")
	content := GenerateTemplate("gpu")
	if err := os.WriteFile(manifestFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test manifest: %v", err)
	}

	client := NewClient()

	// Test validate
	if err := HandleDeployCommand(client, []string{"validate", "-f", manifestFile}); err != nil {
		t.Fatalf("validate command failed: %v", err)
	}

	// Test dry-run
	if err := HandleDeployCommand(client, []string{"dry-run", "-f", manifestFile}); err != nil {
		t.Fatalf("dry-run command failed: %v", err)
	}

	// Test template
	if err := HandleDeployCommand(client, []string{"template", "--type", "sovereign"}); err != nil {
		t.Fatalf("template command failed: %v", err)
	}
}

func TestHandleDeployCommand_Journal(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/journal" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events": []PlacementEvent{
					{
						Sequence:    1,
						Timestamp:   time.Now(),
						PrevHash:    "0000000000000000000000000000000000000000000000000000000000000000",
						CurrentHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
						EventType:   "PLACEMENT_SCHEDULED",
						WorkloadID:  "wl-core-01",
						TenantID:    "tenant-gov",
						NodeID:      "node-dha-01",
						Region:      "dhaka",
						Priority:    1000,
						Reason:      "Sovereign Placement",
					},
				},
			})
		case r.URL.Path == "/v1/journal/verify" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(JournalVerifyResponse{
				Status:       "VERIFIED",
				EntriesCount: 1,
				MerkleRoot:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
				Algorithm:    "SHA-256",
			})
		case r.URL.Path == "/v1/journal/proof" && r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(JournalProofResponse{
				Proof: &MerkleInclusionProof{
					Sequence:   1,
					LeafHash:   "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					MerkleRoot: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
					Path:       []MerkleProofStep{},
				},
				Verified: true,
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

	// 1. journal list
	if err := HandleDeployCommand(client, []string{"journal", "list"}); err != nil {
		t.Fatalf("journal list failed: %v", err)
	}

	// 2. journal verify
	if err := HandleDeployCommand(client, []string{"journal", "verify"}); err != nil {
		t.Fatalf("journal verify failed: %v", err)
	}

	// 3. journal proof
	if err := HandleDeployCommand(client, []string{"journal", "proof", "--seq", "1"}); err != nil {
		t.Fatalf("journal proof failed: %v", err)
	}
}

func TestConsensusCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/scheduler/consensus/status" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(RaftClusterStatusDTO{
				ClusterID:              "aeitron-test-consensus",
				NodeID:                 "bd-dac",
				Region:                 "bd-dac",
				Role:                   "LEADER",
				CurrentTerm:            2,
				LeaderID:               "bd-dac",
				CommitIndex:            15,
				LastApplied:            15,
				LogEntriesCount:        16,
				LatestLogHash:          "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				PqcAttestationVerified: true,
				TotalPeersCount:        4,
				HealthyPeersCount:      4,
				QuorumSatisfied:        true,
				Peers: []RaftPeerDTO{
					{ID: "sg-sin", Region: "sg-sin", Jurisdiction: "SINGAPORE", SubseaCable: "SMW6", RTTMs: 28, NextIndex: 16, MatchIndex: 15, Healthy: true},
					{ID: "de-fra", Region: "de-fra", Jurisdiction: "EU", SubseaCable: "AAE-1", RTTMs: 84, NextIndex: 16, MatchIndex: 15, Healthy: true},
					{ID: "is-north-1", Region: "is-north-1", Jurisdiction: "EU", SubseaCable: "FARICE-1", RTTMs: 112, NextIndex: 16, MatchIndex: 15, Healthy: true},
					{ID: "us-iad", Region: "us-iad", Jurisdiction: "US", SubseaCable: "MAREA", RTTMs: 145, NextIndex: 16, MatchIndex: 15, Healthy: true},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	client := &Client{
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	}

	// Test consensus subcommand
	if err := HandleDeployCommand(client, []string{"consensus"}); err != nil {
		t.Fatalf("consensus subcommand failed: %v", err)
	}

	// Test offline fallback
	offlineClient := &Client{
		BaseURL:    "http://127.0.0.1:54321", // offline port
		HTTPClient: &http.Client{Timeout: 100 * time.Millisecond},
	}
	if err := HandleDeployCommand(offlineClient, []string{"consensus"}); err != nil {
		t.Fatalf("consensus offline fallback failed: %v", err)
	}
}
