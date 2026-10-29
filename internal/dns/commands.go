package dns

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

// PrintDNSHelp displays available DNS management commands.
func PrintDNSHelp() {
	fmt.Println("Aeitron Anycast DNS & Traffic Steering Controller")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron dns <subcommand> [args] [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  zones list [flags]                 List all authoritative DNS zones")
	fmt.Println("  zones create <name> [flags]        Provision a new authoritative zone")
	fmt.Println("  zones delete <zone-id>             Delete a zone and all its resource records")
	fmt.Println("  records list <zone-id>             List all resource records in a zone")
	fmt.Println("  records add <zone-id> [flags]      Add a record with Anycast Geo/Latency routing")
	fmt.Println("  records delete <zone-id> <rec-id>  Delete a record from a zone (--confirm, --approval-token)")
	fmt.Println("  records request-delete <domain> <record-id> [--reason <val>]")
	fmt.Println("                                     Initiate dual-control record deletion challenge")
	fmt.Println("  approve <request-id> [--actor <val>]")
	fmt.Println("                                     Authorize pending deletion under two-person rule")
	fmt.Println("  resolve <name> [flags]             Query live Anycast Geo-steering DNS resolver")
	fmt.Println("  export <zone-id>                   Export zone in standard RFC 1035 BIND format")
	fmt.Println("  health                             Inspect Edge DNS Anycast health & global 5-region matrix")
	fmt.Println("  drain [--reason <val>]             Gracefully drain local PoP and withdraw BGP Anycast routes")
	fmt.Println("  restore                            Re-advertise BGP Anycast routes and resume autonomous health checks")
	fmt.Println("  failover                           Inspect recent Anycast failover events and route withdrawals")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --tenant <id>                      Tenant ID for multi-tenant isolation")
	fmt.Println("  --name <fqdn>                      Record FQDN (e.g. 'api.example.com')")
	fmt.Println("  --type <type>                      Record type (A, AAAA, CNAME, TXT, MX) (default: A)")
	fmt.Println("  --content <val>                    Target IP address or canonical hostname")
	fmt.Println("  --ttl <seconds>                    Time To Live in seconds (default: 300)")
	fmt.Println("  --policy <policy>                  Routing policy: SIMPLE, ANYCAST_GEO, LATENCY_OPTIMAL")
	fmt.Println("  --geo-targets <k=v,...>            Geo targets (e.g. 'dhaka=103.144.180.1,singapore=80.225.241.129')")
	fmt.Println("  --region <region>                  Client region for resolution test (dhaka, singapore, frankfurt)")
	fmt.Println("  --reason <val>                     Reason for draining the PoP (default: scheduled maintenance)")
}

// HandleDNSCommand routes DNS subcommands to appropriate handlers.
func HandleDNSCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintDNSHelp()
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "zones":
		return handleZones(client, rest)
	case "records":
		return handleRecords(client, rest)
	case "approve", "approve-deletion":
		return handleApproveDeletion(client, rest)
	case "request-delete", "request-deletion":
		return handleRequestDeletion(client, rest)
	case "resolve":
		return handleResolve(client, rest)
	case "export":
		return handleExport(client, rest)
	case "health":
		return handleHealth(client, rest)
	case "drain":
		return handleDrain(client, rest)
	case "restore":
		return handleRestore(client, rest)
	case "failover":
		return handleFailover(client, rest)
	case "help", "--help", "-h":
		PrintDNSHelp()
		return nil
	default:
		PrintDNSHelp()
		return fmt.Errorf("unknown dns subcommand: %s", sub)
	}
}

func handleZones(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron dns zones <list|create|delete> [flags]")
	}

	action := args[0]
	rest := args[1:]

	switch action {
	case "list":
		fs := flag.NewFlagSet("zones list", flag.ContinueOnError)
		var tenantID string
		fs.StringVar(&tenantID, "tenant", "", "Tenant ID")
		if err := fs.Parse(rest); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		zones, err := client.ListZones(ctx, tenantID)
		if err != nil {
			return err
		}

		if len(zones) == 0 {
			fmt.Println("No authoritative DNS zones found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ZONE ID\tDOMAIN NAME\tSERIAL\tSTATUS\tDNSSEC\tTENANT")
		for _, z := range zones {
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%v\t%s\n",
				z.ID, z.Name, z.Serial, z.Status, z.DNSSEC, z.TenantID)
		}
		w.Flush()
		return nil

	case "create":
		if len(rest) == 0 {
			return errors.New("usage: aeitron dns zones create <domain-name> [--tenant <id>]")
		}
		domainName := rest[0]
		fs := flag.NewFlagSet("zones create", flag.ContinueOnError)
		var tenantID string
		fs.StringVar(&tenantID, "tenant", "tenant-default", "Tenant ID")
		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		zone, err := client.CreateZone(ctx, tenantID, domainName)
		if err != nil {
			return err
		}

		fmt.Printf("SUCCESS: Created authoritative zone %q (ID: %s, Serial: %d, DNSSEC: %v)\n",
			zone.Name, zone.ID, zone.Serial, zone.DNSSEC)
		return nil

	case "delete":
		if len(rest) == 0 {
			return errors.New("usage: aeitron dns zones delete <zone-id>")
		}
		zoneID := rest[0]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.DeleteZone(ctx, zoneID); err != nil {
			return err
		}
		fmt.Printf("SUCCESS: Deleted zone %q\n", zoneID)
		return nil

	default:
		return fmt.Errorf("unknown zones action: %s", action)
	}
}

func handleRecords(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron dns records <list|add|delete|request-delete> <zone-id> [flags]")
	}

	action := args[0]
	rest := args[1:]

	switch action {
	case "request-delete", "request-deletion":
		return handleRequestDeletion(client, rest)

	case "approve", "approve-deletion":
		return handleApproveDeletion(client, rest)
	case "list":
		if len(rest) == 0 {
			return errors.New("usage: aeitron dns records list <zone-id>")
		}
		zoneID := rest[0]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		records, err := client.ListRecords(ctx, zoneID)
		if err != nil {
			return err
		}

		if len(records) == 0 {
			fmt.Printf("No records found in zone %s\n", zoneID)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RECORD ID\tNAME\tTYPE\tTTL\tPOLICY\tCONTENT / TARGETS")
		for _, r := range records {
			contentStr := r.Content
			if len(r.GeoTargets) > 0 {
				var targets []string
				for reg, ip := range r.GeoTargets {
					targets = append(targets, fmt.Sprintf("%s=%s", reg, ip))
				}
				contentStr += fmt.Sprintf(" (Geo: %s)", strings.Join(targets, ", "))
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				r.ID, r.Name, r.Type, r.TTL, r.RoutingPolicy, contentStr)
		}
		w.Flush()
		return nil

	case "add":
		if len(rest) == 0 {
			return errors.New("usage: aeitron dns records add <zone-id> --name <fqdn> --type <A|AAAA|CNAME> --content <target> [flags]")
		}
		zoneID := rest[0]
		fs := flag.NewFlagSet("records add", flag.ContinueOnError)
		var name, rType, content, policyStr, geoTargetsStr string
		var ttl uint

		fs.StringVar(&name, "name", "", "Record name")
		fs.StringVar(&rType, "type", "A", "Record type")
		fs.StringVar(&content, "content", "", "Target IP or hostname")
		fs.UintVar(&ttl, "ttl", 300, "TTL seconds")
		fs.StringVar(&policyStr, "policy", "SIMPLE", "Routing policy")
		fs.StringVar(&geoTargetsStr, "geo-targets", "", "Geo targets (k=v,...)")

		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}

		if name == "" || content == "" {
			return errors.New("flags --name and --content are required")
		}

		geoTargets := make(map[string]string)
		if geoTargetsStr != "" {
			pairs := strings.Split(geoTargetsStr, ",")
			for _, p := range pairs {
				kv := strings.SplitN(strings.TrimSpace(p), "=", 2)
				if len(kv) == 2 {
					geoTargets[strings.ToLower(kv[0])] = kv[1]
				}
			}
		}

		rec := &DNSRecord{
			ZoneID:        zoneID,
			Name:          name,
			Type:          RecordType(strings.ToUpper(rType)),
			TTL:           uint32(ttl),
			Content:       content,
			RoutingPolicy: RoutingPolicy(strings.ToUpper(policyStr)),
			GeoTargets:    geoTargets,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		created, err := client.AddRecord(ctx, rec)
		if err != nil {
			return err
		}

		fmt.Printf("SUCCESS: Added %s record for %q -> %s (ID: %s, Policy: %s)\n",
			created.Type, created.Name, created.Content, created.ID, created.RoutingPolicy)
		return nil

	case "delete":
		if len(rest) < 2 {
			return errors.New("usage: aeitron dns records delete <zone-id> <record-id> [--confirm] [--approval-token <token>]")
		}
		zoneID := rest[0]
		recordID := rest[1]

		fs := flag.NewFlagSet("records delete", flag.ContinueOnError)
		var confirm bool
		var approvalToken string
		fs.BoolVar(&confirm, "confirm", false, "Confirm deletion of protected record")
		fs.StringVar(&approvalToken, "approval-token", "", "Challenge approval token from two-person approval workflow")
		if err := fs.Parse(rest[2:]); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.DeleteRecord(ctx, zoneID, recordID, confirm, approvalToken); err != nil {
			if strings.Contains(err.Error(), "APPROVAL_REQUIRED") || strings.Contains(err.Error(), "del_req_") {
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Println(" [SECURITY CHALLENGE] Independent Operator Approval Required")
				fmt.Println("--------------------------------------------------------------------------------")
				fmt.Printf("This DNS record is protected by the two-person rule.\n%v\n\n", err)
				fmt.Println("Required Steps:")
				fmt.Printf(" 1. Request an authorized operator to approve:\n    aeitron dns approve <request-id> --actor <operator-id>\n")
				fmt.Printf(" 2. Re-run deletion with approval token and --confirm:\n    aeitron dns records delete %s %s --confirm --approval-token <token>\n", zoneID, recordID)
				return errors.New("record deletion blocked pending human operator approval")
			}
			return err
		}
		fmt.Printf("SUCCESS: Deleted record %q from zone %q\n", recordID, zoneID)
		return nil

	default:
		return fmt.Errorf("unknown records action: %s", action)
	}
}

func handleResolve(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron dns resolve <domain-name> [--region <region>] [--type <type>]")
	}

	queryName := args[0]
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	var region, rType string
	fs.StringVar(&region, "region", "dhaka", "Client region (dhaka, singapore, frankfurt, virginia)")
	fs.StringVar(&rType, "type", "A", "Record type (A, AAAA, CNAME)")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	req := ResolveRequest{
		QueryName:    queryName,
		RecordType:   RecordType(strings.ToUpper(rType)),
		ClientRegion: region,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.Resolve(ctx, req)
	if err != nil {
		return err
	}

	fmt.Printf("[Anycast DNS] Resolved %s (%s) from Client Region: %s (Latency: %.1fms)\n",
		resp.QueryName, resp.RecordType, resp.SelectedRegion, resp.LatencyMs)
	fmt.Printf("Matched Policy: %s\n", resp.MatchedPolicy)
	fmt.Println()

	if len(resp.Records) == 0 {
		fmt.Println("No records returned (NXDOMAIN / NODATA).")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tTTL\tANSWER / TARGET IP")
	for _, r := range resp.Records {
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", r.Name, r.Type, r.TTL, r.Content)
	}
	w.Flush()
	return nil
}

func handleExport(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron dns export <zone-id>")
	}
	zoneID := args[0]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bindText, err := client.ExportBIND(ctx, zoneID)
	if err != nil {
		return err
	}
	fmt.Print(bindText)
	return nil
}

func handleHealth(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st, err := client.GetHealthStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch local health status: %w", err)
	}

	pops, err := client.GetGlobalHealthMatrix(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch global matrix: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Autonomous Edge DNS Anycast Health & Failover HUD")
	fmt.Println("================================================================================")
	fmt.Printf("Local PoP:                 %s (%s)\n", st.RegionID, st.Name)
	fmt.Printf("Health State:              %s\n", st.State)
	fmt.Printf("BGP Announcement:          %s\n", st.BGPState)
	fmt.Printf("Anycast IPv4 / IPv6:       %s / %s\n", st.AnycastIPv4, st.AnycastIPv6)
	fmt.Printf("Failover Target:           %s\n", st.FailoverPoPID)
	fmt.Printf("Flap Damping Penalty:      %.2f (Damped: %t)\n", st.FlapDampingPenalty, st.Damped)
	fmt.Printf("Last Probe Latency:        %.2fms\n", st.LastProbeLatencyMs)
	sovStr := "NO"
	if st.SovereigntyRestricted {
		sovStr = "RESTRICTED (Bangladesh NDMA 2026)"
	}
	fmt.Printf("Sovereign Boundary:        %s\n", sovStr)
	if st.DrainReason != "" {
		fmt.Printf("Drain Reason:              %s\n", st.DrainReason)
	}
	fmt.Println()
	fmt.Println("Global 5-Region Anycast Health Matrix:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "  POP ID\tNAME\tSTATE\tBGP STATE\tFAILOVER\tSOVEREIGN")
	for _, p := range pops {
		pSov := "NO"
		if p.SovereigntyRestricted {
			pSov = "YES (NDMA 2026)"
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%s\t%s\t%s\n",
			p.RegionID, p.Name, p.State, p.BGPState, p.FailoverPoPID, pSov)
	}
	w.Flush()
	fmt.Println("================================================================================")
	return nil
}

func handleDrain(client *Client, args []string) error {
	fs := flag.NewFlagSet("dns drain", flag.ContinueOnError)
	reason := fs.String("reason", "scheduled operator maintenance", "Reason for draining the PoP")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Printf("Draining PoP and withdrawing BGP Anycast routes (Reason: %s)...\n", *reason)
	pop, err := client.DrainPoP(ctx, *reason)
	if err != nil {
		return fmt.Errorf("drain failed: %w", err)
	}

	fmt.Printf("PoP %s successfully transitioned to %s.\n", pop.RegionID, pop.State)
	fmt.Printf("BGP Announcement Posture: %s (Routes Withdrawn, Traffic Reconverged)\n", pop.BGPState)
	fmt.Printf("Failover Target:          %s\n", pop.FailoverPoPID)
	return nil
}

func handleRestore(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Restoring PoP to active routing and re-advertising BGP Anycast routes...")
	pop, err := client.RestorePoP(ctx)
	if err != nil {
		return fmt.Errorf("restore failed: %w", err)
	}

	fmt.Printf("PoP %s successfully restored to %s.\n", pop.RegionID, pop.State)
	fmt.Printf("BGP Announcement Posture: %s (Active Routing Resumed)\n", pop.BGPState)
	return nil
}

func handleFailover(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := client.GetFailoverEvents(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch failover events: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Edge DNS Anycast Failover & Route Withdrawal History")
	fmt.Println("================================================================================")
	if len(events) == 0 {
		fmt.Println("No failover events recorded. All Anycast PoPs operating nominally.")
		fmt.Println("================================================================================")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "  SEQ\tTIMESTAMP\tPOP\tSTATE TRANSITION\tBGP TRANSITION\tTRIGGER\tREASON")
	for _, ev := range events {
		stateTrans := fmt.Sprintf("%s -> %s", ev.OldState, ev.NewState)
		bgpTrans := fmt.Sprintf("%s -> %s", ev.OldBGPState, ev.NewBGPState)
		fmt.Fprintf(w, "  %d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			ev.Sequence, ev.Timestamp.Format("15:04:05"), ev.RegionID, stateTrans, bgpTrans, ev.TriggeredBy, ev.Reason)
	}
	w.Flush()
	fmt.Println("================================================================================")
	return nil
}

func handleApproveDeletion(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron dns approve <request-id> [--actor <val>]")
	}
	requestID := args[0]
	fs := flag.NewFlagSet("dns approve", flag.ContinueOnError)
	var actor string
	fs.StringVar(&actor, "actor", "", "Operator ID authorizing the deletion")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if actor == "" {
		actor = os.Getenv("USER")
		if actor == "" {
			actor = os.Getenv("USERNAME")
		}
		if actor == "" {
			actor = "human_operator"
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.ApproveRecordDeletion(ctx, requestID, actor)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println(" DNS RECORD DELETION APPROVED (DUAL-CONTROL RULE)")
	fmt.Println("================================================================================")
	fmt.Printf("Request ID:     %s\n", result.RequestID)
	fmt.Printf("Status:         %s\n", result.Status)
	fmt.Printf("Approved By:    %s\n", result.ApprovedBy)
	fmt.Printf("Approval Token: %s\n", result.ApprovalToken)
	fmt.Printf("Valid Until:    %s\n", result.ValidUntil.Format(time.RFC3339))
	fmt.Printf("Message:        %s\n", result.Message)
	fmt.Println()
	fmt.Println("To complete deletion with this token:")
	fmt.Printf("  aeitron dns records delete <zone-id> <record-id> --confirm --approval-token %s\n", result.ApprovalToken)
	return nil
}

func handleRequestDeletion(client *Client, args []string) error {
	if len(args) < 2 {
		return errors.New("usage: aeitron dns records request-delete <domain> <record-id> [--reason <val>]")
	}
	domain := args[0]
	recordID := args[1]
	fs := flag.NewFlagSet("dns request-delete", flag.ContinueOnError)
	var reason string
	fs.StringVar(&reason, "reason", "Administrative deletion requested", "Reason for record deletion")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.RequestRecordDeletion(ctx, domain, recordID, reason)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println(" DNS RECORD DELETION REQUEST INITIATED")
	fmt.Println("================================================================================")
	fmt.Printf("Request ID:      %s\n", result.RequestID)
	fmt.Printf("Domain:          %s\n", result.Domain)
	fmt.Printf("Record ID:       %s\n", result.RecordID)
	fmt.Printf("Requester:       %s\n", result.Requester)
	fmt.Printf("Status:          %s\n", result.Status)
	fmt.Printf("Challenge Token: %s\n", result.ChallengeToken)
	fmt.Printf("Expires At:      %s\n", result.ExpiresAt.Format(time.RFC3339))
	fmt.Println()
	fmt.Println("An independent operator must approve this request before deletion can proceed:")
	fmt.Printf("  aeitron dns approve %s --actor <operator-id>\n", result.RequestID)
	return nil
}
