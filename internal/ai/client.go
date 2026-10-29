package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AIProvider represents an AI compute provider registered in the Control Plane.
type AIProvider struct {
	ID                      string    `json:"id"`
	ProjectID               string    `json:"project_id"`
	Name                    string    `json:"name"`
	Endpoint                string    `json:"endpoint"`
	ProviderType            string    `json:"provider_type"`
	APIKey                  string    `json:"api_key,omitempty"`
	CostPerMTokens          float64   `json:"cost_per_m_tokens"`
	Priority                int       `json:"priority"`
	SovereigntyJurisdiction string    `json:"sovereignty_jurisdiction,omitempty"`
	FailoverCooldownSecs    int       `json:"failover_cooldown_secs"`
	Enabled                 bool      `json:"enabled"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// CreateAIProviderRequest defines payload to register a new AI upstream provider.
type CreateAIProviderRequest struct {
	ID                      string  `json:"id"`
	ProjectID               string  `json:"project_id"`
	Name                    string  `json:"name"`
	Endpoint                string  `json:"endpoint"`
	ProviderType            string  `json:"provider_type"`
	APIKey                  string  `json:"api_key,omitempty"`
	CostPerMTokens          float64 `json:"cost_per_m_tokens"`
	Priority                int     `json:"priority"`
	SovereigntyJurisdiction string  `json:"sovereignty_jurisdiction,omitempty"`
	FailoverCooldownSecs    int     `json:"failover_cooldown_secs"`
	Enabled                 bool    `json:"enabled"`
}

// EconomicsSummary tracks aggregate token usage, costs, and arbitrage savings.
type EconomicsSummary struct {
	TotalTokensRouted uint64  `json:"total_tokens_routed"`
	PromptTokens      uint64  `json:"prompt_tokens"`
	CompletionTokens  uint64  `json:"completion_tokens"`
	CostSpentUSD      float64 `json:"cost_spent_usd"`
	CostSavedUSD      float64 `json:"cost_saved_usd"`
	SavingsPercentage float64 `json:"savings_percentage"`
}

// ProviderTelemetry encapsulates real-time operational health and metrics per provider.
type ProviderTelemetry struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	Endpoint           string  `json:"endpoint"`
	Type               string  `json:"type"`
	Jurisdiction       string  `json:"jurisdiction"`
	IsAvailable        bool    `json:"is_available"`
	TotalRequests      uint64  `json:"total_requests"`
	SuccessfulRequests uint64  `json:"successful_requests"`
	RateLimited429     uint64  `json:"rate_limited_429"`
	FailoversTriggered uint64  `json:"failovers_triggered"`
	AvgTTFTMs          float64 `json:"avg_ttft_ms"`
	CostPerMTokens     float64 `json:"cost_per_m_tokens"`
}

// AnalyticsResponse is returned by GET /v1/aeitron/ai/analytics.
type AnalyticsResponse struct {
	Status         string              `json:"status"`
	Engine         string              `json:"engine"`
	Timestamp      string              `json:"timestamp"`
	Economics      EconomicsSummary    `json:"economics"`
	ProvidersCount int                 `json:"providers_count"`
	Providers      []ProviderTelemetry `json:"providers"`
}

// TestInferenceResult represents the outcome of an end-to-end inference request.
type TestInferenceResult struct {
	StatusCode          int
	ProviderID          string
	Model               string
	TTFTMs              string
	FailoverCount       string
	TotalDurationMs     string
	RoutingStrategy     string
	CostPerMTokens      string
	TokensTotal         string
	EstimatedSavingsUSD string
	ResponseContent     string
}

// Client interacts with Aeitron Control Plane and Edge Gateway APIs.
type Client struct {
	ControlPlaneURL string
	GatewayURL      string
	AuthToken       string
	HTTPClient      *http.Client
}

// NewClient initializes a client with environment variable defaults.
func NewClient() *Client {
	cpURL := os.Getenv("AEITRON_CP_ENDPOINT")
	if cpURL == "" {
		cpURL = "http://127.0.0.1:9091"
	}
	gwURL := os.Getenv("AEITRON_GATEWAY_ENDPOINT")
	if gwURL == "" {
		gwURL = "http://127.0.0.1:8080"
	}
	token := os.Getenv("AEITRON_CP_AUTH_TOKEN")
	if token == "" {
		token = "dev-fixture-key-01"
	}

	return &Client{
		ControlPlaneURL: strings.TrimRight(cpURL, "/"),
		GatewayURL:      strings.TrimRight(gwURL, "/"),
		AuthToken:       token,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ListProviders retrieves the registered AI providers for a project.
func (c *Client) ListProviders(ctx context.Context, projectID string) ([]AIProvider, error) {
	if projectID == "" {
		projectID = "proj-default"
	}
	url := fmt.Sprintf("%s/v1/ai/providers?project_id=%s", c.ControlPlaneURL, projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("control plane connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Primary: Decode Control Plane API envelope format {"providers": [...], "count": N}
	var envelope struct {
		Providers []AIProvider `json:"providers"`
		Count     int          `json:"count"`
	}
	if err := json.Unmarshal(bodyBytes, &envelope); err == nil && (envelope.Providers != nil || bytes.Contains(bodyBytes, []byte(`"providers"`))) {
		if envelope.Providers == nil {
			return []AIProvider{}, nil
		}
		return envelope.Providers, nil
	}

	// Fallback: Raw array format [...] for backward compatibility
	var rawProviders []AIProvider
	if err := json.Unmarshal(bodyBytes, &rawProviders); err == nil {
		if rawProviders == nil {
			return []AIProvider{}, nil
		}
		return rawProviders, nil
	}

	return nil, fmt.Errorf("failed to decode provider list response: %s", string(bodyBytes))
}

// AddProvider registers a new upstream AI provider in the Control Plane Key Vault.
func (c *Client) AddProvider(ctx context.Context, providerReq CreateAIProviderRequest) (*AIProvider, error) {
	if providerReq.ProjectID == "" {
		providerReq.ProjectID = "proj-default"
	}
	data, err := json.Marshal(providerReq)
	if err != nil {
		return nil, fmt.Errorf("failed to encode provider request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/ai/providers", c.ControlPlaneURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("control plane connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var created AIProvider
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode created provider: %w", err)
	}
	return &created, nil
}

// DeleteProvider deletes an AI provider from the Control Plane Key Vault.
func (c *Client) DeleteProvider(ctx context.Context, id string) error {
	url := fmt.Sprintf("%s/v1/ai/providers/%s", c.ControlPlaneURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("failed to construct request: %w", err)
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("control plane connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetAnalytics fetches live token volumes, costs, and savings from the Gateway.
func (c *Client) GetAnalytics(ctx context.Context) (*AnalyticsResponse, error) {
	url := fmt.Sprintf("%s/v1/aeitron/ai/analytics", c.GatewayURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway connection error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gateway returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var analytics AnalyticsResponse
	if err := json.NewDecoder(resp.Body).Decode(&analytics); err != nil {
		return nil, fmt.Errorf("failed to decode analytics response: %w", err)
	}
	return &analytics, nil
}

// TestInference dispatches an AI inference request across the Aeitron Universal AI Director.
func (c *Client) TestInference(ctx context.Context, model, prompt, strategy, jurisdiction string) (*TestInferenceResult, error) {
	if model == "" {
		model = "llama-3.3-70b"
	}
	if prompt == "" {
		prompt = "Where should this AI workload run right now?"
	}
	if strategy == "" {
		strategy = "cost"
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"temperature": 0.7,
		"max_tokens":  256,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request payload: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.GatewayURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to construct request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Aeitron-Routing-Strategy", strategy)
	if jurisdiction != "" {
		req.Header.Set("X-Aeitron-Jurisdiction", jurisdiction)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway dispatch failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &TestInferenceResult{
		StatusCode:          resp.StatusCode,
		ProviderID:          resp.Header.Get("X-Aeitron-Provider"),
		Model:               model,
		TTFTMs:              resp.Header.Get("X-Aeitron-TTFT-Ms"),
		FailoverCount:       resp.Header.Get("X-Aeitron-Failover-Count"),
		TotalDurationMs:     resp.Header.Get("X-Aeitron-Total-Duration-Ms"),
		RoutingStrategy:     resp.Header.Get("X-Aeitron-Routing-Strategy"),
		CostPerMTokens:      resp.Header.Get("X-Aeitron-Cost-Per-MTokens"),
		TokensTotal:         resp.Header.Get("X-Aeitron-Tokens-Total"),
		EstimatedSavingsUSD: resp.Header.Get("X-Aeitron-Estimated-Savings-USD"),
	}

	// Try extracting text content from chat completions response format
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(bodyBytes, &chatResp); err == nil {
		if chatResp.Error != nil && chatResp.Error.Message != "" {
			result.ResponseContent = fmt.Sprintf("Error: %s", chatResp.Error.Message)
		} else if len(chatResp.Choices) > 0 {
			result.ResponseContent = chatResp.Choices[0].Message.Content
		} else {
			result.ResponseContent = string(bodyBytes)
		}
	} else {
		result.ResponseContent = string(bodyBytes)
	}

	return result, nil
}

// AIRouteRequest defines the input payload to route an AI inference prompt.
type AIRouteRequest struct {
	Model                string  `json:"model"`
	Prompt               string  `json:"prompt"`
	PromptPrefixHash     string  `json:"prompt_prefix_hash,omitempty"`
	ClientOrigin         string  `json:"client_origin"`
	RequiredJurisdiction string  `json:"required_jurisdiction,omitempty"`
	Strategy             string  `json:"strategy"`
	MaxLatencyP95Ms      int     `json:"max_latency_p95_ms,omitempty"`
	MaxCostPerHour       float64 `json:"max_cost_per_hour,omitempty"`
	TenantID             string  `json:"tenant_id,omitempty"`
	ProjectID            string  `json:"project_id,omitempty"`
}

// AIRouteResponse defines the decision returned by the Multi-Objective AI Router.
type AIRouteResponse struct {
	Status                         string  `json:"status"`
	AssignedClusterID              string  `json:"assigned_cluster_id"`
	ClusterName                    string  `json:"cluster_name"`
	Region                         string  `json:"region"`
	Provider                       string  `json:"provider"`
	Jurisdiction                   string  `json:"jurisdiction"`
	MeasuredRTTMs                  int     `json:"measured_rtt_ms"`
	EstimatedTTFTMs                float64 `json:"estimated_ttft_ms"`
	KVCacheHit                     bool    `json:"kv_cache_hit"`
	KVCacheWarmth                  float64 `json:"kv_cache_warmth"`
	PrefillLatencyReductionPercent float64 `json:"prefill_latency_reduction_percent"`
	HourlyCost                     float64 `json:"hourly_cost"`
	CarbonGCO2PerKWh               float64 `json:"carbon_gco2_kwh"`
	CarbonClassification           string  `json:"carbon_classification"`
	OptimizationScore              float64 `json:"optimization_score"`
	Reason                         string  `json:"reason"`
}

// GPUCluster represents telemetry and capacity for an edge GPU cluster.
type GPUCluster struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Region             string         `json:"region"`
	Jurisdiction       string         `json:"jurisdiction"`
	Provider           string         `json:"provider"`
	GPUType            string         `json:"gpu_type"`
	TotalGPUs          int            `json:"total_gpus"`
	AvailableGPUs      int            `json:"available_gpus"`
	CostPerGPUHour     float64        `json:"cost_per_gpu_hour"`
	CarbonIntensity    float64        `json:"carbon_intensity"`
	LatencyRTTMs       map[string]int `json:"latency_rtt_ms"`
	ActiveModels       []string       `json:"active_models"`
	KVCacheCapacityMB  int            `json:"kv_cache_capacity_mb"`
	KVCacheUsedMB      int            `json:"kv_cache_used_mb"`
	HealthStatus       string         `json:"health_status"`
	TemperatureCelsius float64        `json:"temperature_celsius"`
	PowerWatts         float64        `json:"power_watts"`
	MaxTDPWatts        float64        `json:"max_tdp_watts"`
	ThrottleFactor     float64        `json:"throttle_factor"`
	LastHealthCheck    time.Time      `json:"last_health_check"`
}

// ClusterTelemetryResponse represents the outcome of ingesting GPU telemetry.
type ClusterTelemetryResponse struct {
	ClusterID          string  `json:"cluster_id"`
	HealthStatus       string  `json:"health_status"`
	TemperatureCelsius float64 `json:"temperature_celsius"`
	PowerWatts         float64 `json:"power_watts"`
	ThrottleFactor     float64 `json:"throttle_factor"`
	EventRecorded      bool    `json:"event_recorded"`
}

// ClusterHealthResponse represents detailed thermal and throttle telemetry.
type ClusterHealthResponse struct {
	ClusterID          string    `json:"cluster_id"`
	HealthStatus       string    `json:"health_status"`
	TemperatureCelsius float64   `json:"temperature_celsius"`
	PowerWatts         float64   `json:"power_watts"`
	MaxTDPWatts        float64   `json:"max_tdp_watts"`
	ThrottleFactor     float64   `json:"throttle_factor"`
	LastHealthCheck    time.Time `json:"last_health_check"`
}

// RouteInference invokes the Multi-Objective Intelligent AI Router on the Control Plane.
func (c *Client) RouteInference(ctx context.Context, req AIRouteRequest) (*AIRouteResponse, error) {
	url := fmt.Sprintf("%s/v1/ai/route", strings.TrimRight(c.ControlPlaneURL, "/"))
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to dispatch route request: %w", err)
	}
	defer resp.Body.Close()

	var routeResp AIRouteResponse
	if err := json.NewDecoder(resp.Body).Decode(&routeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &routeResp, nil
}

// ListClusters retrieves live GPU cluster telemetry from the Control Plane.
func (c *Client) ListClusters(ctx context.Context) ([]GPUCluster, error) {
	url := fmt.Sprintf("%s/v1/ai/clusters", strings.TrimRight(c.ControlPlaneURL, "/"))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve clusters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to retrieve clusters (status %d): %s", resp.StatusCode, string(body))
	}

	var data struct {
		Clusters []GPUCluster `json:"clusters"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode clusters response: %w", err)
	}

	return data.Clusters, nil
}

// IngestClusterTelemetry sends live die temperature and power draw to the cluster state machine.
func (c *Client) IngestClusterTelemetry(ctx context.Context, clusterID string, tempCelsius, powerWatts float64) (*ClusterTelemetryResponse, error) {
	url := fmt.Sprintf("%s/v1/ai/clusters/%s/telemetry", strings.TrimRight(c.ControlPlaneURL, "/"), clusterID)
	payload := map[string]float64{
		"temperature_celsius": tempCelsius,
		"power_watts":         powerWatts,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to dispatch telemetry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("telemetry ingestion failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res ClusterTelemetryResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode telemetry response: %w", err)
	}

	return &res, nil
}

// GetClusterHealth retrieves the current thermal state and throttle factor for a cluster.
func (c *Client) GetClusterHealth(ctx context.Context, clusterID string) (*ClusterHealthResponse, error) {
	url := fmt.Sprintf("%s/v1/ai/clusters/%s/health", strings.TrimRight(c.ControlPlaneURL, "/"), clusterID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("health query failed (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res ClusterHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &res, nil
}
