package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aeitron/api-cli/internal/ai"
	"github.com/aeitron/api-cli/internal/billing"
	"github.com/aeitron/api-cli/internal/db"
	"github.com/aeitron/api-cli/internal/dns"
	"github.com/aeitron/api-cli/internal/dr"
	"github.com/aeitron/api-cli/internal/events"
	"github.com/aeitron/api-cli/internal/gpu"
	"github.com/aeitron/api-cli/internal/network"
	"github.com/aeitron/api-cli/internal/replication"
	"github.com/aeitron/api-cli/internal/security"
	"github.com/aeitron/api-cli/internal/storage"
	"github.com/aeitron/api-cli/internal/subsea"
	"github.com/aeitron/api-cli/internal/vm"
	"github.com/aeitron/api-cli/internal/wasm"
	"github.com/aeitron/api-cli/internal/workload"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]
	switch command {
	case "ai":
		client := ai.NewClient()
		if err := ai.HandleAICommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "bgp":
		client := network.NewClient()
		if err := network.HandleBGPCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "network":
		client := network.NewClient()
		if err := network.HandleNetworkCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "mesh":
		client := network.NewClient()
		if err := network.HandleMeshCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "xdp":
		client := network.NewClient()
		if err := network.HandleXDPCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "deploy", "workload", "scheduler":
		client := workload.NewClient()
		if err := workload.HandleDeployCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "dns":
		client := dns.NewClient()
		if err := dns.HandleDNSCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "events", "schemas":
		if err := events.HandleEventsCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "gpu", "vllm", "mig":
		client := gpu.NewClient()
		if err := gpu.HandleGPUCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "vm", "microvm", "largevm":
		client := vm.NewClient()
		if err := vm.HandleVMCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "security":
		client := security.NewClient()
		if err := security.HandleSecurityCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "hsm", "kms":
		client := security.NewClient()
		if err := security.HandleHSMCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "storage", "s3":
		client := storage.NewClient()
		if err := storage.HandleStorageCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "subsea", "cable":
		client := subsea.NewClient()
		if err := subsea.HandleSubseaCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "wasm":
		client := wasm.NewClient()
		if err := wasm.HandleWasmCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "replication":
		client := replication.NewClient()
		if err := replication.HandleReplicationCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "db", "database", "migrate", "migration":
		client := db.NewClient()
		if err := db.HandleDBCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "billing", "bill", "invoice", "invoices":
		client := billing.NewClient()
		if err := billing.HandleBillingCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "dr", "disaster-recovery":
		client := dr.NewClient()
		if err := dr.HandleDRCommand(client, os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "status":
		fmt.Println("================================================================================")
		fmt.Println(" Aeitron Global Infrastructure - Runtime Status")
		fmt.Println("================================================================================")

		netClient := network.NewClient()
		netCtx, netCancel := context.WithTimeout(context.Background(), 2*time.Second)
		pops, netErr := netClient.ListPoPHealth(netCtx)
		netCancel()

		fmt.Println("Edge Anycast Points of Presence (PoPs):")
		if netErr == nil && len(pops) > 0 {
			for _, p := range pops {
				statusStr := "ONLINE"
				if !p.Healthy || p.Status == "POP_DRAINING" {
					statusStr = "DRAINING"
				} else if p.Status == "POP_DEGRADED" {
					statusStr = "DEGRADED"
				}
				failoverInfo := ""
				if p.FailoverPoPID != "" {
					failoverInfo = fmt.Sprintf(", Failover: %s", p.FailoverPoPID)
				}
				fmt.Printf("  %-32s (%s): %s (RTT: %.1fms, Target: %s%s)\n",
					p.Name, p.Region, statusStr, p.LastRTTMs, p.TargetEndpoint, failoverInfo)
			}
		} else if netErr != nil {
			fmt.Printf("  Edge Anycast PoPs: UNAVAILABLE (control plane unreachable: %v)\n", netErr)
		} else {
			fmt.Println("  Edge Anycast PoPs: NO POPS REGISTERED")
		}
		fmt.Println()

		// Query Subsea cable status
		subseaClient := subsea.NewClient()
		subseaCtx, subseaCancel := context.WithTimeout(context.Background(), 2*time.Second)
		cables, subseaErr := subseaClient.GetSubseaStatus(subseaCtx)
		subseaCancel()

		fmt.Println("Subsea Cable Telemetry (RFC 5357 TWAMP):")
		if subseaErr == nil && len(cables) > 0 {
			for _, c := range cables {
				fmt.Printf("  %-16s (%s): %s (RTT: %.1fms, Jitter: %.1fms, Loss: %.1f%%, BGP: %s)\n",
					c.ID, c.Role, c.State, c.RTTMs, c.JitterMs, c.LossPercent, c.BGPCommunity)
			}
		} else if subseaErr != nil {
			fmt.Printf("  Subsea Cables: UNAVAILABLE (%v)\n", subseaErr)
		} else {
			fmt.Println("  Subsea Cables: NO ACTIVE MONITORS")
		}
		fmt.Println()

		// Query Sovereign S3 Storage status
		storageClient := storage.NewClient()
		storageCtx, storageCancel := context.WithTimeout(context.Background(), 2*time.Second)
		buckets, storageErr := storageClient.ListBuckets(storageCtx)
		storageCancel()

		fmt.Println("Sovereign S3 Storage (NDMA 2026):")
		if storageErr == nil {
			fmt.Printf("  Active Buckets:       %d registered\n", len(buckets))
			for _, b := range buckets {
				fmt.Printf("    * s3://%-20s [%s - %s]\n", b.Name, b.Sovereignty, b.PrimaryRegion)
			}
		} else {
			fmt.Printf("  Storage Status:       UNAVAILABLE (%v)\n", storageErr)
		}
		fmt.Println()

		// Attempt querying AI Director status
		client := ai.NewClient()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		analytics, err := client.GetAnalytics(ctx)
		cancel()
		if err == nil {
			fmt.Printf("AI Traffic Director (%s):\n", analytics.Engine)
			fmt.Printf("  Status:               %s\n", analytics.Status)
			fmt.Printf("  Active Providers:     %d\n", analytics.ProvidersCount)
			fmt.Printf("  Total Tokens Routed:  %d\n", analytics.Economics.TotalTokensRouted)
			fmt.Printf("  Arbitraged Savings:   $%.4f USD (%.1f%% cheaper than Hyperscalers)\n",
				analytics.Economics.CostSavedUSD, analytics.Economics.SavingsPercentage)
		} else {
			fmt.Println("AI Traffic Director:")
			fmt.Printf("  Gateway Status:       UNAVAILABLE (gateway unreachable: %v)\n", err)
		}
		fmt.Println("================================================================================")
	case "version", "--version", "-v":
		fmt.Println("aeitron version 0.1.0-alpha (linux/amd64, windows/amd64, darwin/arm64)")
	default:
		printHelp()
	}
}

func printHelp() {
	fmt.Println("Aeitron Global Infrastructure CLI (aeitron)")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron <command> [subcommand] [flags]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  ai           Manage AI compute providers, run spot arbitrage tests, and inspect token savings")
	fmt.Println("  bgp          Manage BGP-4 Anycast peering sessions, prefix advertisements, and graceful drain")
	fmt.Println("  billing      Manage usage metering, subscription tiers ($1M ARR), invoices, and multi-rail checkout")
	fmt.Println("  deploy       Deploy a universal workload to the global edge fabric")
	fmt.Println("  dns          Manage Anycast DNS zones, records, and autonomous BGP health failover")
	fmt.Println("  dr           Multi-region disaster recovery, tenant isolation, and automated failover")
	fmt.Println("  db           Manage distributed database migrations (CockroachDB/ClickHouse) and seed engine")
	fmt.Println("  events       Inspect NATS JetStream event schemas, CloudEvents contracts, and validation")
	fmt.Println("  gpu          Manage hardware GPU slicing, NVIDIA MIG profiles, and edge vLLM clusters")
	fmt.Println("  mesh         Manage quantum-safe WireGuard inter-PoP mesh, ML-KEM-1024 ratchets, and subsea routing")
	fmt.Println("  network      Manage BGP DFZ policies, RPKI validation, flap damping, and Maglev flow routing")
	fmt.Println("  replication  Inspect multi-region sovereign replication HUD, anti-entropy sync, and topology")
	fmt.Println("  security     Audit Bangladesh NDMA 2026 data residency and configure edge WAF/firewall")
	fmt.Println("  hsm          Inspect PKCS#11 hardware security module, slots, and active cryptographic health")
	fmt.Println("  status       Inspect real-time health and token economics of global edge PoPs")
	fmt.Println("  storage      Manage sovereign S3 buckets, Reed-Solomon chunks, and NDMA 2026 data residency")
	fmt.Println("  subsea       Inspect subsea cable health, RFC 5357 TWAMP active probes, and BGP steering")
	fmt.Println("  version      Print CLI version information")
	fmt.Println("  vm           Manage serverless MicroVMs (<5ms) and enterprise LargeVMs (up to 128 vCPUs)")
	fmt.Println("  wasm         Deploy, list, and manage sandboxed edge-native WebAssembly plugins")
	fmt.Println("  xdp          Inspect line-rate AF_XDP packet counters, volumetric DDoS drops, and hardware offload")
	fmt.Println()
	fmt.Println("Run 'aeitron <command> --help' for command-specific flags and usage.")
}
