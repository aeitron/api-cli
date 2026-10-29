package billing

import (
	"testing"
)

func TestBillingCLICommands_Execution(t *testing.T) {
	client := NewClient()

	// 1. Help
	if err := HandleBillingCommand(client, []string{"help"}); err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	// 2. Plans
	if err := HandleBillingCommand(client, []string{"plans"}); err != nil {
		t.Fatalf("plans command failed: %v", err)
	}

	// 3. Status
	if err := HandleBillingCommand(client, []string{"status", "--tenant", "tenant-test-01"}); err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	// 4. Usage
	if err := HandleBillingCommand(client, []string{"usage", "--tenant", "tenant-test-01"}); err != nil {
		t.Fatalf("usage command failed: %v", err)
	}

	// 5. Invoices
	if err := HandleBillingCommand(client, []string{"invoices", "--tenant", "tenant-test-01"}); err != nil {
		t.Fatalf("invoices command failed: %v", err)
	}

	// 6. Generate Invoice
	if err := HandleBillingCommand(client, []string{"generate", "--tenant", "tenant-test-01"}); err != nil {
		t.Fatalf("generate command failed: %v", err)
	}

	// 7. Pay via bKash
	if err := HandleBillingCommand(client, []string{"pay", "INV-202610-001001", "--gateway", "bkash", "--mfs", "01712345678"}); err != nil {
		t.Fatalf("pay command failed: %v", err)
	}

	// 8. Tokens balance & ledger
	if err := HandleBillingCommand(client, []string{"tokens", "--tenant", "tenant-test-01"}); err != nil {
		t.Fatalf("tokens command failed: %v", err)
	}

	// 9. Reserve tokens (Phase 1)
	if err := HandleBillingCommand(client, []string{"reserve", "50000", "GPU_VLLM_INFERENCE", "--tenant", "tenant-test-01"}); err != nil {
		t.Fatalf("reserve command failed: %v", err)
	}

	// 10. Arbitrage cost savings
	if err := HandleBillingCommand(client, []string{"arbitrage", "--tenant", "tenant-test-01"}); err != nil {
		t.Fatalf("arbitrage command failed: %v", err)
	}

	// 11. Mushak-6.3 tax challan
	if err := HandleBillingCommand(client, []string{"mushak", "INV-202610-001001", "--bin", "001234567-0101"}); err != nil {
		t.Fatalf("mushak command failed: %v", err)
	}

	// 12. Unknown command error
	if err := HandleBillingCommand(client, []string{"unknown-cmd"}); err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
}
