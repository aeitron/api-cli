package security

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client communicates with the Aeitron Security and Sovereignty Policy Engine.
type Client struct {
	BaseURL    string
	AuthToken  string
	HTTPClient *http.Client
}

// NewClient initializes a client with environment defaults.
func NewClient() *Client {
	endpoint := os.Getenv("AEITRON_SECURITY_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("AEITRON_CP_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = "http://127.0.0.1:9094"
	}

	token := os.Getenv("AEITRON_CP_AUTH_TOKEN")
	if token == "" {
		token = os.Getenv("AEITRON_API_KEY")
	}

	return &Client{
		BaseURL:   strings.TrimRight(endpoint, "/"),
		AuthToken: token,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// EvaluateSovereignty requests data residency and NDMA 2026 sovereignty evaluation.
func (c *Client) EvaluateSovereignty(ctx context.Context, req SovereigntyEvaluationRequest) (*SovereigntyEvaluationResult, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/v1/security/sovereignty/evaluate"
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
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sovereignty evaluation failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var res SovereigntyEvaluationResult
	if err := json.Unmarshal(respBody, &res); err != nil {
		return nil, fmt.Errorf("failed to decode evaluation response: %w", err)
	}
	return &res, nil
}

// BlockIP pushes a dynamic drop rule to the L4 eBPF / XDP firewall.
func (c *Client) BlockIP(ctx context.Context, ip, reason string, durationSeconds int) (*BlockedIPEntry, error) {
	if ip == "" {
		return nil, errors.New("ip address is required")
	}
	payload := map[string]interface{}{
		"ip_address":       ip,
		"reason":           reason,
		"duration_seconds": durationSeconds,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/v1/security/firewall/block"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("block ip failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var entry BlockedIPEntry
	if err := json.Unmarshal(respBody, &entry); err != nil {
		return nil, fmt.Errorf("failed to decode blocked ip entry: %w", err)
	}
	return &entry, nil
}

// UnblockIP removes an IP from the firewall drop list.
func (c *Client) UnblockIP(ctx context.Context, ip string) error {
	if ip == "" {
		return errors.New("ip address is required")
	}
	payload := map[string]string{
		"ip_address": ip,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := c.BaseURL + "/v1/security/firewall/unblock"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unblock ip failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ListBlockedIPs retrieves all active drop entries.
func (c *Client) ListBlockedIPs(ctx context.Context) ([]BlockedIPEntry, error) {
	url := c.BaseURL + "/v1/security/firewall/blocked"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list blocked ips failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Count int              `json:"count"`
		IPs   []BlockedIPEntry `json:"ips"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode blocked ips list: %w", err)
	}
	return result.IPs, nil
}

// GetPolicy retrieves the edge security posture for a domain.
func (c *Client) GetPolicy(ctx context.Context, domainID string) (*SecurityPolicy, error) {
	if domainID == "" {
		return nil, errors.New("domain id is required")
	}

	url := fmt.Sprintf("%s/v1/security/policies/%s", c.BaseURL, domainID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get policy failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var p SecurityPolicy
	if err := json.Unmarshal(respBody, &p); err != nil {
		return nil, fmt.Errorf("failed to decode security policy: %w", err)
	}
	return &p, nil
}

// UpdatePolicy saves edge security posture configurations for a domain.
func (c *Client) UpdatePolicy(ctx context.Context, p *SecurityPolicy) (*SecurityPolicy, error) {
	if p == nil || p.DomainID == "" {
		return nil, errors.New("domain id is required in security policy")
	}

	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/security/policies/%s", c.BaseURL, p.DomainID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("update policy failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var saved SecurityPolicy
	if err := json.Unmarshal(respBody, &saved); err != nil {
		return nil, fmt.Errorf("failed to decode updated policy: %w", err)
	}
	return &saved, nil
}

// GetAuditChain queries the cryptographic Merkle audit journal and verifies integrity.
func (c *Client) GetAuditChain(ctx context.Context) ([]AuditEntry, bool, error) {
	url := c.BaseURL + "/v1/security/audit/chain"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("get audit chain failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Verified bool         `json:"verified"`
		Count    int          `json:"count"`
		Entries  []AuditEntry `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, fmt.Errorf("failed to decode audit chain: %w", err)
	}
	return result.Entries, result.Verified, nil
}

// GetHSMStatus fetches PKCS#11 hardware security module health, active slot, and FIPS metrics.
func (c *Client) GetHSMStatus(ctx context.Context) (map[string]interface{}, error) {
	url := c.BaseURL + "/v1/kms/hsm/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hsm status query failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode hsm status: %w", err)
	}
	return res, nil
}

// GetHSMSlots lists all detected hardware cryptoprocessor slots.
func (c *Client) GetHSMSlots(ctx context.Context) (map[string]interface{}, error) {
	url := c.BaseURL + "/v1/kms/hsm/slots"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hsm slots query failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode hsm slots: %w", err)
	}
	return res, nil
}

// TestHSMWrap performs an active round-trip hardware key wrapping verification.
func (c *Client) TestHSMWrap(ctx context.Context) (map[string]interface{}, error) {
	url := c.BaseURL + "/v1/kms/hsm/test-wrap"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("security service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hsm test-wrap failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode hsm test-wrap: %w", err)
	}
	return res, nil
}
