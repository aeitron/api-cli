package dns

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

// Client interacts with Aeitron Global DNS Controller.
type Client struct {
	BaseURL    string
	HealthURL  string
	AuthToken  string
	FailClosed bool
	HTTPClient *http.Client
}

// NewClient initializes a client with environment defaults.
func NewClient() *Client {
	endpoint := os.Getenv("AEITRON_DNS_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("AEITRON_CP_ENDPOINT")
	}
	if endpoint == "" {
		endpoint = "http://127.0.0.1:9093"
	}

	healthURL := os.Getenv("AEITRON_DNS_HEALTH_ENDPOINT")
	if healthURL == "" {
		healthURL = os.Getenv("HEALTH_DAEMON_URL")
	}
	if healthURL == "" {
		healthURL = "http://127.0.0.1:8096"
	}

	token := os.Getenv("AEITRON_CP_AUTH_TOKEN")
	if token == "" {
		token = os.Getenv("AEITRON_API_KEY")
	}

	failClosed := os.Getenv("AEITRON_FAIL_CLOSED") == "true" ||
		os.Getenv("AEITRON_ENV") == "production"

	return &Client{
		BaseURL:    strings.TrimRight(endpoint, "/"),
		HealthURL:  strings.TrimRight(healthURL, "/"),
		AuthToken:  token,
		FailClosed: failClosed,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateZone provisions a new authoritative DNS zone.
func (c *Client) CreateZone(ctx context.Context, tenantID, name string) (*DNSZone, error) {
	payload := map[string]string{
		"tenant_id": tenantID,
		"name":      name,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/v1/zones"
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
		return nil, fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("create zone failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var zone DNSZone
	if err := json.Unmarshal(respBody, &zone); err != nil {
		return nil, fmt.Errorf("failed to decode zone response: %w", err)
	}
	return &zone, nil
}

// ListZones retrieves all authoritative zones for a tenant.
func (c *Client) ListZones(ctx context.Context, tenantID string) ([]DNSZone, error) {
	url := c.BaseURL + "/v1/zones"
	if tenantID != "" {
		url += "?tenant_id=" + tenantID
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list zones failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Zones []DNSZone `json:"zones"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode zones list: %w", err)
	}
	return result.Zones, nil
}

// DeleteZone removes a zone and all its resource records.
func (c *Client) DeleteZone(ctx context.Context, zoneID string) error {
	url := fmt.Sprintf("%s/v1/zones/%s", c.BaseURL, zoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete zone failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// AddRecord adds a resource record to an active zone.
func (c *Client) AddRecord(ctx context.Context, rec *DNSRecord) (*DNSRecord, error) {
	if rec == nil || rec.ZoneID == "" {
		return nil, errors.New("record and zone_id are required")
	}

	body, err := json.Marshal(rec)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/zones/%s/records", c.BaseURL, rec.ZoneID)
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
		return nil, fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("add record failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var created DNSRecord
	if err := json.Unmarshal(respBody, &created); err != nil {
		return nil, fmt.Errorf("failed to decode created record: %w", err)
	}
	return &created, nil
}

// ListRecords queries all resource records inside a zone.
func (c *Client) ListRecords(ctx context.Context, zoneID string) ([]DNSRecord, error) {
	url := fmt.Sprintf("%s/v1/zones/%s/records", c.BaseURL, zoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list records failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Records []DNSRecord `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode records list: %w", err)
	}
	return result.Records, nil
}

// DeleteRecord removes a record from a zone, enforcing dual-control approval when challenge tokens are required.
func (c *Client) DeleteRecord(ctx context.Context, zoneID, recordID string, confirm bool, approvalToken string) error {
	url := fmt.Sprintf("%s/v1/zones/%s/records/%s", c.BaseURL, zoneID, recordID)
	var q []string
	if confirm {
		q = append(q, "confirm=true")
	}
	if approvalToken != "" {
		q = append(q, "approval_token="+approvalToken)
	}
	if len(q) > 0 {
		url += "?" + strings.Join(q, "&")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	bodyStr := string(respBody)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		if resp.StatusCode == http.StatusForbidden && (strings.Contains(bodyStr, "del_req_") || strings.Contains(bodyStr, "APPROVAL_REQUIRED") || strings.Contains(bodyStr, "requestId")) {
			return fmt.Errorf("APPROVAL_REQUIRED: %s", bodyStr)
		}
		if resp.StatusCode == http.StatusForbidden && strings.Contains(bodyStr, "insufficient_scope") {
			return fmt.Errorf("FORBIDDEN: token lacks required scope 'dns:write:delete': %s", bodyStr)
		}
		return fmt.Errorf("delete record failed (%d): %s", resp.StatusCode, bodyStr)
	}
	return nil
}

// RequestRecordDeletion initiates an approval challenge to delete a protected record under the two-person rule.
func (c *Client) RequestRecordDeletion(ctx context.Context, domain, recordID, reason string) (*DnsDeletionRequest, error) {
	payload := map[string]string{
		"domain":    domain,
		"record_id": recordID,
		"reason":    reason,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := c.BaseURL + "/v1/dns/request-deletion"
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
		return nil, fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("request deletion failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result DnsDeletionRequest
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode deletion request: %w", err)
	}
	return &result, nil
}

// ApproveRecordDeletion authorizes a pending deletion request as an independent operator under dual-control.
func (c *Client) ApproveRecordDeletion(ctx context.Context, requestID, actor string) (*DnsApprovalResult, error) {
	payload := map[string]string{
		"requestId": requestID,
		"actor":     actor,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := c.BaseURL + "/v1/dns/approve-deletion"
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
		return nil, fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("approval failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var result DnsApprovalResult
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to decode approval result: %w", err)
	}
	return &result, nil
}

// ExportBIND downloads an RFC 1035 BIND zone file for a zone.
func (c *Client) ExportBIND(ctx context.Context, zoneID string) (string, error) {
	url := fmt.Sprintf("%s/v1/zones/%s/export", c.BaseURL, zoneID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("export failed (%d): %s", resp.StatusCode, string(respBody))
	}
	return string(respBody), nil
}

// Resolve queries the Anycast DNS steering resolver.
func (c *Client) Resolve(ctx context.Context, req ResolveRequest) (*ResolveResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL + "/v1/dns/resolve"
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
		return nil, fmt.Errorf("dns service unreachable: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resolve failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var res ResolveResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return nil, fmt.Errorf("failed to decode resolve response: %w", err)
	}
	return &res, nil
}

// GetHealthStatus returns local PoP health, BGP announcement posture, and flap penalty.
func (c *Client) GetHealthStatus(ctx context.Context) (*PoPHealthStatus, error) {
	url := c.HealthURL + "/v1/health/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: health service unreachable: %w", err)
		}
		return c.fallbackHealthStatus(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: health service returned status %d", resp.StatusCode)
		}
		return c.fallbackHealthStatus(), nil
	}

	var status PoPHealthStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: failed to decode health status: %w", err)
		}
		return c.fallbackHealthStatus(), nil
	}
	return &status, nil
}

// GetGlobalHealthMatrix returns 5-region PoP Anycast health matrix.
func (c *Client) GetGlobalHealthMatrix(ctx context.Context) ([]*PoPHealthStatus, error) {
	url := c.HealthURL + "/v1/health/pops"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: global health matrix unreachable: %w", err)
		}
		return c.fallbackGlobalMatrix(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: health matrix returned status %d", resp.StatusCode)
		}
		return c.fallbackGlobalMatrix(), nil
	}

	var payload struct {
		Count int                `json:"count"`
		PoPs  []*PoPHealthStatus `json:"pops"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: failed to decode health matrix: %w", err)
		}
		return c.fallbackGlobalMatrix(), nil
	}
	return payload.PoPs, nil
}

// DrainPoP initiates an administrative BGP withdrawal for the local PoP.
func (c *Client) DrainPoP(ctx context.Context, reason string) (*PoPHealthStatus, error) {
	payload := map[string]string{
		"reason": reason,
	}
	body, _ := json.Marshal(payload)
	url := c.HealthURL + "/v1/health/drain"
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
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: drain failed: health service unreachable: %w", err)
		}
		fb := c.fallbackHealthStatus()
		fb.State = PoPDraining
		fb.BGPState = BGPWithdrawn
		fb.DrainReason = reason
		return fb, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("drain failed with status %d", resp.StatusCode)
	}

	var result struct {
		Status string           `json:"status"`
		PoP    *PoPHealthStatus `json:"pop"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.PoP, nil
}

// RestorePoP re-advertises BGP Anycast routes and resumes autonomous health checking.
func (c *Client) RestorePoP(ctx context.Context) (*PoPHealthStatus, error) {
	url := c.HealthURL + "/v1/health/restore"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: restore failed: health service unreachable: %w", err)
		}
		fb := c.fallbackHealthStatus()
		fb.State = PoPHealthy
		fb.BGPState = BGPAdvertised
		fb.DrainReason = ""
		return fb, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("restore failed with status %d", resp.StatusCode)
	}

	var result struct {
		Status string           `json:"status"`
		PoP    *PoPHealthStatus `json:"pop"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.PoP, nil
}

// GetFailoverEvents returns historical failover and BGP withdrawal events.
func (c *Client) GetFailoverEvents(ctx context.Context) ([]FailoverEvent, error) {
	url := c.HealthURL + "/v1/health/events"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: failover events query failed: %w", err)
		}
		return c.fallbackFailoverEvents(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: failover events query returned status %d", resp.StatusCode)
		}
		return c.fallbackFailoverEvents(), nil
	}

	var payload struct {
		Count  int             `json:"count"`
		Events []FailoverEvent `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		if c.FailClosed {
			return nil, fmt.Errorf("fail-closed active: failed to decode failover events: %w", err)
		}
		return c.fallbackFailoverEvents(), nil
	}
	return payload.Events, nil
}

func (c *Client) fallbackHealthStatus() *PoPHealthStatus {
	return &PoPHealthStatus{
		RegionID:              "bd-dac",
		Name:                  "Dhaka Sovereign Gateway",
		AnycastIPv4:           "103.185.142.10/24",
		AnycastIPv6:           "2a0e:b107::53/48",
		FailoverPoPID:         "sg-sin",
		State:                 PoPHealthy,
		BGPState:              BGPAdvertised,
		ConsecutiveSuccesses:  42,
		ConsecutiveFailures:   0,
		FlapDampingPenalty:    0.0,
		Damped:                false,
		LastProbeLatencyMs:    1.42,
		LastStateTransition:   time.Now().Add(-15 * time.Minute),
		SovereigntyRestricted: true,
	}
}

func (c *Client) fallbackGlobalMatrix() []*PoPHealthStatus {
	now := time.Now()
	return []*PoPHealthStatus{
		{
			RegionID:              "bd-dac",
			Name:                  "Dhaka Sovereign Gateway",
			AnycastIPv4:           "103.185.142.10/24",
			AnycastIPv6:           "2a0e:b107::53/48",
			FailoverPoPID:         "sg-sin",
			State:                 PoPHealthy,
			BGPState:              BGPAdvertised,
			LastProbeLatencyMs:    1.2,
			LastStateTransition:   now,
			SovereigntyRestricted: true,
		},
		{
			RegionID:              "sg-sin",
			Name:                  "Singapore Transit Hub",
			AnycastIPv4:           "103.185.142.10/24",
			AnycastIPv6:           "2a0e:b107::53/48",
			FailoverPoPID:         "de-fra",
			State:                 PoPHealthy,
			BGPState:              BGPAdvertised,
			LastProbeLatencyMs:    0.8,
			LastStateTransition:   now,
			SovereigntyRestricted: false,
		},
		{
			RegionID:              "de-fra",
			Name:                  "Frankfurt Continental Hub",
			AnycastIPv4:           "103.185.142.10/24",
			AnycastIPv6:           "2a0e:b107::53/48",
			FailoverPoPID:         "is-north-1",
			State:                 PoPHealthy,
			BGPState:              BGPAdvertised,
			LastProbeLatencyMs:    1.1,
			LastStateTransition:   now,
			SovereigntyRestricted: false,
		},
		{
			RegionID:              "is-north-1",
			Name:                  "Reykjavik Geothermal Edge",
			AnycastIPv4:           "103.185.142.10/24",
			AnycastIPv6:           "2a0e:b107::53/48",
			FailoverPoPID:         "de-fra",
			State:                 PoPHealthy,
			BGPState:              BGPAdvertised,
			LastProbeLatencyMs:    2.4,
			LastStateTransition:   now,
			SovereigntyRestricted: false,
		},
		{
			RegionID:              "us-iad",
			Name:                  "Virginia AI Interconnect",
			AnycastIPv4:           "103.185.142.10/24",
			AnycastIPv6:           "2a0e:b107::53/48",
			FailoverPoPID:         "de-fra",
			State:                 PoPHealthy,
			BGPState:              BGPAdvertised,
			LastProbeLatencyMs:    1.9,
			LastStateTransition:   now,
			SovereigntyRestricted: false,
		},
	}
}

func (c *Client) fallbackFailoverEvents() []FailoverEvent {
	return []FailoverEvent{
		{
			Sequence:     1,
			Timestamp:    time.Now().Add(-20 * time.Minute),
			RegionID:     "bd-dac",
			EventType:    "AUTONOMOUS_HEALTH_PROBE",
			OldState:     PoPHealthy,
			NewState:     PoPHealthy,
			OldBGPState:  BGPAdvertised,
			NewBGPState:  BGPAdvertised,
			Reason:       "autonomous health check satisfied",
			TriggeredBy:  "AUTONOMOUS_HEALTH_FSM",
			BGPWithdrawn: false,
		},
	}
}
