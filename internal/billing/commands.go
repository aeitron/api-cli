package billing

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// HandleBillingCommand dispatches billing subcommands.
func HandleBillingCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintBillingHelp()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	switch args[0] {
	case "help", "-h", "--help":
		PrintBillingHelp()
		return nil

	case "plans", "pricing":
		return handlePlans(ctx, client)

	case "status", "account":
		tenantID := resolveTenantID(args[1:])
		return handleStatus(ctx, client, tenantID)

	case "usage":
		tenantID := resolveTenantID(args[1:])
		return handleUsage(ctx, client, tenantID)

	case "invoices", "list-invoices":
		tenantID := resolveTenantID(args[1:])
		return handleInvoices(ctx, client, tenantID)

	case "generate", "create-invoice":
		tenantID := resolveTenantID(args[1:])
		return handleGenerate(ctx, client, tenantID)

	case "pay", "checkout":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron billing pay <invoice-id> [--gateway stripe|bkash|sslcommerz|sepa] [--mfs <01XXXXXXXXX>]")
		}
		invoiceID := args[1]
		gateway := "STRIPE_CARD"
		mfsNumber := ""
		for i := 2; i < len(args); i++ {
			if args[i] == "--gateway" && i+1 < len(args) {
				g := strings.ToUpper(args[i+1])
				switch g {
				case "STRIPE", "CARD":
					gateway = "STRIPE_CARD"
				case "BKASH", "MFS":
					gateway = "BKASH_MFS"
				case "SSLCOMMERZ", "SSL":
					gateway = "SSLCOMMERZ"
				case "SEPA", "WIRE":
					gateway = "SEPA_WIRE"
				default:
					gateway = g
				}
				i++
			} else if (args[i] == "--mfs" || args[i] == "--phone") && i+1 < len(args) {
				mfsNumber = args[i+1]
				i++
			}
		}
		return handlePay(ctx, client, invoiceID, gateway, mfsNumber)

	case "tokens", "token-ledger":
		tenantID := resolveTenantID(args[1:])
		return handleTokens(ctx, client, tenantID)

	case "reserve", "reserve-tokens":
		if len(args) < 3 {
			return fmt.Errorf("usage: aeitron billing reserve <amount> <purpose> [--tenant <id>]")
		}
		var amount int64
		_, err := fmt.Sscanf(args[1], "%d", &amount)
		if err != nil || amount <= 0 {
			return fmt.Errorf("invalid token amount: %s", args[1])
		}
		purpose := args[2]
		tenantID := resolveTenantID(args[3:])
		return handleReserveTokens(ctx, client, tenantID, amount, purpose)

	case "arbitrage", "spot-savings":
		tenantID := resolveTenantID(args[1:])
		return handleArbitrage(ctx, client, tenantID)

	case "mushak", "mushak-6.3":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron billing mushak <invoice-id> [--bin <customer-bin>]")
		}
		invoiceID := args[1]
		customerBIN := ""
		for i := 2; i < len(args); i++ {
			if args[i] == "--bin" && i+1 < len(args) {
				customerBIN = args[i+1]
			}
		}
		return handleMushak(ctx, client, invoiceID, customerBIN)

	default:
		return fmt.Errorf("unknown billing subcommand: '%s'. Run 'aeitron billing help' for usage", args[0])
	}
}

func handlePlans(ctx context.Context, client *Client) error {
	plans, err := client.GetPlans(ctx)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Commercial Subscription Plans ($1M ARR Model)")
	fmt.Println("================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIER\tBASE PRICE\tEGRESS\tREQUESTS\tMICROVM\tVLLM TOKENS\tSLA\tVIP/NDMA")
	for _, p := range plans {
		fmt.Fprintf(w, "%s\t$%.2f/mo\t%.0f GB\t%dM\t%.0f hrs\t%dk\t%.3f%%\t%v\n",
			p.TierName,
			p.BasePriceUSD,
			p.IncludedEgressGB,
			p.IncludedRequests/1000000,
			p.IncludedMicroVMVCPUHours,
			p.IncludedVLLMTokens/1000,
			p.SLAAvailabilityPercent,
			p.NDMASovereignCertified,
		)
	}
	w.Flush()
	fmt.Println()
	return nil
}

func handleStatus(ctx context.Context, client *Client, tenantID string) error {
	acc, err := client.GetAccount(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Aeitron Billing Account Status: %s (%s)\n", acc.OrganizationName, acc.TenantID)
	fmt.Println("================================================================================")
	fmt.Printf("  Account ID:            %s\n", acc.AccountID)
	fmt.Printf("  Subscription Tier:     %s\n", acc.Tier)
	fmt.Printf("  Account Status:        %s\n", acc.Status)
	fmt.Printf("  Billing Email:         %s\n", acc.BillingEmail)
	fmt.Printf("  Sovereign Territory:   %s\n", acc.SovereignJurisdiction)
	fmt.Printf("  Token Ledger Balance:  %d TOKENS\n", acc.TokenBalance)
	fmt.Printf("  Cycle Period:          %s to %s\n\n",
		acc.BillingPeriodStart.Format("2006-01-02"),
		acc.BillingPeriodEnd.Format("2006-01-02"),
	)
	return nil
}

func handleUsage(ctx context.Context, client *Client, tenantID string) error {
	report, err := client.GetUsageReport(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Aeitron Metered Resource Consumption: %s [Tier: %s]\n", report.TenantID, report.Tier)
	fmt.Println("================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "METRIC DIMENSION\tCONSUMED\tINCLUDED IN PLAN\tUTILIZATION\tOVERAGE COST")
	fmt.Fprintf(w, "Bandwidth Egress\t%.2f GB\t%.0f GB\t%.1f%%\t$%.2f\n",
		report.UsageSummary.EgressGB,
		report.Plan.IncludedEgressGB,
		report.QuotaUtilization["egress_bytes"],
		report.Overage.EgressOverageUSD,
	)
	fmt.Fprintf(w, "HTTP/DNS Requests\t%d\t%d\t%.1f%%\t$%.2f\n",
		report.UsageSummary.RequestsTotal,
		report.Plan.IncludedRequests,
		report.QuotaUtilization["requests_total"],
		report.Overage.RequestsOverageUSD,
	)
	fmt.Fprintf(w, "MicroVM vCPU Execution\t%.2f hrs\t%.0f hrs\t%.1f%%\t$%.2f\n",
		report.UsageSummary.MicroVMVCPUHours,
		report.Plan.IncludedMicroVMVCPUHours,
		report.QuotaUtilization["microvm_vcpu_sec"],
		report.Overage.MicroVMOverageUSD,
	)
	fmt.Fprintf(w, "vLLM Inference Tokens\t%d\t%d\t%.1f%%\t$%.2f\n",
		report.UsageSummary.VLLMTokensTotal,
		report.Plan.IncludedVLLMTokens,
		report.QuotaUtilization["vllm_tokens_total"],
		report.Overage.VLLMOverageUSD,
	)
	w.Flush()

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("  Base Subscription:     $%.2f\n", report.Overage.BaseFeeUSD)
	fmt.Printf("  Total Overages:        $%.2f\n", report.Overage.TotalOverageUSD)
	fmt.Printf("  Current Subtotal:      $%.2f (excluding statutory VAT/Tax)\n\n", report.Overage.SubtotalUSD)
	return nil
}

func handleInvoices(ctx context.Context, client *Client, tenantID string) error {
	invoices, err := client.ListInvoices(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Aeitron Invoices Ledger (%d Invoices Found)\n", len(invoices))
	fmt.Println("================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INVOICE ID\tTENANT\tPERIOD\tSUBTOTAL\tTAX (VAT)\tTOTAL DUE\tSTATUS\tSEAL HASH")
	for _, inv := range invoices {
		shortSeal := inv.CryptographicSealHash
		if len(shortSeal) > 12 {
			shortSeal = shortSeal[:12] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t$%.2f (%.0f%%)\t$%.2f\t%s\t%s\n",
			inv.InvoiceID,
			inv.TenantID,
			inv.PeriodEnd.Format("2006-01-02"),
			inv.SubtotalUSD,
			inv.TaxAmountUSD,
			inv.TaxRatePercent,
			inv.TotalAmountUSD,
			inv.Status,
			shortSeal,
		)
	}
	w.Flush()
	fmt.Println()
	return nil
}

func handleGenerate(ctx context.Context, client *Client, tenantID string) error {
	inv, err := client.GenerateInvoice(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Generated New Invoice: %s (Status: %s)\n", inv.InvoiceID, inv.Status)
	fmt.Println("================================================================================")
	fmt.Printf("  Tenant ID:             %s\n", inv.TenantID)
	fmt.Printf("  Billing Period:        %s to %s\n", inv.PeriodStart.Format("2006-01-02"), inv.PeriodEnd.Format("2006-01-02"))
	fmt.Printf("  Base Subscription:     $%.2f\n", inv.BaseAmountUSD)
	fmt.Printf("  Metered Overages:      $%.2f\n", inv.OverageAmountUSD)
	fmt.Printf("  Subtotal:              $%.2f\n", inv.SubtotalUSD)
	fmt.Printf("  Statutory Tax (%s): $%.2f (%.0f%%)\n", inv.TaxClassification, inv.TaxAmountUSD, inv.TaxRatePercent)
	fmt.Printf("  Total Amount Due:      $%.2f %s\n", inv.TotalAmountUSD, inv.Currency)
	fmt.Printf("  HMAC-SHA256 Seal:      %s\n\n", inv.CryptographicSealHash)
	return nil
}

func handlePay(ctx context.Context, client *Client, invoiceID, gateway, mfsNumber string) error {
	tx, err := client.ProcessPayment(ctx, invoiceID, gateway, mfsNumber)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Payment Settlement Confirmed")
	fmt.Println("================================================================================")
	fmt.Printf("  Transaction ID:        %s\n", tx.TransactionID)
	fmt.Printf("  Invoice Reference:     %s\n", tx.InvoiceID)
	fmt.Printf("  Settlement Amount:     $%.2f %s\n", tx.AmountUSD, tx.Currency)
	fmt.Printf("  Payment Rail:          %s\n", tx.Gateway)
	fmt.Printf("  Gateway Reference:     %s\n", tx.GatewayReference)
	fmt.Printf("  Settlement Status:     %s\n", tx.Status)
	fmt.Printf("  Message:               %s\n\n", tx.Message)
	return nil
}

func resolveTenantID(args []string) string {
	for i := 0; i < len(args); i++ {
		if (args[i] == "--tenant" || args[i] == "-t") && i+1 < len(args) {
			return args[i+1]
		}
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		return args[0]
	}
	return "tenant-corp-01"
}

// PrintBillingHelp prints CLI usage.
func PrintBillingHelp() {
	fmt.Println("Usage: aeitron billing <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  plans                               List available subscription plans ($1M ARR matrix)")
	fmt.Println("  status [--tenant <id>]              Show tenant billing account status and balance")
	fmt.Println("  usage [--tenant <id>]               Show real-time metered resource usage & quotas")
	fmt.Println("  tokens [--tenant <id>]              View pre-paid token ledger and active reservations")
	fmt.Println("  reserve <amount> <purpose>          Place a 2-phase reservation hold on tokens")
	fmt.Println("  arbitrage [--tenant <id>]           Show spot cloud migration cost savings & ROI")
	fmt.Println("  invoices [--tenant <id>]            List invoices with cryptographic audit seals")
	fmt.Println("  generate [--tenant <id>]            Generate itemized monthly invoice with statutory tax")
	fmt.Println("  mushak <invoice-id> [--bin <bin>]   View statutory Bangladesh NBR Form Mushak-6.3 challan")
	fmt.Println("  pay <invoice-id> [flags]            Settle invoice via global or sovereign payment rail")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --gateway <stripe|bkash|sslcommerz|sepa>  Payment gateway (default: stripe)")
	fmt.Println("  --mfs <01XXXXXXXXX>                      Mobile number for bKash MFS checkout")
	fmt.Println("  --bin <13-digit-bin>                     Customer Business Identification Number")
	fmt.Println("  --tenant <id>                            Target tenant identifier")
	fmt.Println()
}

func handleTokens(ctx context.Context, client *Client, tenantID string) error {
	ledger, resvs, err := client.GetTokens(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Aeitron Token Ledger & Balance ===\n")
	fmt.Printf("  Ledger Record ID:      %s\n", ledger.ID)
	fmt.Printf("  Tenant ID:             %s\n", ledger.TenantID)
	fmt.Printf("  Total Token Balance:   %d %s\n", ledger.TokenBalance, ledger.Currency)
	fmt.Printf("  Reserved / Held:       %d %s\n", ledger.ReservedTokens, ledger.Currency)
	fmt.Printf("  Available for Compute: %d %s\n", ledger.AvailableTokens, ledger.Currency)
	fmt.Printf("  Last Updated:          %s\n", ledger.UpdatedAt.Format(time.RFC3339))

	if len(resvs) > 0 {
		fmt.Printf("\nActive 2-Phase Reservations (%d):\n", len(resvs))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RESERVATION ID\tAMOUNT\tPURPOSE\tSTATUS\tEXPIRES AT")
		for _, r := range resvs {
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n",
				r.ReservationID, r.Amount, r.Purpose, r.Status, r.ExpiresAt.Format("15:04:05 UTC"))
		}
		w.Flush()
	}
	fmt.Println()
	return nil
}

func handleReserveTokens(ctx context.Context, client *Client, tenantID string, amount int64, purpose string) error {
	res, err := client.ReserveTokens(ctx, tenantID, amount, purpose)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Token Reservation Placed (Phase 1) ===\n")
	fmt.Printf("  Reservation ID:   %s\n", res.ReservationID)
	fmt.Printf("  Tenant ID:        %s\n", res.TenantID)
	fmt.Printf("  Reserved Amount:  %d TOKENS\n", res.Amount)
	fmt.Printf("  Target Purpose:   %s\n", res.Purpose)
	fmt.Printf("  Status:           %s\n", res.Status)
	fmt.Printf("  Expires At:       %s\n\n", res.ExpiresAt.Format(time.RFC3339))
	return nil
}

func handleArbitrage(ctx context.Context, client *Client, tenantID string) error {
	summary, err := client.GetArbitrageSummary(ctx, tenantID)
	if err != nil {
		return err
	}
	records, err := client.ListArbitrage(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Spot Cloud Arbitrage & Value-Capture ===\n")
	fmt.Printf("  Workloads Optimized:      %d\n", summary.TotalWorkloadsOptimized)
	fmt.Printf("  Average Cost Reduction:   %.2f%%\n", summary.AverageSavingsPercent)
	fmt.Printf("  Customer Hourly Savings:  $%.4f / hr\n", summary.TotalHourlySavingsUSD)
	fmt.Printf("  Annualized ROI Savings:   $%.2f / yr\n", summary.TotalAnnualizedSavingsUSD)
	fmt.Printf("  Platform Value Capture:   $%.4f / hr (20%% recurring fee)\n", summary.TotalPlatformRevenueUSD)

	if len(records) > 0 {
		fmt.Printf("\nWorkload Migration & Routing Arbitrage Records:\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "WORKLOAD\tFROM PROVIDER\tTO AEITRON\tBEFORE\tAFTER\tSAVINGS\tCUSTOMER/HR")
		for _, r := range records {
			fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t$%.2f\t%.1f%%\t+$%.2f/hr\n",
				r.WorkloadID, r.ProviderFrom, r.ProviderTo,
				r.CostBeforeUSDHr, r.CostAfterUSDHr, r.SavingsPercent, r.CustomerSavingsUSDHr)
		}
		w.Flush()
	}
	fmt.Println()
	return nil
}

func handleMushak(ctx context.Context, client *Client, invoiceID, customerBIN string) error {
	mushak, err := client.GetMushak63(ctx, invoiceID, customerBIN, "Dhaka, Bangladesh")
	if err != nil {
		return err
	}

	fmt.Printf("\n====================================================================\n")
	fmt.Printf(" গণপ্রজাতন্ত্রী বাংলাদেশ সরকার, জাতীয় রাজস্ব বোর্ড\n")
	fmt.Printf(" কর চালানপত্র (মূসক-৬.৩) [NBR Statutory Tax Invoice]\n")
	fmt.Printf("====================================================================\n")
	fmt.Printf("  চালান নং (Challan No):  %s\n", mushak.ChallanNumber)
	fmt.Printf("  মূল ইনভয়েস আইডি:       %s\n", mushak.InvoiceID)
	fmt.Printf("  ইস্যুর তারিখ:           %s\n", mushak.IssueDate.Format("2006-01-02 15:04:05 UTC"))
	fmt.Printf("  সরবরাহকারী বিআইএন:      %s (%s)\n", mushak.SupplierBIN, mushak.SupplierName)
	fmt.Printf("  ক্রেতা বিআইএন:          %s (%s)\n", mushak.CustomerBIN, mushak.CustomerName)
	fmt.Printf("  সরবরাহের মাধ্যম:        %s\n", mushak.DeliveryMethod)
	fmt.Printf("--------------------------------------------------------------------\n")
	fmt.Printf("  মোট মূল্য (মূসক ব্যতীত): ৳%.2f BDT\n", mushak.TotalExcludingVATBDT)
	fmt.Printf("  মূসক হার (NBR Cloud VAT):%.2f%%\n", mushak.VATRatePercent)
	fmt.Printf("  মোট মূসক পরিমাণ:       ৳%.2f BDT\n", mushak.TotalVATAmountBDT)
	fmt.Printf("  সর্বমোট প্রদেয় মূল্য:   ৳%.2f BDT\n", mushak.TotalIncludingVATBDT)
	fmt.Printf("--------------------------------------------------------------------\n")
	fmt.Printf("  NBR HMAC-SHA256 Seal:   %s\n\n", mushak.NBRDigitalSignature)
	return nil
}
