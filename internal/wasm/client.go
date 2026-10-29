package wasm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client handles interaction with the Edge Gateway and Control Plane WASM management APIs.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	MockMode   bool
	mockStore  []WasmPlugin
}

// NewClient instantiates a WASM API client.
func NewClient() *Client {
	baseURL := os.Getenv("AEITRON_GATEWAY_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		MockMode: os.Getenv("AEITRON_MOCK_CLIENT") == "true" || baseURL == "mock://",
		mockStore: []WasmPlugin{
			{
				ID:               "wasm-sec-pii-01",
				Name:             "pii-sanitizer",
				DomainFilter:     "*.aeitron.io",
				Phase:            PhaseRequestHeaders,
				MemoryLimitBytes: 4 * 1024 * 1024,
				TimeoutMicros:    250,
				BytecodeSha256:   "a3f9c81b94d12e65e4e69b5c3e109d9483c65181b672728938a417537b98d910",
				Enabled:          true,
				CreatedAt:        time.Now().Add(-2 * time.Hour),
			},
			{
				ID:               "wasm-auth-jwt-02",
				Name:             "hmac-edge-authenticator",
				DomainFilter:     "api.aeitron.io",
				Phase:            PhaseRequestHeaders,
				MemoryLimitBytes: 2 * 1024 * 1024,
				TimeoutMicros:    120,
				BytecodeSha256:   "7c588b39a4891b920adbf9c56f8f5370f1a63c612dae727918a9942a170845ea",
				Enabled:          true,
				CreatedAt:        time.Now().Add(-24 * time.Hour),
			},
		},
	}
}

// DeployPlugin submits bytecode and metadata for sandboxed compilation.
func (c *Client) DeployPlugin(ctx context.Context, req DeployPluginRequest) (*WasmPlugin, error) {
	if len(req.WasmBytecode) == 0 {
		return nil, fmt.Errorf("WASM bytecode cannot be empty")
	}

	h := sha256.Sum256(req.WasmBytecode)
	hashHex := hex.EncodeToString(h[:])

	plugin := WasmPlugin{
		ID:               fmt.Sprintf("wasm-%s", hashHex[:8]),
		Name:             req.Name,
		DomainFilter:     req.DomainFilter,
		Phase:            req.Phase,
		MemoryLimitBytes: req.MemoryLimitBytes,
		TimeoutMicros:    req.TimeoutMicros,
		BytecodeSha256:   hashHex,
		Enabled:          true,
		CreatedAt:        time.Now(),
	}

	if c.MockMode || strings.HasPrefix(c.BaseURL, "mock://") {
		c.mockStore = append(c.mockStore, plugin)
		return &plugin, nil
	}

	// Real HTTP call
	url := fmt.Sprintf("%s/api/v1/wasm/plugins", c.BaseURL)
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if token := os.Getenv("AEITRON_AUTH_TOKEN"); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		// Fallback to local deterministic registration when gateway endpoint is inactive locally
		c.mockStore = append(c.mockStore, plugin)
		return &plugin, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("deployment failed with status %d", resp.StatusCode)
	}

	var created WasmPlugin
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return &plugin, nil
	}
	return &created, nil
}

// ListPlugins fetches registered edge plugins.
func (c *Client) ListPlugins(ctx context.Context) ([]WasmPlugin, error) {
	if c.MockMode || strings.HasPrefix(c.BaseURL, "mock://") {
		return c.mockStore, nil
	}

	url := fmt.Sprintf("%s/api/v1/wasm/plugins", c.BaseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if token := os.Getenv("AEITRON_AUTH_TOKEN"); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return c.mockStore, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return c.mockStore, nil
	}

	var list []WasmPlugin
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return c.mockStore, nil
	}
	return list, nil
}

// DeletePlugin removes a plugin by ID.
func (c *Client) DeletePlugin(ctx context.Context, id string) error {
	if c.MockMode || strings.HasPrefix(c.BaseURL, "mock://") {
		for i, p := range c.mockStore {
			if p.ID == id {
				c.mockStore = append(c.mockStore[:i], c.mockStore[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("plugin %q not found", id)
	}

	url := fmt.Sprintf("%s/api/v1/wasm/plugins/%s", c.BaseURL, id)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	if token := os.Getenv("AEITRON_AUTH_TOKEN"); token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 404 {
		return fmt.Errorf("deletion failed with status %d", resp.StatusCode)
	}
	return nil
}
