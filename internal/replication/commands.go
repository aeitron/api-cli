package replication

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"
)

// PrintReplicationHelp prints available subcommands for multi-region replication.
func PrintReplicationHelp() {
	fmt.Println("Aeitron Multi-Region Sovereign Replication & Database Engine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron [database|replication] <subcommand> [args] [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  status                         Inspect local PoP replication HUD and Merkle tree root")
	fmt.Println("  database [status|sync]         Inspect ClickHouse columnar & CockroachDB transactional engine HUD")
	fmt.Println("  sync [--peer <region-id>]      Force an anti-entropy synchronization cycle with peer PoP")
	fmt.Println("  topology                       Display global 5-region replication mesh and zero-egress boundaries")
}

// HandleReplicationCommand routes CLI subcommands for replication and database.
func HandleReplicationCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintReplicationHelp()
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "database", "db":
		return handleDatabaseSubcommand(client, rest)

	case "status":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status, err := client.GetStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get replication status: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Printf(" Aeitron Sovereign Replication HUD: %s\n", status.LocalRegion)
		fmt.Println("================================================================================")
		fmt.Printf("  Total Records:          %d\n", status.TotalRecords)
		fmt.Printf("  Sovereign Records (BD): %d (Air-gapped, zero-egress enforced)\n", status.SovereignCount)
		fmt.Printf("  Global Records:         %d (Replicated across 5 PoPs)\n", status.GlobalCount)
		fmt.Printf("  Merkle Tree Root Hash:  %s\n", status.MerkleRootHash)
		fmt.Printf("  Illegal Egress Blocked: %d attempts\n", status.TotalViolations)
		fmt.Println()
		fmt.Println("Active Peer Synchronizations:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PEER\tNAME\tSTATUS\tRTT\tLAG\tSYNCED\tPARITY\tSOVEREIGN BOUNDARY")
		for _, p := range status.Peers {
			onlineStr := "ONLINE"
			if !p.Online {
				onlineStr = "OFFLINE"
			}
			parityStr := "IN_SYNC"
			if !p.MerkleParity {
				parityStr = "DIVERGED"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%.1fms\t%dms\t%d\t%s\t%s\n",
				p.RegionID, p.Name, onlineStr, p.LatencyMs, p.LagMs, p.RecordsSynced, parityStr, p.SovereignBoundary)
		}
		w.Flush()
		fmt.Println("================================================================================")
		return nil

	case "sync":
		fs := flag.NewFlagSet("replication sync", flag.ContinueOnError)
		peer := fs.String("peer", "sg-sin", "Target peer region ID to synchronize with")
		if err := fs.Parse(rest); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Printf("Initiating anti-entropy Merkle reconciliation with peer %q...\n", *peer)
		synced, err := client.TriggerSync(ctx, *peer)
		if err != nil {
			return fmt.Errorf("sync failed: %w", err)
		}

		fmt.Printf("Replication sync complete: %d records reconciled. Merkle tree verified in-sync.\n", synced)
		return nil

	case "topology":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status, err := client.GetStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get topology: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Println(" Aeitron Global Sovereign Multi-Region Replication Topology (5 PoPs)")
		fmt.Println("================================================================================")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NODE ID\tLOCATION\tROLE\tJURISDICTION\tEGRESS POLICY")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			"bd-dac", "Dhaka, Bangladesh", "Sovereign Primary DC", "Bangladesh NDMA 2026", "AIR_GAPPED (Zero-Egress)")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			"sg-sin", "Singapore Hub", "Global Routing & DNS", "ASEAN Hub", "GLOBAL_POLICIES_ONLY")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			"de-fra", "Frankfurt, Germany", "EU Central & GDPR Gate", "EU GDPR Chapter V", "RESTRICTED (EU Sovereign)")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			"is-north-1", "Reykjavík, Iceland", "100% Green Geo Compute", "Nordic Sovereign", "GLOBAL_POLICIES_ONLY")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			"us-iad", "Virginia, USA", "US East AI Gateway", "US Commercial", "GLOBAL_POLICIES_ONLY")
		w.Flush()
		fmt.Println("================================================================================")
		fmt.Printf("Local Engine: %s | Active Peer Links: %d | Zero-Egress Status: ACTIVE\n", status.LocalRegion, len(status.Peers))
		fmt.Println("================================================================================")
		return nil

	default:
		PrintReplicationHelp()
		return fmt.Errorf("unknown replication subcommand %q", sub)
	}
}

func handleDatabaseSubcommand(client *Client, args []string) error {
	action := "status"
	var rest []string
	if len(args) > 0 {
		action = args[0]
		rest = args[1:]
	}

	switch action {
	case "status":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		status, err := client.GetDatabaseStatus(ctx)
		if err != nil {
			return fmt.Errorf("failed to get database status: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Println(" AEITRON MULTI-REGION SOVEREIGN DATABASE ENGINE (STAGE 16)")
		fmt.Println(" ClickHouse Columnar Analytics + CockroachDB/PostgreSQL Transactional ACID")
		fmt.Println("================================================================================")
		fmt.Printf("Local Node Region:     %s (%s)\n", status.LocalRegion, status.PoPName)
		fmt.Printf("Uptime:                %ds\n", status.UptimeSeconds)
		fmt.Printf("PQC Transit Ratchet:   %s\n", status.PQCAttestation)
		fmt.Printf("Zero-Egress Blocked:   %d attempts (Bangladesh NDMA 2026 Ch. 7 & 8)\n", status.TotalViolations)
		fmt.Println()
		fmt.Println("--- COLUMNAR ANALYTICAL STORE (ClickHouse MergeTree) ---")
		fmt.Printf("  Total Records:          %d (%d compressed blocks)\n", status.AnalyticalTotal, status.AnalyticalBlocks)
		fmt.Printf("  Sovereign Quarantined:  %d (Strictly locked to Dhaka)\n", status.AnalyticalSovereign)
		fmt.Printf("  Global Replicated:      %d (Synchronized across all PoPs)\n", status.AnalyticalGlobal)
		fmt.Println()
		fmt.Println("--- TRANSACTIONAL ACID STORE (Distributed PostgreSQL/CockroachDB) ---")
		fmt.Printf("  Committed Tx:           %d (Aborted: %d)\n", status.TxCommitted, status.TxAborted)
		fmt.Printf("  Latest WAL Sequence:    %d\n", status.LatestWALSeq)
		fmt.Printf("  Merkle Head Hash:       %s\n", status.LatestWALHash)
		fmt.Println()
		fmt.Println("--- 5-REGION UNDERSEA PQC REPLICATION PEERS ---")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PEER ID\tPOP NAME\tCABLE\tRTT (MS)\tLAG (MS)\tANA SYNC\tTX SYNC\tPQC ATTEST\tSTATUS")

		// Sort peer keys
		var peerKeys []string
		for k := range status.Peers {
			peerKeys = append(peerKeys, k)
		}
		sort.Strings(peerKeys)

		for _, k := range peerKeys {
			p := status.Peers[k]
			attestStr := "FAILED"
			if p.PQCAttestationOK {
				attestStr = "VERIFIED"
			}
			statusStr := "IN_SYNC"
			if !p.InSync {
				statusStr = "SYNCING"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%.1f ms\t%d ms\t%d\t%d\t%s\t%s\n",
				p.RegionID, p.PoPName, p.SubseaCable, p.LatencyMs, p.ReplicationLagMs, p.AnalyticalSynced, p.TxSynced, attestStr, statusStr)
		}
		w.Flush()
		fmt.Println("================================================================================")
		return nil

	case "sync":
		fs := flag.NewFlagSet("database sync", flag.ContinueOnError)
		peer := fs.String("peer", "sg-sin", "Target peer region ID to synchronize with")
		if err := fs.Parse(rest); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		fmt.Printf("Initiating quantum-safe undersea database sync with peer %q...\n", *peer)
		res, err := client.SyncDatabase(ctx, *peer)
		if err != nil {
			return fmt.Errorf("database sync failed: %w", err)
		}

		fmt.Printf("Database sync complete with peer %q over subsea cable %v:\n", *peer, res["subsea_cable"])
		fmt.Printf("  Analytical Records Synced:   %v\n", res["analytical_synced"])
		fmt.Printf("  Transactional Rows Synced:   %v\n", res["tx_synced"])
		fmt.Printf("  Sovereign Egress Blocked:    %v violations (NDMA 2026)\n", res["violations_blocked"])
		fmt.Printf("  PQC Attestation:             %v\n", res["pqc_attestation"])
		return nil

	default:
		return fmt.Errorf("unknown database action %q; use 'status' or 'sync'", action)
	}
}
