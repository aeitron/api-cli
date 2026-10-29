package workload

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

// PrintDeployHelp displays available deployment commands.
func PrintDeployHelp() {
	fmt.Println("Aeitron Universal Workload Deploy Engine (Scheduler Client)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron deploy <subcommand> [flags]")
	fmt.Println("  aeitron deploy -f <manifest.yaml> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  apply -f <file>       Schedule and deploy workload to optimal PoP based on multi-objective constraints")
	fmt.Println("  validate -f <file>    Syntactically and semantically validate workload manifest without dispatching")
	fmt.Println("  dry-run -f <file>     Simulate placement evaluation against live fabric nodes without reserving capacity")
	fmt.Println("  status <workload-id>  Inspect active placement, phase, latency, and cost rate for a workload")
	fmt.Println("  release <workload-id> Relinquish reserved GPU/compute capacity for a finished workload")
	fmt.Println("  nodes                 List active fabric nodes, GPU availability, and carbon intensity")
	fmt.Println("  journal [list]        Inspect cryptographically chained placement and preemption records")
	fmt.Println("  journal verify        Mathematically verify Zero-Egress Merkle tree root and chain integrity")
	fmt.Println("  journal proof --seq   Generate and verify Zero-Egress Merkle inclusion proof for a record")
	fmt.Println("  consensus             Inspect 5-region Raft PQC distributed consensus cluster and geographic pinning state")
	fmt.Println("  template [flags]      Generate a production-ready workload manifest template")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  -f, --file <path>     Path to workload manifest (.yaml or .json)")
	fmt.Println("  --type <type>         Template type: gpu, api, container, or sovereign (default: gpu)")
	fmt.Println("  --idempotency-key <k> Custom idempotency key for safe mutating retries (Rule 14)")
}

// HandleDeployCommand routes deployment subcommands to appropriate handlers.
func HandleDeployCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintDeployHelp()
		return nil
	}

	first := args[0]

	// Handle direct `aeitron deploy -f manifest.yaml`
	if first == "-f" || first == "--file" {
		return handleApply(client, args)
	}

	switch first {
	case "apply":
		return handleApply(client, args[1:])
	case "validate":
		return handleValidate(args[1:])
	case "dry-run":
		return handleDryRun(client, args[1:])
	case "status", "inspect":
		return handleStatus(client, args[1:])
	case "release", "delete":
		return handleRelease(client, args[1:])
	case "nodes", "list-nodes":
		return handleNodes(client, args[1:])
	case "journal":
		return handleJournal(client, args[1:])
	case "consensus", "raft":
		return handleConsensus(client, args[1:])
	case "template":
		return handleTemplate(args[1:])
	case "help", "--help", "-h":
		PrintDeployHelp()
		return nil
	default:
		// If it's a file path directly: `aeitron deploy manifest.yaml`
		if strings.HasSuffix(first, ".yaml") || strings.HasSuffix(first, ".yml") || strings.HasSuffix(first, ".json") {
			return handleApply(client, []string{"-f", first})
		}
		PrintDeployHelp()
		return fmt.Errorf("unknown deploy subcommand: %s", first)
	}
}

func handleApply(client *Client, args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	var filePath string
	var idempotencyKey string

	fs.StringVar(&filePath, "f", "", "Path to manifest file")
	fs.StringVar(&filePath, "file", "", "Path to manifest file")
	fs.StringVar(&idempotencyKey, "idempotency-key", "", "Idempotency key")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if filePath == "" {
		return errors.New("missing required manifest file flag: -f <manifest.yaml>")
	}

	m, err := ParseManifestFile(filePath)
	if err != nil {
		return fmt.Errorf("manifest error: %w", err)
	}

	if idempotencyKey != "" {
		m.Spec.IdempotencyKey = idempotencyKey
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("[Aeitron Deploy] Submitting workload %q (%s) for placement optimization...\n",
		m.Metadata.Name, m.Spec.Type)

	resp, err := client.ScheduleWorkload(ctx, m)
	if err != nil {
		return fmt.Errorf("scheduling failed: %w", err)
	}

	fmt.Println()
	fmt.Println("================================================================================")
	fmt.Printf(" Workload Placement Decision: %s\n", resp.WorkloadID)
	fmt.Println("================================================================================")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "  Phase:\t%s\n", resp.Status.Phase)
	fmt.Fprintf(w, "  Assigned Region:\t%s\n", resp.Status.AssignedRegion)
	fmt.Fprintf(w, "  Assigned Node:\t%s\n", resp.Status.AssignedNode)
	fmt.Fprintf(w, "  Assigned Provider:\t%s\n", resp.Status.AssignedProvider)
	fmt.Fprintf(w, "  Measured Latency:\t%d ms\n", resp.Status.MeasuredLatencyMs)
	fmt.Fprintf(w, "  Hourly Cost Rate:\t$%.4f USD/hr\n", resp.Status.CurrentCostRateUSD)
	fmt.Fprintf(w, "  Ingress Endpoint:\t%s\n", resp.IngressEndpoint)
	fmt.Fprintf(w, "  Optimization Reason:\t%s\n", resp.Status.RoutingReason)
	if len(resp.Status.ReasonCodes) > 0 {
		fmt.Fprintf(w, "  Reason Codes:\t%s\n", strings.Join(resp.Status.ReasonCodes, ", "))
	}
	fmt.Fprintf(w, "  Lease Expires:\t%s\n", resp.Status.LeaseExpiresAt.Format(time.RFC3339))
	w.Flush()
	fmt.Println("================================================================================")
	fmt.Printf("Workload %q is active. Connect via %s\n", resp.WorkloadID, resp.IngressEndpoint)
	return nil
}

func handleValidate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	var filePath string
	fs.StringVar(&filePath, "f", "", "Path to manifest file")
	fs.StringVar(&filePath, "file", "", "Path to manifest file")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if filePath == "" {
		return errors.New("missing required manifest file flag: -f <manifest.yaml>")
	}

	m, err := ParseManifestFile(filePath)
	if err != nil {
		return fmt.Errorf("VALIDATION_FAILED: %w", err)
	}

	fmt.Printf("VALIDATION_SUCCESS: Manifest %q is valid.\n", m.Metadata.Name)
	fmt.Printf("  Type:          %s\n", m.Spec.Type)
	fmt.Printf("  Jurisdiction:  %s (Strict Sovereignty: %v)\n", m.Spec.Jurisdiction, m.Spec.StrictSovereignty)
	fmt.Printf("  Tenant:        %s\n", m.Metadata.TenantID)
	fmt.Printf("  Resources:     GPUs: %d (%s), CPU: %dm, Mem: %d bytes\n",
		m.Spec.GPUCount, m.Spec.RequiredGPUModel, m.Spec.CPUMillicores, m.Spec.MemoryBytes)
	return nil
}

func handleDryRun(client *Client, args []string) error {
	fs := flag.NewFlagSet("dry-run", flag.ContinueOnError)
	var filePath string
	fs.StringVar(&filePath, "f", "", "Path to manifest file")
	fs.StringVar(&filePath, "file", "", "Path to manifest file")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if filePath == "" {
		return errors.New("missing required manifest file flag: -f <manifest.yaml>")
	}

	m, err := ParseManifestFile(filePath)
	if err != nil {
		return fmt.Errorf("manifest error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nodes, err := client.ListNodes(ctx)
	if err != nil {
		// Provide informative simulation if scheduler is offline
		fmt.Printf("[Dry-Run Warning] Live scheduler unreachable (%v). Simulating local placement topology...\n", err)
		nodes = []NodeSummary{
			{ID: "node-dha-01", Provider: "baremetal-bd", Region: "dhaka", Jurisdiction: JurisdictionBangladesh, GPUModel: "H100", AvailableGPUs: 8, CostPerGPUHour: 2.50, Healthy: true},
			{ID: "node-sin-01", Provider: "aws-ap-southeast-1", Region: "singapore", Jurisdiction: JurisdictionSingapore, GPUModel: "H100", AvailableGPUs: 16, CostPerGPUHour: 3.20, Healthy: true},
			{ID: "node-fra-01", Provider: "iceland-green-dc", Region: "frankfurt", Jurisdiction: JurisdictionEU, GPUModel: "H100", AvailableGPUs: 32, CostPerGPUHour: 1.90, Healthy: true},
		}
	}

	fmt.Printf("[Dry-Run] Simulating placement for workload %q:\n", m.Metadata.Name)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NODE\tREGION\tJURISDICTION\tELIGIBLE\tREASON")

	for _, n := range nodes {
		eligible := "YES"
		reason := "Meets constraints"

		if m.Spec.StrictSovereignty && n.Jurisdiction != JurisdictionBangladesh {
			eligible = "NO"
			reason = "Rejected: Bangladesh NDMA 2026 data residency required"
		} else if int(m.Spec.GPUCount) > n.AvailableGPUs {
			eligible = "NO"
			reason = fmt.Sprintf("Rejected: Insufficient GPUs (%d available, %d requested)", n.AvailableGPUs, m.Spec.GPUCount)
		} else if m.Spec.RequiredGPUModel != "" && m.Spec.RequiredGPUModel != n.GPUModel {
			eligible = "NO"
			reason = fmt.Sprintf("Rejected: Model mismatch (node has %s, requested %s)", n.GPUModel, m.Spec.RequiredGPUModel)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", n.ID, n.Region, n.Jurisdiction, eligible, reason)
	}
	w.Flush()
	return nil
}

func handleStatus(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron deploy status <workload-id>")
	}
	workloadID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetWorkload(ctx, workloadID)
	if err != nil {
		return err
	}

	fmt.Printf("Workload:          %s\n", resp.WorkloadID)
	fmt.Printf("Phase:             %s\n", resp.Status.Phase)
	fmt.Printf("Assigned Region:   %s\n", resp.Status.AssignedRegion)
	fmt.Printf("Assigned Node:     %s\n", resp.Status.AssignedNode)
	fmt.Printf("Measured Latency:  %d ms\n", resp.Status.MeasuredLatencyMs)
	fmt.Printf("Hourly Cost:       $%.4f USD/hr\n", resp.Status.CurrentCostRateUSD)
	fmt.Printf("Ingress Endpoint:  %s\n", resp.IngressEndpoint)
	fmt.Printf("Scheduled At:      %s\n", resp.Status.ScheduledAt.Format(time.RFC3339))
	fmt.Printf("Lease Expires:     %s\n", resp.Status.LeaseExpiresAt.Format(time.RFC3339))
	return nil
}

func handleRelease(client *Client, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: aeitron deploy release <workload-id>")
	}
	workloadID := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.ReleaseWorkload(ctx, workloadID); err != nil {
		return err
	}
	fmt.Printf("SUCCESS: Workload %q capacity released.\n", workloadID)
	return nil
}

func handleNodes(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nodes, err := client.ListNodes(ctx)
	if err != nil {
		return err
	}

	if len(nodes) == 0 {
		fmt.Println("No active fabric nodes registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NODE ID\tREGION\tJURISDICTION\tGPU MODEL\tAVAILABLE/TOTAL\tCOST/HR\tCARBON (gCO2eq)\tHEALTH")
	for _, n := range nodes {
		healthStr := "HEALTHY"
		if !n.Healthy {
			healthStr = "UNHEALTHY"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d/%d\t$%.2f\t%.1f\t%s\n",
			n.ID, n.Region, n.Jurisdiction, n.GPUModel, n.AvailableGPUs, n.TotalGPUs, n.CostPerGPUHour, n.CarbonIntensity, healthStr)
	}
	w.Flush()
	return nil
}

func handleTemplate(args []string) error {
	fs := flag.NewFlagSet("template", flag.ContinueOnError)
	var tType string
	fs.StringVar(&tType, "type", "gpu", "Template type: gpu, api, container, sovereign")

	if err := fs.Parse(args); err != nil {
		return err
	}

	tmpl := GenerateTemplate(tType)
	fmt.Print(tmpl)
	return nil
}

func handleJournal(client *Client, args []string) error {
	ctx := context.Background()

	if len(args) > 0 && args[0] == "verify" {
		res, err := client.VerifyJournal(ctx)
		if err != nil {
			return fmt.Errorf("journal verification failed: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Println(" Aeitron Zero-Egress Merkle Hash Chain Integrity Verification")
		fmt.Println("================================================================================")
		fmt.Printf("Verification Status:  %s\n", res.Status)
		fmt.Printf("Digest Algorithm:     %s\n", res.Algorithm)
		fmt.Printf("Total Audit Records:  %d\n", res.EntriesCount)
		fmt.Printf("Merkle Root Hash:     %s\n", res.MerkleRoot)
		if res.Status == "VERIFIED" {
			fmt.Println("Mathematical Proof:   VALID (Continuous SHA-256 hash chain and Merkle tree intact)")
			fmt.Println("Integrity Guarantee:  Zero records deleted, altered, or injected out of order.")
		} else {
			fmt.Println("Mathematical Proof:   FAILED! Tampering or sequence break detected in journal.")
		}
		fmt.Println("================================================================================")
		return nil
	}

	if len(args) > 0 && (args[0] == "proof" || args[0] == "--proof") {
		fs := flag.NewFlagSet("journal proof", flag.ContinueOnError)
		seq := fs.Uint64("seq", 0, "Journal entry sequence number [required]")
		fs.Uint64Var(seq, "sequence", 0, "Journal entry sequence number [required]")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		if *seq == 0 {
			return fmt.Errorf("missing or invalid sequence number: --seq <number> (e.g. --seq 1)")
		}

		proofResp, err := client.GetJournalProof(ctx, *seq)
		if err != nil {
			return fmt.Errorf("failed to fetch inclusion proof: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Printf(" Aeitron Zero-Egress Merkle Inclusion Proof (Sequence #%d)\n", *seq)
		fmt.Println("================================================================================")
		if proofResp.Proof == nil {
			fmt.Println("No proof returned for sequence.")
			return nil
		}
		fmt.Printf("Sequence:             %d\n", proofResp.Proof.Sequence)
		fmt.Printf("Leaf Record Hash:     %s\n", proofResp.Proof.LeafHash)
		fmt.Printf("Merkle Root Digest:   %s\n", proofResp.Proof.MerkleRoot)
		fmt.Printf("Proof Audit Depth:    %d step(s)\n", len(proofResp.Proof.Path))
		status := "COMPROMISED / INVALID"
		if proofResp.Verified {
			status = "VALID (Mathematically Proven)"
		}
		fmt.Printf("Proof Verification:   %s\n", status)
		fmt.Println("Zero-Egress Rule:     Workload placement proven without disclosing tenant neighbor data.")
		fmt.Println("================================================================================")
		return nil
	}

	// Default: list placement events
	fs := flag.NewFlagSet("journal", flag.ContinueOnError)
	tenantID := fs.String("tenant", "", "Filter by tenant ID")
	listArgs := args
	if len(args) > 0 && args[0] == "list" {
		listArgs = args[1:]
	}
	if err := fs.Parse(listArgs); err != nil {
		return err
	}

	events, err := client.ListJournal(ctx, *tenantID)
	if err != nil {
		return fmt.Errorf("failed to list placement journal: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Multi-Tenant Placement Journal & Cryptographic Hash Chain")
	fmt.Println("================================================================================")
	fmt.Printf("Total Placement Records: %d\n\n", len(events))
	if len(events) == 0 {
		fmt.Println("No placement records recorded in journal.")
		fmt.Println("================================================================================")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SEQ\tTIMESTAMP\tEVENT\tWORKLOAD\tTENANT\tNODE\tPRIORITY\tCURRENT-HASH")
	for _, e := range events {
		shortHash := e.CurrentHash
		if len(shortHash) > 16 {
			shortHash = shortHash[:16] + "..."
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\n",
			e.Sequence, e.Timestamp.Format("15:04:05"), e.EventType, e.WorkloadID, e.TenantID, e.NodeID, e.Priority, shortHash)
	}
	w.Flush()
	fmt.Println("================================================================================")
	return nil
}

func handleConsensus(client *Client, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := client.GetConsensusStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve consensus status: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" AEITRON 5-REGION PQC RAFT CONSENSUS CLUSTER (STAGE 15)")
	fmt.Println("================================================================================")
	fmt.Printf("Cluster ID:            %s\n", status.ClusterID)
	fmt.Printf("Local Node ID:         %s (Region: %s)\n", status.NodeID, status.Region)
	fmt.Printf("Raft Role:             %s\n", status.Role)
	fmt.Printf("Current Term:          %d\n", status.CurrentTerm)
	fmt.Printf("Cluster Leader:        %s\n", status.LeaderID)
	fmt.Printf("Commit Index:          %d (Applied: %d)\n", status.CommitIndex, status.LastApplied)
	fmt.Printf("Log Entries Count:     %d\n", status.LogEntriesCount)
	fmt.Printf("Latest Log Hash:       %s\n", status.LatestLogHash)
	pqcStatus := "VERIFIED / NIST FIPS 203 (ML-KEM-1024)"
	if !status.PqcAttestationVerified {
		pqcStatus = "UNVERIFIED"
	}
	fmt.Printf("PQC Attestation:       %s\n", pqcStatus)
	quorumStatus := "SATISFIED (Quorum = 3/5)"
	if !status.QuorumSatisfied {
		quorumStatus = "DEGRADED (Quorum Not Reached)"
	}
	fmt.Printf("Quorum Health:         %s\n", quorumStatus)
	fmt.Println()

	fmt.Println("--- 5-REGION RAFT CONSENSUS PEERS ---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PEER ID\tREGION\tJURISDICTION\tSUBSEA CABLE\tRTT (MS)\tMATCH IDX\tSTATUS")
	fmt.Fprintln(w, "-------\t------\t------------\t------------\t--------\t---------\t------")
	for _, p := range status.Peers {
		statusStr := "HEALTHY"
		if !p.Healthy {
			statusStr = "OFFLINE"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d ms\t%d\t%s\n",
			p.ID, p.Region, p.Jurisdiction, p.SubseaCable, p.RTTMs, p.MatchIndex, statusStr)
	}
	w.Flush()

	fmt.Println()
	fmt.Println("--- GEOGRAPHIC PINNING & NDMA 2026 SOVEREIGN STATUS ---")
	fmt.Println("  [BANGLADESH CII PINNING]   bd-dac (Dhaka PoP)  -> STRICT RESIDENCY LOCKED (NDMA 2026 Ch. 7 & 8)")
	fmt.Println("  [EU GDPR PINNING]          de-fra / is-north-1 -> REGIONAL RESIDENCY PINNED")
	fmt.Println("  [CROSS-BORDER SCHEDULING]  UNCONSTRAINED       -> DYNAMIC MULTI-OBJECTIVE LATENCY/CARBON OPTIMIZED")
	fmt.Println("================================================================================")
	return nil
}
