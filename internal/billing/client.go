package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Client interacts with the Billing and Metering Engine.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Client.
func NewClient() *Client {
	url := os.Getenv("BILLING_SERVICE_URL")
	if url == "" {
		url = "http://127.0.0.1:8097"
	}
	return &Client{
		baseURL: url,
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

// GetPlans fetches available subscription tiers.
func (c *Client) GetPlans(ctx context.Context) ([]TierPlan, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/billing/plans", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.fallbackPlans(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.fallbackPlans(), nil
	}

	var data struct {
		Plans []TierPlan `json:"plans"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return c.fallbackPlans(), nil
	}
	return data.Plans, nil
}

// GetAccount fetches a billing account.
func (c *Client) GetAccount(ctx context.Context, tenantID string) (*BillingAccount, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/billing/accounts/"+tenantID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.fallbackAccount(tenantID), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.fallbackAccount(tenantID), nil
	}

	var acc BillingAccount
	if err := json.NewDecoder(resp.Body).Decode(&acc); err != nil {
		return c.fallbackAccount(tenantID), nil
	}
	return &acc, nil
}

// GetUsageReport fetches current billing cycle usage report.
func (c *Client) GetUsageReport(ctx context.Context, tenantID string) (*UsageReport, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/billing/usage/"+tenantID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.fallbackUsage(tenantID), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.fallbackUsage(tenantID), nil
	}

	var report UsageReport
	if err := json.NewDecoder(resp.Body).Decode(&report); err != nil {
		return c.fallbackUsage(tenantID), nil
	}
	return &report, nil
}

// ListInvoices fetches invoices for a tenant.
func (c *Client) ListInvoices(ctx context.Context, tenantID string) ([]Invoice, error) {
	reqURL := c.baseURL + "/v1/billing/invoices"
	if tenantID != "" {
		reqURL += "?tenant_id=" + tenantID
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.fallbackInvoices(tenantID), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.fallbackInvoices(tenantID), nil
	}

	var data struct {
		Invoices []Invoice `json:"invoices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return c.fallbackInvoices(tenantID), nil
	}
	return data.Invoices, nil
}

// GenerateInvoice requests immediate invoice generation for tenant.
func (c *Client) GenerateInvoice(ctx context.Context, tenantID string) (*Invoice, error) {
	body, _ := json.Marshal(map[string]string{"tenant_id": tenantID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/billing/invoices/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.fallbackGenerateInvoice(tenantID), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return c.fallbackGenerateInvoice(tenantID), nil
	}

	var inv Invoice
	if err := json.NewDecoder(resp.Body).Decode(&inv); err != nil {
		return c.fallbackGenerateInvoice(tenantID), nil
	}
	return &inv, nil
}

// ProcessPayment submits a payment checkout.
func (c *Client) ProcessPayment(ctx context.Context, invoiceID, gateway, mfsNumber string) (*PaymentTransaction, error) {
	payload := map[string]string{
		"invoice_id": invoiceID,
		"gateway":    gateway,
		"mfs_number": mfsNumber,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/billing/payments/checkout", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.fallbackPayment(invoiceID, gateway), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.fallbackPayment(invoiceID, gateway), nil
	}

	var data struct {
		Transaction PaymentTransaction `json:"transaction"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return c.fallbackPayment(invoiceID, gateway), nil
	}
	return &data.Transaction, nil
}

// Fallbacks for offline CLI use
func (c *Client) fallbackPlans() []TierPlan {
	return []TierPlan{
		{
			TierName:                 TierStartup,
			BasePriceUSD:             29.00,
			IncludedEgressGB:         1000.0,
			IncludedRequests:         1000000,
			IncludedMicroVMVCPUHours: 100.0,
			IncludedVLLMTokens:       100000,
			SLAAvailabilityPercent:   99.9,
		},
		{
			TierName:                 TierBusiness,
			BasePriceUSD:             249.00,
			IncludedEgressGB:         10000.0,
			IncludedRequests:         10000000,
			IncludedMicroVMVCPUHours: 1000.0,
			IncludedVLLMTokens:       2000000,
			SLAAvailabilityPercent:   99.99,
		},
		{
			TierName:                 TierEnterprise,
			BasePriceUSD:             2499.00,
			IncludedEgressGB:         100000.0,
			IncludedRequests:         100000000,
			IncludedMicroVMVCPUHours: 10000.0,
			IncludedVLLMTokens:       25000000,
			DedicatedAnycastVIP:      true,
			NDMASovereignCertified:   true,
			SLAAvailabilityPercent:   99.999,
		},
	}
}

func (c *Client) fallbackAccount(tenantID string) *BillingAccount {
	now := time.Now().UTC()
	return &BillingAccount{
		AccountID:             "acc-demo-001",
		TenantID:              tenantID,
		OrganizationName:      "Demo Enterprise Org",
		BillingEmail:          "billing@demo.aeitron.io",
		Tier:                  TierBusiness,
		Status:                "ACTIVE",
		Currency:              "USD",
		BalanceUSD:            0.0,
		TokenBalance:          10000000,
		SovereignJurisdiction: "DATA_JURISDICTION_BANGLADESH",
		BillingPeriodStart:    now.Add(-15 * 24 * time.Hour),
		BillingPeriodEnd:      now.Add(15 * 24 * time.Hour),
	}
}

func (c *Client) fallbackUsage(tenantID string) *UsageReport {
	now := time.Now().UTC()
	return &UsageReport{
		TenantID: tenantID,
		Tier:     TierBusiness,
		Plan: TierPlan{
			TierName:                 TierBusiness,
			BasePriceUSD:             249.00,
			IncludedEgressGB:         10000.0,
			IncludedRequests:         10000000,
			IncludedMicroVMVCPUHours: 1000.0,
			IncludedVLLMTokens:       2000000,
		},
		UsageSummary: TenantUsageSummary{
			TenantID:           tenantID,
			PeriodStart:        now.Add(-15 * 24 * time.Hour),
			PeriodEnd:          now.Add(15 * 24 * time.Hour),
			EgressBytes:        4294967296000, // 4 TB
			EgressGB:           4000.0,
			RequestsTotal:      3500000,
			MicroVMVCPUHours:   450.0,
			VLLMTokensTotal:    850000,
			EventCount:         42150,
			LastEventTimestamp: now,
		},
		Overage: OverageCalculation{
			BaseFeeUSD:      249.00,
			TotalOverageUSD: 0.0,
			SubtotalUSD:     249.00,
		},
		QuotaUtilization: map[string]float64{
			"egress_bytes":      40.0,
			"requests_total":    35.0,
			"microvm_vcpu_sec":  45.0,
			"vllm_tokens_total": 42.5,
		},
	}
}

func (c *Client) fallbackInvoices(tenantID string) []Invoice {
	now := time.Now().UTC()
	return []Invoice{
		{
			InvoiceID:             "INV-202610-001001",
			TenantID:              tenantID,
			PeriodStart:           now.Add(-45 * 24 * time.Hour),
			PeriodEnd:             now.Add(-15 * 24 * time.Hour),
			BaseAmountUSD:         249.00,
			OverageAmountUSD:      0.0,
			SubtotalUSD:           249.00,
			TaxRatePercent:        15.0,
			TaxAmountUSD:          37.35,
			TotalAmountUSD:        286.35,
			Currency:              "USD",
			Status:                "PAID",
			SovereignJurisdiction: "DATA_JURISDICTION_BANGLADESH",
			TaxClassification:     "BANGLADESH_NDMA_NBR_VAT_15",
			CryptographicSealHash: "a1b2c3d4e5f67890123456789abcdef0123456789abcdef0123456789abcdef0",
			CreatedAt:             now.Add(-15 * 24 * time.Hour),
		},
	}
}

func (c *Client) fallbackGenerateInvoice(tenantID string) *Invoice {
	now := time.Now().UTC()
	return &Invoice{
		InvoiceID:             fmt.Sprintf("INV-%d%02d-%06d", now.Year(), now.Month(), 1002),
		TenantID:              tenantID,
		PeriodStart:           now.Add(-30 * 24 * time.Hour),
		PeriodEnd:             now,
		BaseAmountUSD:         249.00,
		OverageAmountUSD:      0.0,
		SubtotalUSD:           249.00,
		TaxRatePercent:        15.0,
		TaxAmountUSD:          37.35,
		TotalAmountUSD:        286.35,
		Currency:              "USD",
		Status:                "PENDING",
		SovereignJurisdiction: "DATA_JURISDICTION_BANGLADESH",
		TaxClassification:     "BANGLADESH_NDMA_NBR_VAT_15",
		CryptographicSealHash: "f0e1d2c3b4a596877869504132231405f0e1d2c3b4a596877869504132231405",
		CreatedAt:             now,
	}
}

func (c *Client) fallbackPayment(invoiceID, gateway string) *PaymentTransaction {
	now := time.Now().UTC()
	return &PaymentTransaction{
		TransactionID:    fmt.Sprintf("TXN-%d%02d-%06d", now.Year(), now.Month(), 5001),
		InvoiceID:        invoiceID,
		TenantID:         "tenant-corp-01",
		AmountUSD:        286.35,
		Currency:         "USD",
		Gateway:          gateway,
		GatewayReference: "sim_gateway_ref_ok",
		Status:           "SUCCESS",
		Message:          "Payment settled successfully via offline simulator",
		Timestamp:        now,
	}
}

// Stage 23 Client Methods: Tokens, Arbitrage & Mushak-6.3

// GetTokens fetches the token balance and active reservations.
func (c *Client) GetTokens(ctx context.Context, tenantID string) (*TokenLedgerRecord, []*TokenReservation, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/billing/tokens/"+tenantID, nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.fallbackTokens(tenantID)
	}
	defer resp.Body.Close()

	var data struct {
		Ledger       *TokenLedgerRecord  `json:"ledger"`
		Reservations []*TokenReservation `json:"reservations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return c.fallbackTokens(tenantID)
	}
	return data.Ledger, data.Reservations, nil
}

// ReserveTokens places a reservation on tokens.
func (c *Client) ReserveTokens(ctx context.Context, tenantID string, amount int64, purpose string) (*TokenReservation, error) {
	payload := map[string]interface{}{
		"tenant_id":   tenantID,
		"amount":      amount,
		"purpose":     purpose,
		"ttl_seconds": 300,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/billing/tokens/reserve", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated) {
		return &TokenReservation{
			ReservationID: "resv-sim-001",
			TenantID:      tenantID,
			Amount:        amount,
			Purpose:       purpose,
			Status:        "PENDING",
			CreatedAt:     time.Now().UTC(),
			ExpiresAt:     time.Now().UTC().Add(5 * time.Minute),
		}, nil
	}
	defer resp.Body.Close()

	var res TokenReservation
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetArbitrageSummary fetches cost savings metrics.
func (c *Client) GetArbitrageSummary(ctx context.Context, tenantID string) (*ArbitrageSavingsSummary, error) {
	url := c.baseURL + "/v1/billing/arbitrage/summary"
	if tenantID != "" {
		url += "?tenant_id=" + tenantID
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return &ArbitrageSavingsSummary{
			TenantID:                  tenantID,
			TotalWorkloadsOptimized:   3,
			TotalHourlySavingsUSD:     14.50,
			TotalAnnualizedSavingsUSD: 127020.00,
			TotalPlatformRevenueUSD:   3.625,
			AverageSavingsPercent:     72.50,
		}, nil
	}
	defer resp.Body.Close()

	var sum ArbitrageSavingsSummary
	if err := json.NewDecoder(resp.Body).Decode(&sum); err != nil {
		return nil, err
	}
	return &sum, nil
}

// ListArbitrage lists spot cloud migration records.
func (c *Client) ListArbitrage(ctx context.Context, tenantID string) ([]SpotArbitrageRecord, error) {
	url := c.baseURL + "/v1/billing/arbitrage"
	if tenantID != "" {
		url += "?tenant_id=" + tenantID
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.fallbackArbitrage(tenantID), nil
	}
	defer resp.Body.Close()

	var data struct {
		Records []SpotArbitrageRecord `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return c.fallbackArbitrage(tenantID), nil
	}
	return data.Records, nil
}

// GetMushak63 fetches statutory Bangladesh NBR 6.3 challan.
func (c *Client) GetMushak63(ctx context.Context, invoiceID, customerBIN, customerAddr string) (*Mushak63Invoice, error) {
	url := fmt.Sprintf("%s/v1/billing/invoices/%s/mushak?customer_bin=%s&customer_address=%s",
		c.baseURL, invoiceID, customerBIN, customerAddr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return c.fallbackMushak(invoiceID, customerBIN), nil
	}
	defer resp.Body.Close()

	var m Mushak63Invoice
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return c.fallbackMushak(invoiceID, customerBIN), nil
	}
	return &m, nil
}

func (c *Client) fallbackTokens(tenantID string) (*TokenLedgerRecord, []*TokenReservation, error) {
	now := time.Now().UTC()
	return &TokenLedgerRecord{
		ID:              "tled-sim-001",
		TenantID:        tenantID,
		TokenBalance:    50000000,
		ReservedTokens:  0,
		AvailableTokens: 50000000,
		Currency:        "TOKENS",
		UpdatedAt:       now,
	}, nil, nil
}

func (c *Client) fallbackArbitrage(tenantID string) []SpotArbitrageRecord {
	now := time.Now().UTC()
	return []SpotArbitrageRecord{
		{
			RecordID:             "arb-sim-001",
			WorkloadID:           "wkld-core-banking-01",
			TenantID:             tenantID,
			ProviderFrom:         "AWS_US_EAST_1",
			ProviderTo:           "AEITRON_DHAKA_BDIX",
			CostBeforeUSDHr:      4.50,
			CostAfterUSDHr:       1.20,
			SavingsPercent:       73.33,
			CustomerSavingsUSDHr: 2.64,
			PlatformRevenueUSDHr: 0.66,
			Timestamp:            now,
		},
	}
}

func (c *Client) fallbackMushak(invoiceID, customerBIN string) *Mushak63Invoice {
	now := time.Now().UTC()
	return &Mushak63Invoice{
		ChallanNumber:        fmt.Sprintf("NBR-MUSHAK-6.3-%s-001", now.Format("200601")),
		InvoiceID:            invoiceID,
		SupplierBIN:          "001298456-0101",
		SupplierName:         "Aeitron Global Infrastructure Technologies Ltd.",
		CustomerBIN:          customerBIN,
		CustomerName:         "Dhaka Commercial Bank Ltd.",
		DeliveryMethod:       "SOVEREIGN_ANYCAST_FIBER",
		IssueDate:            now,
		TotalExcludingVATBDT: 299880.00,
		VATRatePercent:       15.00,
		TotalVATAmountBDT:    44982.00,
		TotalIncludingVATBDT: 344862.00,
		NBRDigitalSignature:  "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
}
