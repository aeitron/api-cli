package security

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintSecurityHelp displays available security and sovereignty CLI commands.
func PrintSecurityHelp() {
	fmt.Println("Aeitron Security Policy & Bangladesh NDMA 2026 Sovereignty Engine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron security <subcommand> [args] [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  audit [flags]                     Audit workload sovereignty and NDMA 2026 compliance")
	fmt.Println("  audit-log                         Inspect tamper-proof cryptographic Merkle audit journal")
	fmt.Println("  firewall list                     List all active L4 eBPF / XDP drop entries")
	fmt.Println("  firewall block <ip> [flags]       Inject dynamic L4 drop rule for abusive IP")
	fmt.Println("  firewall unblock <ip>             Remove IP from drop list")
	fmt.Println("  policy get <domain-id>            Inspect domain WAF and edge security posture")
	fmt.Println("  policy set <domain-id> [flags]    Configure domain WAF, rate limits, and bot defenses")
	fmt.Println("  hsm status                        Inspect PKCS#11 hardware token and FIPS 140-3 status")
	fmt.Println("  hsm slots                         List all detected hardware cryptographic slots")
	fmt.Println("  hsm test-wrap                     Execute active round-trip key wrap/unwrap test")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --tenant <id>                     Tenant identifier")
	fmt.Println("  --project <id>                    Project identifier")
	fmt.Println("  --workload <id>                   Workload identifier")
	fmt.Println("  --source <region>                 Source origin region (e.g. 'dhaka')")
	fmt.Println("  --target <region>                 Target destination region (e.g. 'singapore')")
	fmt.Println("  --classification <class>          CII_CRITICAL, FINANCIAL_BANKING, HEALTHCARE_RECORDS, GENERAL_ENTERPRISE")
	fmt.Println("  --strict                          Enforce strict sovereignty constraints (default: true)")
	fmt.Println("  --local-mirror                    Workload maintains active synchronized Dhaka mirror")
	fmt.Println("  --reason <string>                 Reason for firewall block action")
	fmt.Println("  --duration <seconds>              Drop rule duration in seconds (default: 3600)")
	fmt.Println("  --waf                             Enable/disable WAF")
	fmt.Println("  --waf-mode <mode>                 WAF mode: BLOCK or LOG")
	fmt.Println("  --owasp                           Enable/disable OWASP Core Ruleset")
	fmt.Println("  --rate-limit                      Enable/disable Edge Rate Limiting")
	fmt.Println("  --rpm <requests>                  Rate limit requests per minute")
	fmt.Println("  --burst <requests>                Rate limit burst capacity")
	fmt.Println("  --bot                             Enable/disable Bot Protection")
}

// HandleSecurityCommand routes security subcommands.
func HandleSecurityCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintSecurityHelp()
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "audit", "sovereignty":
		return handleAudit(client, rest)
	case "audit-log", "log":
		return handleAuditLog(client, rest)
	case "firewall":
		return handleFirewall(client, rest)
	case "policy":
		return handlePolicy(client, rest)
	case "hsm", "pkcs11":
		return HandleHSMCommand(client, rest)
	case "help", "--help", "-h":
		PrintSecurityHelp()
		return nil
	default:
		PrintSecurityHelp()
		return fmt.Errorf("unknown security subcommand: %s", sub)
	}
}

func handleAudit(client *Client, args []string) error {
	fs := flag.NewFlagSet("security audit", flag.ContinueOnError)
	var tenantID, projectID, workloadID, source, target, classStr string
	var strict, localMirror bool

	fs.StringVar(&tenantID, "tenant", "tenant-default", "Tenant ID")
	fs.StringVar(&projectID, "project", "project-alpha", "Project ID")
	fs.StringVar(&workloadID, "workload", "app-core", "Workload ID")
	fs.StringVar(&source, "source", "dhaka", "Source Region")
	fs.StringVar(&target, "target", "singapore", "Target Region")
	fs.StringVar(&classStr, "classification", "CII_CRITICAL", "Data Classification")
	fs.BoolVar(&strict, "strict", true, "Strict Sovereignty")
	fs.BoolVar(&localMirror, "local-mirror", false, "Has Local Dhaka Mirror")

	if err := fs.Parse(args); err != nil {
		return err
	}

	req := SovereigntyEvaluationRequest{
		TenantID:            tenantID,
		ProjectID:           projectID,
		WorkloadID:          workloadID,
		SourceRegion:        source,
		TargetRegion:        target,
		Classification:      DataClassification(classStr),
		StrictSovereignty:   strict,
		HasLocalDhakaMirror: localMirror,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.EvaluateSovereignty(ctx, req)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println("              AEITRON SOVEREIGNTY & DATA RESIDENCY AUDIT VERDICT               ")
	fmt.Println("================================================================================")
	fmt.Printf("Workload ID:       %s\n", req.WorkloadID)
	fmt.Printf("Classification:    %s\n", req.Classification)
	fmt.Printf("Routing Path:      %s -> %s\n", req.SourceRegion, req.TargetRegion)
	fmt.Printf("Dhaka Mirror:      %v\n", req.HasLocalDhakaMirror)
	fmt.Printf("Standard:          %s\n", result.Standard)
	fmt.Printf("Verdict:           %s\n", result.Verdict)
	fmt.Printf("Compliant:         %v\n", result.Compliant)
	fmt.Printf("Evaluation Reason: %s\n", result.Reason)
	fmt.Printf("Audit Hash:        %s\n", result.AuditHash)
	fmt.Printf("Audit Index:       %d\n", result.AuditIndex)
	fmt.Printf("Timestamp:         %s\n", result.EvaluatedAt.Format(time.RFC3339))
	fmt.Println("================================================================================")

	if !result.Compliant {
		return fmt.Errorf("audit failed: workload violates %s data residency mandate", result.Standard)
	}
	return nil
}

func handleAuditLog(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entries, verified, err := client.GetAuditChain(ctx)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println("       AEITRON BANGLADESH NDMA 2026 CRYPTOGRAPHIC AUDIT JOURNAL               ")
	fmt.Println("================================================================================")
	statusStr := "VERIFIED (SHA-256 Merkle Chain Valid)"
	if !verified {
		statusStr = "COMPROMISED (Tamper Detected)"
	}
	fmt.Printf("Chain Integrity Status:  %s\n", statusStr)
	fmt.Printf("Total Audit Records:     %d\n", len(entries))
	fmt.Println("================================================================================")

	if len(entries) == 0 {
		fmt.Println("No audit records found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INDEX\tTIMESTAMP\tACTION\tSUBJECT\tVERDICT\tCURRENT HASH")
	for _, e := range entries {
		shortHash := e.CurrentHash
		if len(shortHash) > 16 {
			shortHash = shortHash[:16] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			e.Index, e.Timestamp.Format("2006-01-02 15:04:05"), e.Action, e.Subject, e.Verdict, shortHash)
	}
	w.Flush()
	return nil
}

func handleFirewall(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron security firewall <list|block|unblock> [flags]")
	}

	action := args[0]
	rest := args[1:]

	switch action {
	case "list":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ips, err := client.ListBlockedIPs(ctx)
		if err != nil {
			return err
		}

		if len(ips) == 0 {
			fmt.Println("No active L4 drop entries in edge firewall.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "IP ADDRESS\tFILTER MODE\tDROPPED PKTS\tEXPIRES AT\tREASON")
		for _, b := range ips {
			exp := b.ExpiresAt.Format("2006-01-02 15:04:05")
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
				b.IPAddress, b.FilterMode, b.DroppedPackets, exp, b.Reason)
		}
		w.Flush()
		return nil

	case "block":
		if len(rest) == 0 {
			return errors.New("usage: aeitron security firewall block <ip> [--reason <reason>] [--duration <seconds>]")
		}
		ip := rest[0]
		fs := flag.NewFlagSet("firewall block", flag.ContinueOnError)
		var reason string
		var duration int
		fs.StringVar(&reason, "reason", "cli-manual-mitigation", "Reason for drop rule")
		fs.IntVar(&duration, "duration", 3600, "Drop duration in seconds")

		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		entry, err := client.BlockIP(ctx, ip, reason, duration)
		if err != nil {
			return err
		}

		fmt.Printf("SUCCESS: Injected L4 drop rule for %s (Mode: %s, Expires: %s, Reason: %s)\n",
			entry.IPAddress, entry.FilterMode, entry.ExpiresAt.Format(time.RFC3339), entry.Reason)
		return nil

	case "unblock":
		if len(rest) == 0 {
			return errors.New("usage: aeitron security firewall unblock <ip>")
		}
		ip := rest[0]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.UnblockIP(ctx, ip); err != nil {
			return err
		}
		fmt.Printf("SUCCESS: Removed %s from edge firewall drop list\n", ip)
		return nil

	default:
		return fmt.Errorf("unknown firewall action: %s", action)
	}
}

func handlePolicy(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron security policy <get|set> <domain-id> [flags]")
	}

	action := args[0]
	rest := args[1:]

	switch action {
	case "get":
		if len(rest) == 0 {
			return errors.New("usage: aeitron security policy get <domain-id>")
		}
		domainID := rest[0]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		p, err := client.GetPolicy(ctx, domainID)
		if err != nil {
			return err
		}

		fmt.Println("================================================================================")
		fmt.Printf("               EDGE SECURITY POSTURE: %s\n", p.DomainID)
		fmt.Println("================================================================================")
		fmt.Printf("Policy ID:         %s\n", p.PolicyID)
		fmt.Printf("WAF Enabled:       %v (Mode: %s)\n", p.WAFEnabled, p.WAFMode)
		fmt.Printf("OWASP Ruleset:     %v\n", p.OWASPRulesetEnabled)
		fmt.Printf("Rate Limiting:     %v (%d RPM, Burst: %d)\n", p.RateLimitEnabled, p.RateLimitRPM, p.BurstCapacity)
		fmt.Printf("Bot Protection:    %v\n", p.BotProtectionEnabled)
		fmt.Printf("Last Updated:      %s\n", p.UpdatedAt.Format(time.RFC3339))
		fmt.Println("================================================================================")
		return nil

	case "set":
		if len(rest) == 0 {
			return errors.New("usage: aeitron security policy set <domain-id> [flags]")
		}
		domainID := rest[0]
		fs := flag.NewFlagSet("policy set", flag.ContinueOnError)
		var waf, owasp, rateLimit, bot bool
		var wafMode string
		var rpm, burst uint

		fs.BoolVar(&waf, "waf", true, "Enable WAF")
		fs.StringVar(&wafMode, "waf-mode", "BLOCK", "WAF Mode: BLOCK or LOG")
		fs.BoolVar(&owasp, "owasp", true, "Enable OWASP Ruleset")
		fs.BoolVar(&rateLimit, "rate-limit", true, "Enable Rate Limiting")
		fs.UintVar(&rpm, "rpm", 10000, "Requests per minute")
		fs.UintVar(&burst, "burst", 500, "Burst capacity")
		fs.BoolVar(&bot, "bot", true, "Enable Bot Protection")

		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}

		p := &SecurityPolicy{
			DomainID:             domainID,
			WAFEnabled:           waf,
			WAFMode:              strings.ToUpper(wafMode),
			OWASPRulesetEnabled:  owasp,
			RateLimitEnabled:     rateLimit,
			RateLimitRPM:         uint32(rpm),
			BurstCapacity:        uint32(burst),
			BotProtectionEnabled: bot,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		saved, err := client.UpdatePolicy(ctx, p)
		if err != nil {
			return err
		}

		fmt.Printf("SUCCESS: Updated edge security policy for domain %q (WAF: %v, Mode: %s, RateLimit: %d RPM)\n",
			saved.DomainID, saved.WAFEnabled, saved.WAFMode, saved.RateLimitRPM)
		return nil

	default:
		return fmt.Errorf("unknown policy action: %s", action)
	}
}
