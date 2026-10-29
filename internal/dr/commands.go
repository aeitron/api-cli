package dr

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// HandleDRCommand dispatches disaster recovery and tenant isolation subcommands.
func HandleDRCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintDRHelp()
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	switch args[0] {
	case "help", "-h", "--help":
		PrintDRHelp()
		return nil

	case "status":
		tenantID := resolveTenantID(args[1:])
		return handleStatus(ctx, client, tenantID)

	case "policy":
		tenantID := resolveTenantID(args[1:])
		return handlePolicy(ctx, client, tenantID)

	case "failover":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron dr failover <tenant-id> [target-region] [--force]")
		}
		tenantID := args[1]
		targetRegion := ""
		force := false
		for i := 2; i < len(args); i++ {
			if args[i] == "--force" {
				force = true
			} else if targetRegion == "" && !strings.HasPrefix(args[i], "-") {
				targetRegion = args[i]
			}
		}
		return handleFailover(ctx, client, tenantID, targetRegion, force)

	case "failback":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron dr failback <tenant-id>")
		}
		tenantID := args[1]
		return handleFailback(ctx, client, tenantID)

	case "rehearse", "test", "simulate":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron dr rehearse <tenant-id>")
		}
		tenantID := args[1]
		return handleRehearsal(ctx, client, tenantID)

	case "journal", "events", "audit":
		tenantID := resolveTenantID(args[1:])
		return handleJournal(ctx, client, tenantID)

	case "profiles", "isolation":
		return handleProfiles(ctx, client)

	default:
		fmt.Printf("Unknown dr subcommand '%s'. Run 'aeitron dr help' for usage.\n", args[0])
		return nil
	}
}

func resolveTenantID(args []string) string {
	for i := 0; i < len(args); i++ {
		if (args[i] == "--tenant" || args[i] == "-t") && i+1 < len(args) {
			return args[i+1]
		}
	}
	if envTenant := os.Getenv("AEITRON_TENANT_ID"); envTenant != "" {
		return envTenant
	}
	return "tenant-global-ai-02"
}

func handleStatus(ctx context.Context, client *Client, tenantID string) error {
	st, err := client.GetStatus(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Aeitron Multi-Region Disaster Recovery Status: %s ===\n", tenantID)
	fmt.Printf("  Operational State:       %s\n", st.State)
	fmt.Printf("  Active Serving Region:   %s\n", strings.ToUpper(st.CurrentActiveRegion))
	fmt.Printf("  Primary Region:          %s\n", st.PrimaryRegion)
	fmt.Printf("  Secondary Region:        %s\n", st.SecondaryRegion)
	fmt.Printf("  Witness Consensus:       %s\n", st.WitnessConsensus)
	fmt.Printf("  Primary Health Score:    %.2f / 1.00\n", st.PrimaryHealthScore)
	fmt.Printf("  Secondary Health Score:  %.2f / 1.00\n", st.SecondaryHealthScore)
	fmt.Printf("  Estimated RPO:           %.3f seconds (Data Loss Bound)\n", st.CurrentRPOEstimateSeconds)
	fmt.Printf("  Estimated RTO:           %.2f seconds (Failover Speed)\n", st.CurrentRTOEstimateSeconds)
	fmt.Printf("  Sovereignty Preserved:   %t\n", st.IsSovereigntyPreserved)
	if st.LastFailoverAt != nil {
		fmt.Printf("  Last Failover:           %s (%s)\n", st.LastFailoverAt.Format(time.RFC3339), st.LastFailoverReason)
	}
	fmt.Println()
	return nil
}

func handlePolicy(ctx context.Context, client *Client, tenantID string) error {
	pol, err := client.GetPolicy(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Disaster Recovery Policy: %s ===\n", tenantID)
	fmt.Printf("  Primary Region:          %s\n", pol.PrimaryRegion)
	fmt.Printf("  Secondary Standby:       %s\n", pol.SecondaryRegion)
	if pol.TertiaryRegion != "" {
		fmt.Printf("  Tertiary Fallback:       %s\n", pol.TertiaryRegion)
	}
	fmt.Printf("  Failover Topology:       %s\n", pol.FailoverMode)
	fmt.Printf("  Target RPO:              %.3f seconds\n", pol.TargetRPOSeconds)
	fmt.Printf("  Target RTO:              %.2f seconds\n", pol.TargetRTOSeconds)
	fmt.Printf("  Sovereignty Lock:        %t (NDMA 2026 Strict Bound)\n", pol.SovereigntyLock)
	fmt.Printf("  Automated Failover:      %t\n", pol.AutoFailoverEnabled)
	fmt.Printf("  Health Grace Period:     %d seconds\n", pol.HealthGracePeriodSeconds)
	fmt.Printf("  Min Witness Quorum:      %d probers\n", pol.MinWitnessQuorum)
	fmt.Printf("  Latest Rehearsal Score:  %.1f%% (Rule 68 Validated)\n", pol.RehearsalScore)
	fmt.Println()
	return nil
}

func handleFailover(ctx context.Context, client *Client, tenantID string, targetRegion string, force bool) error {
	fmt.Printf("Initiating failover sequence for '%s' to '%s' (Force: %t)...\n", tenantID, targetRegion, force)
	st, err := client.TriggerFailover(ctx, tenantID, targetRegion, force)
	if err != nil {
		return err
	}

	fmt.Println("\n====================================================================")
	fmt.Println(" Disaster Recovery Failover Successfully Executed")
	fmt.Println("====================================================================")
	fmt.Printf("  Tenant ID:               %s\n", st.TenantID)
	fmt.Printf("  New Active Region:       %s\n", strings.ToUpper(st.CurrentActiveRegion))
	fmt.Printf("  State:                   %s\n", st.State)
	fmt.Printf("  Measured RTO:            %.2f seconds\n", st.CurrentRTOEstimateSeconds)
	fmt.Printf("  Sovereignty Preserved:   %t\n", st.IsSovereigntyPreserved)
	fmt.Printf("  Action Attributed:       %s\n", st.LastFailoverReason)
	fmt.Println("--------------------------------------------------------------------")
	fmt.Println("  Traffic rerouted via BGP Anycast and WireGuard mesh update.")
	fmt.Println()
	return nil
}

func handleFailback(ctx context.Context, client *Client, tenantID string) error {
	fmt.Printf("Restoring tenant '%s' to primary home region...\n", tenantID)
	st, err := client.TriggerFailback(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Println("\n====================================================================")
	fmt.Println(" Disaster Recovery Failback Completed")
	fmt.Println("====================================================================")
	fmt.Printf("  Tenant ID:               %s\n", st.TenantID)
	fmt.Printf("  Restored Active Region:  %s\n", strings.ToUpper(st.CurrentActiveRegion))
	fmt.Printf("  State:                   %s\n", st.State)
	fmt.Printf("  Witness Health Score:    %.2f / 1.00\n", st.PrimaryHealthScore)
	fmt.Println("--------------------------------------------------------------------")
	fmt.Println("  Anycast BGP routes re-advertised to primary Point of Presence.")
	fmt.Println()
	return nil
}

func handleRehearsal(ctx context.Context, client *Client, tenantID string) error {
	fmt.Printf("Executing non-disruptive DR rehearsal simulation for '%s' (Rule 68)...\n", tenantID)
	report, err := client.RunRehearsal(ctx, tenantID)
	if err != nil {
		return err
	}

	fmt.Println("\n====================================================================")
	fmt.Println(" Disaster Recovery Rehearsal Report (Rule 68 Path Validation)")
	fmt.Println("====================================================================")
	fmt.Printf("  Rehearsal ID:            %s\n", report.RehearsalID)
	fmt.Printf("  Tenant ID:               %s\n", report.TenantID)
	fmt.Printf("  Simulated Primary:       %s\n", report.SimulatedSourceRegion)
	fmt.Printf("  Simulated Secondary:     %s\n", report.SimulatedTargetRegion)
	fmt.Printf("  Replication Lag:         %.3f seconds\n", report.ReplicationLagSeconds)
	fmt.Printf("  Secondary Capacity:      %t\n", report.SecondaryCapacityAvailable)
	fmt.Printf("  Sovereignty Compliant:   %t\n", report.SovereigntyCompliant)
	fmt.Printf("  BGP Withdrawal Tested:   %t\n", report.BGPWithdrawalSimulated)
	fmt.Printf("  Simulated RTO:           %.2f seconds\n", report.SimulatedRTOSeconds)
	fmt.Printf("  Overall Rehearsal:       %s\n", formatPassStatus(report.Passed))
	fmt.Printf("  Digital Seal (HMAC):     %s\n", report.DigitalSealHMAC)
	fmt.Printf("  Audit Summary:           %s\n", report.Details)
	fmt.Println()
	return nil
}

func handleJournal(ctx context.Context, client *Client, tenantID string) error {
	events, err := client.GetJournal(ctx, tenantID, 20)
	if err != nil {
		return err
	}

	fmt.Printf("\n=== Cryptographic DR Event Journal (Tenant: %s) ===\n", tenantID)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "EVENT ID\tEVENT TYPE\tFROM\tTO\tTRIGGER\tTIME\tSEAL HASH")
	for _, e := range events {
		shortHMAC := e.EventHMAC
		if len(shortHMAC) > 12 {
			shortHMAC = shortHMAC[:12] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			e.EventID, e.EventType, e.FromRegion, e.ToRegion, e.Trigger, e.Timestamp.Format("15:04:05"), shortHMAC)
	}
	w.Flush()
	fmt.Println()
	return nil
}

func handleProfiles(ctx context.Context, client *Client) error {
	profiles, err := client.ListProfiles(ctx)
	if err != nil {
		return err
	}

	fmt.Println("\n=== Multi-Region Tenant Isolation Profiles ===")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TENANT ID\tNAME\tJURISDICTION\tTIER\tVNET ID\tALLOWED REGIONS")
	for _, p := range profiles {
		allowed := strings.Join(p.AllowedRegions, ",")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			p.TenantID, p.TenantName, p.Jurisdiction, p.IsolationLevel, p.VNetID, allowed)
	}
	w.Flush()
	fmt.Println()
	return nil
}

func formatPassStatus(passed bool) string {
	if passed {
		return "PASSED (100% Ready for Zero-Loss Failover)"
	}
	return "FAILED (Remediation Required Before Live Outage)"
}

// PrintDRHelp displays command usage.
func PrintDRHelp() {
	fmt.Println(`Usage: aeitron dr <command> [options]

Commands:
  status [--tenant <id>]             Display multi-region DR status, health scores, and RPO/RTO
  policy [--tenant <id>]             Display tenant DR policy, topology, and legal constraints
  failover <tenant-id> [region]      Execute automated or operator-directed regional failover
  failback <tenant-id>               Restore tenant workload to primary home region
  rehearse <tenant-id>               Run non-disruptive DR dry-run simulation (Rule 68)
  journal [--tenant <id>]            View cryptographic tamper-evident event audit log
  profiles                           List tenant isolation profiles, overlay VNets, and boundaries

Flags:
  --tenant, -t <id>                  Target tenant identifier
  --force                            Force failover even if sovereignty warnings are present`)
}
