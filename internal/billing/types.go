package billing

import "time"

// Subscription Tiers
const (
	TierStartup    = "STARTUP"
	TierBusiness   = "BUSINESS"
	TierEnterprise = "ENTERPRISE"
)

// BillingAccount represents account summary.
type BillingAccount struct {
	AccountID             string    `json:"account_id"`
	TenantID              string    `json:"tenant_id"`
	OrganizationName      string    `json:"organization_name"`
	BillingEmail          string    `json:"billing_email"`
	Tier                  string    `json:"tier"`
	Status                string    `json:"status"`
	Currency              string    `json:"currency"`
	BalanceUSD            float64   `json:"balance_usd"`
	TokenBalance          int64     `json:"token_balance"`
	SovereignJurisdiction string    `json:"sovereign_jurisdiction"`
	BillingPeriodStart    time.Time `json:"billing_period_start"`
	BillingPeriodEnd      time.Time `json:"billing_period_end"`
}

// TierPlan defines quotas.
type TierPlan struct {
	TierName                  string  `json:"tier_name"`
	BasePriceUSD              float64 `json:"base_price_usd"`
	IncludedEgressGB          float64 `json:"included_egress_gb"`
	IncludedRequests          int64   `json:"included_requests"`
	IncludedMicroVMVCPUHours  float64 `json:"included_microvm_vcpu_hours"`
	IncludedVLLMTokens        int64   `json:"included_vllm_tokens"`
	IncludedStorageGB         float64 `json:"included_storage_gb"`
	OverageEgressPerGB        float64 `json:"overage_egress_per_gb"`
	OverageRequestPer1M       float64 `json:"overage_request_per_1m"`
	OverageMicroVMPerVCPUHour float64 `json:"overage_microvm_per_vcpu_hour"`
	OverageTokenPer1K         float64 `json:"overage_token_per_1k"`
	OverageStoragePerGBMonth  float64 `json:"overage_storage_per_gb_month"`
	SLAAvailabilityPercent    float64 `json:"sla_availability_percent"`
	DedicatedAnycastVIP       bool    `json:"dedicated_anycast_vip"`
	NDMASovereignCertified    bool    `json:"ndma_sovereign_certified"`
}

// TenantUsageSummary aggregates usage.
type TenantUsageSummary struct {
	TenantID           string    `json:"tenant_id"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
	EgressBytes        int64     `json:"egress_bytes"`
	EgressGB           float64   `json:"egress_gb"`
	RequestsTotal      int64     `json:"requests_total"`
	MicroVMVCPUSeconds float64   `json:"microvm_vcpu_seconds"`
	MicroVMVCPUHours   float64   `json:"microvm_vcpu_hours"`
	VLLMTokensTotal    int64     `json:"vllm_tokens_total"`
	StorageGBHours     float64   `json:"storage_gb_hours"`
	EventCount         int64     `json:"event_count"`
	LastEventTimestamp time.Time `json:"last_event_timestamp"`
}

// OverageCalculation detail.
type OverageCalculation struct {
	BaseFeeUSD         float64 `json:"base_fee_usd"`
	EgressOverageUSD   float64 `json:"egress_overage_usd"`
	RequestsOverageUSD float64 `json:"requests_overage_usd"`
	MicroVMOverageUSD  float64 `json:"microvm_overage_usd"`
	VLLMOverageUSD     float64 `json:"vllm_overage_usd"`
	StorageOverageUSD  float64 `json:"storage_overage_usd"`
	TotalOverageUSD    float64 `json:"total_overage_usd"`
	SubtotalUSD        float64 `json:"subtotal_usd"`
}

// UsageReport response.
type UsageReport struct {
	TenantID         string             `json:"tenant_id"`
	Tier             string             `json:"tier"`
	Plan             TierPlan           `json:"plan"`
	UsageSummary     TenantUsageSummary `json:"usage_summary"`
	Overage          OverageCalculation `json:"overage"`
	QuotaUtilization map[string]float64 `json:"quota_utilization"`
}

// InvoiceLineItem representation.
type InvoiceLineItem struct {
	ItemCode      string  `json:"item_code"`
	Description   string  `json:"description"`
	Quantity      float64 `json:"quantity"`
	Unit          string  `json:"unit"`
	UnitPriceUSD  float64 `json:"unit_price_usd"`
	TotalPriceUSD float64 `json:"total_price_usd"`
}

// Invoice representation.
type Invoice struct {
	InvoiceID             string            `json:"invoice_id"`
	AccountID             string            `json:"account_id"`
	TenantID              string            `json:"tenant_id"`
	BillingEmail          string            `json:"billing_email"`
	PeriodStart           time.Time         `json:"period_start"`
	PeriodEnd             time.Time         `json:"period_end"`
	BaseAmountUSD         float64           `json:"base_amount_usd"`
	OverageAmountUSD      float64           `json:"overage_amount_usd"`
	SubtotalUSD           float64           `json:"subtotal_usd"`
	TaxRatePercent        float64           `json:"tax_rate_percent"`
	TaxAmountUSD          float64           `json:"tax_amount_usd"`
	TotalAmountUSD        float64           `json:"total_amount_usd"`
	Currency              string            `json:"currency"`
	Status                string            `json:"status"`
	LineItems             []InvoiceLineItem `json:"line_items"`
	SovereignJurisdiction string            `json:"sovereign_jurisdiction"`
	TaxClassification     string            `json:"tax_classification"`
	CryptographicSealHash string            `json:"cryptographic_seal_hash"`
	CreatedAt             time.Time         `json:"created_at"`
	PaidAt                *time.Time        `json:"paid_at,omitempty"`
}

// PaymentTransaction representation.
type PaymentTransaction struct {
	TransactionID    string    `json:"transaction_id"`
	InvoiceID        string    `json:"invoice_id"`
	TenantID         string    `json:"tenant_id"`
	AmountUSD        float64   `json:"amount_usd"`
	Currency         string    `json:"currency"`
	Gateway          string    `json:"gateway"`
	GatewayReference string    `json:"gateway_reference"`
	Status           string    `json:"status"`
	Message          string    `json:"message"`
	Timestamp        time.Time `json:"timestamp"`
}

// TokenLedgerRecord maps to token_ledgers table.
type TokenLedgerRecord struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	TokenBalance    int64     `json:"token_balance"`
	ReservedTokens  int64     `json:"reserved_tokens"`
	AvailableTokens int64     `json:"available_tokens"`
	Currency        string    `json:"currency"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TokenReservation represents a 2-phase hold.
type TokenReservation struct {
	ReservationID string    `json:"reservation_id"`
	TenantID      string    `json:"tenant_id"`
	Amount        int64     `json:"amount"`
	Purpose       string    `json:"purpose"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// SpotArbitrageRecord maps to spot_arbitrage_records table.
type SpotArbitrageRecord struct {
	RecordID             string    `json:"record_id"`
	WorkloadID           string    `json:"workload_id"`
	TenantID             string    `json:"tenant_id"`
	ProviderFrom         string    `json:"provider_from"`
	ProviderTo           string    `json:"provider_to"`
	CostBeforeUSDHr      float64   `json:"cost_before_usd_hr"`
	CostAfterUSDHr       float64   `json:"cost_after_usd_hr"`
	SavingsPercent       float64   `json:"savings_percent"`
	CustomerSavingsUSDHr float64   `json:"customer_savings_usd_hr"`
	PlatformRevenueUSDHr float64   `json:"platform_revenue_usd_hr"`
	Timestamp            time.Time `json:"timestamp"`
}

// ArbitrageSavingsSummary provides aggregate ROI metrics.
type ArbitrageSavingsSummary struct {
	TenantID                  string  `json:"tenant_id"`
	TotalWorkloadsOptimized   int     `json:"total_workloads_optimized"`
	TotalHourlySavingsUSD     float64 `json:"total_hourly_savings_usd"`
	TotalAnnualizedSavingsUSD float64 `json:"total_annualized_savings_usd"`
	TotalPlatformRevenueUSD   float64 `json:"total_platform_revenue_usd"`
	AverageSavingsPercent     float64 `json:"average_savings_percent"`
}

// Mushak63Invoice represents statutory NBR Mushak-6.3 challan.
type Mushak63Invoice struct {
	ChallanNumber        string           `json:"challan_number"`
	InvoiceID            string           `json:"invoice_id"`
	SupplierBIN          string           `json:"supplier_bin"`
	SupplierName         string           `json:"supplier_name"`
	CustomerBIN          string           `json:"customer_bin"`
	CustomerName         string           `json:"customer_name"`
	DeliveryMethod       string           `json:"delivery_method"`
	IssueDate            time.Time        `json:"issue_date"`
	LineItems            []MushakLineItem `json:"line_items"`
	TotalExcludingVATBDT float64          `json:"total_excluding_vat_bdt"`
	VATRatePercent       float64          `json:"vat_rate_percent"`
	TotalVATAmountBDT    float64          `json:"total_vat_amount_bdt"`
	TotalIncludingVATBDT float64          `json:"total_including_vat_bdt"`
	NBRDigitalSignature  string           `json:"nbr_digital_signature"`
}

// MushakLineItem line item representation.
type MushakLineItem struct {
	SerialNumber             int     `json:"serial_number"`
	GoodOrServiceDescription string  `json:"description"`
	Quantity                 float64 `json:"quantity"`
	UnitOfMeasure            string  `json:"unit_of_measure"`
	UnitPriceBDT             float64 `json:"unit_price_bdt"`
	TotalValueBDT            float64 `json:"total_value_bdt"`
	VATAmountBDT             float64 `json:"vat_amount_bdt"`
	TotalWithVATBDT          float64 `json:"total_with_vat_bdt"`
}
