package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ListProviders(t *testing.T) {
	// 1. Test Control Plane standard envelope format {"providers": [...], "count": 1}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ai/providers" {
			t.Errorf("expected path /v1/ai/providers, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("project_id") != "test-proj" {
			t.Errorf("expected query param project_id=test-proj, got %s", r.URL.Query().Get("project_id"))
		}
		providers := []AIProvider{
			{
				ID:             "vllm-spot",
				ProjectID:      "test-proj",
				Name:           "CoreWeave Spot",
				Endpoint:       "https://vllm.example.com",
				ProviderType:   "vllm",
				CostPerMTokens: 0.50,
				Priority:       1,
				Enabled:        true,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"providers": providers,
			"count":     len(providers),
		})
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	providers, err := client.ListProviders(context.Background(), "test-proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].ID != "vllm-spot" {
		t.Errorf("expected provider ID vllm-spot, got %s", providers[0].ID)
	}
}

func TestClient_ListProviders_RawArrayFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providers := []AIProvider{
			{
				ID:           "raw-prov",
				ProjectID:    "test-proj",
				Name:         "Raw Provider",
				ProviderType: "openai",
				Enabled:      true,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(providers)
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	providers, err := client.ListProviders(context.Background(), "test-proj")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(providers) != 1 || providers[0].ID != "raw-prov" {
		t.Fatalf("expected raw provider fallback to succeed, got %+v", providers)
	}
}

func TestClient_ListProviders_EmptyEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"providers": [], "count": 0}`))
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	providers, err := client.ListProviders(context.Background(), "test-proj")
	if err != nil {
		t.Fatalf("unexpected error on empty envelope: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("expected 0 providers, got %d", len(providers))
	}
}

func TestClient_ListProviders_NullEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"providers": null, "count": 0}`))
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	providers, err := client.ListProviders(context.Background(), "test-proj")
	if err != nil {
		t.Fatalf("unexpected error on null envelope: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("expected 0 providers for null envelope, got %d", len(providers))
	}
}

func TestClient_ListProviders_EmptyRawArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	providers, err := client.ListProviders(context.Background(), "test-proj")
	if err != nil {
		t.Fatalf("unexpected error on raw empty array: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("expected 0 providers, got %d", len(providers))
	}
}

func TestClient_ListProviders_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"broken_json": true`))
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	_, err := client.ListProviders(context.Background(), "test-proj")
	if err == nil {
		t.Fatalf("expected error on malformed JSON, got nil")
	}
}

func TestClient_AddProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req CreateAIProviderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		resp := AIProvider{
			ID:             req.ID,
			ProjectID:      req.ProjectID,
			Name:           req.Name,
			Endpoint:       req.Endpoint,
			ProviderType:   req.ProviderType,
			CostPerMTokens: req.CostPerMTokens,
			Priority:       req.Priority,
			Enabled:        true,
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	created, err := client.AddProvider(context.Background(), CreateAIProviderRequest{
		ID:             "azure-openai",
		Name:           "Azure East US",
		Endpoint:       "https://azure.openai.com",
		ProviderType:   "azure",
		CostPerMTokens: 2.50,
		Priority:       2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.ID != "azure-openai" {
		t.Errorf("expected ID azure-openai, got %s", created.ID)
	}
	if created.CostPerMTokens != 2.50 {
		t.Errorf("expected cost 2.50, got %f", created.CostPerMTokens)
	}
}

func TestClient_GetAnalytics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/aeitron/ai/analytics" {
			t.Errorf("expected /v1/aeitron/ai/analytics, got %s", r.URL.Path)
		}
		resp := AnalyticsResponse{
			Status: "healthy",
			Engine: "Aeitron Universal AI Traffic Director v0.1.0",
			Economics: EconomicsSummary{
				TotalTokensRouted: 50000,
				PromptTokens:      25000,
				CompletionTokens:  25000,
				CostSpentUSD:      0.025,
				CostSavedUSD:      0.100,
				SavingsPercentage: 80.0,
			},
			ProvidersCount: 1,
			Providers: []ProviderTelemetry{
				{
					ID:             "spot-coreweave",
					Type:           "vllm",
					TotalRequests:  50,
					CostPerMTokens: 0.50,
					IsAvailable:    true,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		GatewayURL: server.URL,
		HTTPClient: server.Client(),
	}

	analytics, err := client.GetAnalytics(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if analytics.Economics.TotalTokensRouted != 50000 {
		t.Errorf("expected 50000 tokens, got %d", analytics.Economics.TotalTokensRouted)
	}
	if analytics.Economics.SavingsPercentage != 80.0 {
		t.Errorf("expected 80.0%% savings, got %f", analytics.Economics.SavingsPercentage)
	}
}

func TestClient_TestInference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("expected /v1/chat/completions, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Aeitron-Routing-Strategy") != "cost" {
			t.Errorf("expected strategy header cost, got %s", r.Header.Get("X-Aeitron-Routing-Strategy"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Aeitron-Provider", "spot-coreweave")
		w.Header().Set("X-Aeitron-TTFT-Ms", "22")
		w.Header().Set("X-Aeitron-Failover-Count", "0")
		w.Header().Set("X-Aeitron-Total-Duration-Ms", "35")
		w.Header().Set("X-Aeitron-Routing-Strategy", "cost")
		w.Header().Set("X-Aeitron-Cost-Per-MTokens", "0.50")
		w.Header().Set("X-Aeitron-Tokens-Total", "1000")
		w.Header().Set("X-Aeitron-Estimated-Savings-USD", "0.002000")

		body := map[string]interface{}{
			"id": "chatcmpl-test",
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": "This workload should run on CoreWeave Spot in Virginia.",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	client := &Client{
		GatewayURL: server.URL,
		HTTPClient: server.Client(),
	}

	result, err := client.TestInference(context.Background(), "llama-3.3-70b", "Where to run?", "cost", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ProviderID != "spot-coreweave" {
		t.Errorf("expected provider spot-coreweave, got %s", result.ProviderID)
	}
	if result.TokensTotal != "1000" {
		t.Errorf("expected tokens 1000, got %s", result.TokensTotal)
	}
	if result.EstimatedSavingsUSD != "0.002000" {
		t.Errorf("expected savings 0.002000, got %s", result.EstimatedSavingsUSD)
	}
	if result.ResponseContent != "This workload should run on CoreWeave Spot in Virginia." {
		t.Errorf("unexpected content: %s", result.ResponseContent)
	}
}

func TestHandleAICommand_Help(t *testing.T) {
	client := &Client{}
	if err := HandleAICommand(client, []string{}); err != nil {
		t.Errorf("expected nil error for empty args, got %v", err)
	}
	if err := HandleAICommand(client, []string{"help"}); err != nil {
		t.Errorf("expected nil error for help, got %v", err)
	}
}

func TestClient_RouteInference(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ai/route" {
			t.Errorf("expected path /v1/ai/route, got %s", r.URL.Path)
		}
		var req AIRouteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		resp := AIRouteResponse{
			Status:                         "ROUTED",
			AssignedClusterID:              "is-green-gpu01",
			ClusterName:                    "Iceland Clean Power Cluster",
			Region:                         "is-north-1",
			Provider:                       "iceland-green-dc",
			Jurisdiction:                   "IS",
			MeasuredRTTMs:                  35,
			EstimatedTTFTMs:                53.0,
			KVCacheHit:                     true,
			KVCacheWarmth:                  1.0,
			PrefillLatencyReductionPercent: 60.0,
			HourlyCost:                     1.45,
			CarbonGCO2PerKWh:               12.0,
			CarbonClassification:           "ZERO_CARBON_CLEAN_POWER",
			OptimizationScore:              18.5,
			Reason:                         "Optimized via Multi-Objective AI Router",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	req := AIRouteRequest{
		Model:        "llama-3.3-70b",
		Prompt:       "Analyze portfolio risk",
		ClientOrigin: "frankfurt",
		Strategy:     "green",
	}

	resp, err := client.RouteInference(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ROUTED" {
		t.Errorf("expected ROUTED, got %s", resp.Status)
	}
	if resp.AssignedClusterID != "is-green-gpu01" {
		t.Errorf("expected is-green-gpu01, got %s", resp.AssignedClusterID)
	}
	if !resp.KVCacheHit {
		t.Errorf("expected KVCacheHit to be true")
	}
}

func TestClient_ListClusters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/ai/clusters" {
			t.Errorf("expected path /v1/ai/clusters, got %s", r.URL.Path)
		}
		clusters := []GPUCluster{
			{
				ID:                "bd-dhaka-dgx01",
				Name:              "Dhaka Sovereign DGX SuperPOD",
				Region:            "ap-south-2",
				Jurisdiction:      "BD",
				Provider:          "baremetal-bd",
				GPUType:           "H100-SXM5-80GB",
				TotalGPUs:         8,
				AvailableGPUs:     7,
				CostPerGPUHour:    2.10,
				CarbonIntensity:   480.0,
				ActiveModels:      []string{"llama-3.3-70b"},
				KVCacheCapacityMB: 655360,
				KVCacheUsedMB:     122880,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"clusters": clusters,
			"count":    len(clusters),
		})
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	clusters, err := client.ListClusters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].ID != "bd-dhaka-dgx01" {
		t.Errorf("expected ID bd-dhaka-dgx01, got %s", clusters[0].ID)
	}
}

func TestHandleAICommand_RouteAndClusters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/ai/route":
			resp := AIRouteResponse{
				Status:            "ROUTED",
				AssignedClusterID: "bd-dhaka-dgx01",
				ClusterName:       "Dhaka Sovereign DGX",
				Region:            "ap-south-2",
				Provider:          "baremetal-bd",
				Jurisdiction:      "BD",
				MeasuredRTTMs:     4,
				EstimatedTTFTMs:   22.0,
				HourlyCost:        2.10,
				CarbonGCO2PerKWh:  480.0,
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case "/v1/ai/clusters":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"clusters": []GPUCluster{
					{
						ID:                 "bd-dhaka-dgx01",
						Name:               "Dhaka DGX",
						Region:             "ap-south-2",
						Jurisdiction:       "BD",
						TotalGPUs:          8,
						AvailableGPUs:      7,
						CostPerGPUHour:     2.10,
						CarbonIntensity:    480.0,
						HealthStatus:       "NOMINAL",
						TemperatureCelsius: 68.5,
						PowerWatts:         520.0,
						MaxTDPWatts:        700.0,
						ThrottleFactor:     1.0,
					},
				},
			})
		case "/v1/ai/clusters/bd-dhaka-dgx01/telemetry":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ClusterTelemetryResponse{
				ClusterID:          "bd-dhaka-dgx01",
				HealthStatus:       "THROTTLED",
				TemperatureCelsius: 86.5,
				PowerWatts:         720.0,
				ThrottleFactor:     0.65,
				EventRecorded:      true,
			})
		case "/v1/ai/clusters/bd-dhaka-dgx01/health":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ClusterHealthResponse{
				ClusterID:          "bd-dhaka-dgx01",
				HealthStatus:       "NOMINAL",
				TemperatureCelsius: 68.5,
				PowerWatts:         520.0,
				MaxTDPWatts:        700.0,
				ThrottleFactor:     1.0,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{
		ControlPlaneURL: server.URL,
		HTTPClient:      server.Client(),
	}

	// Route command test
	if err := HandleAICommand(client, []string{"route", "--prompt", "test inference", "--origin", "dhaka"}); err != nil {
		t.Errorf("expected nil error for route, got %v", err)
	}

	// Clusters command test
	if err := HandleAICommand(client, []string{"clusters"}); err != nil {
		t.Errorf("expected nil error for clusters, got %v", err)
	}

	// Clusters --health command test
	if err := HandleAICommand(client, []string{"clusters", "--health"}); err != nil {
		t.Errorf("expected nil error for clusters --health, got %v", err)
	}

	// Inject thermal test
	if err := HandleAICommand(client, []string{"inject-thermal", "--cluster", "bd-dhaka-dgx01", "--temp", "86.5", "--power", "720.0"}); err != nil {
		t.Errorf("expected nil error for inject-thermal, got %v", err)
	}

	// Cluster health test
	if err := HandleAICommand(client, []string{"cluster-health", "bd-dhaka-dgx01"}); err != nil {
		t.Errorf("expected nil error for cluster-health, got %v", err)
	}
}
