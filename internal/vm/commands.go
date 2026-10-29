package vm

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
)

// PrintVMHelp displays available VM management commands.
func PrintVMHelp() {
	fmt.Println("Aeitron MicroVM & LargeVM Serverless Compute Engine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron vm <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  overview            Display aggregate compute capacity, vCPU usage, and PQC attestation")
	fmt.Println("  nodes               List all physical edge compute hosts across PoPs")
	fmt.Println("  list, instances     List running MicroVMs and LargeVMs with filtering")
	fmt.Println("  get <id>            Inspect detailed runtime configuration, NUMA, and TAP network")
	fmt.Println("  create [flags]      Provision and boot a MicroVM (<5ms) or LargeVM (up to 128 vCPUs)")
	fmt.Println("  action <id> <act>   Trigger lifecycle action (pause, resume, start, stop, reboot)")
	fmt.Println("  delete <id>         Terminate instance with synchronous NDMA 2026 memory zeroization")
	fmt.Println("  pools               Inspect standby pre-warmed snapshot pools")
	fmt.Println()
	fmt.Println("Flags for create:")
	fmt.Println("  --name <val>        Instance descriptive name")
	fmt.Println("  --tenant <id>       Tenant ID owning the instance")
	fmt.Println("  --workload <id>     Workload or task identifier")
	fmt.Println("  --pop <region>      Target Edge PoP (bd-dac, sg-sin, de-fra) (default: bd-dac)")
	fmt.Println("  --tier <tier>       Sizing: nano, micro, small, medium, large, xlarge, 2xlarge, 4xlarge, 8xlarge, baremetal-grade (default: micro)")
	fmt.Println("  --sovereign <mode>  Sovereignty mode: BANGLADESH_NDMA_2026 or STANDARD")
}

// HandleVMCommand routes VM subcommands.
func HandleVMCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintVMHelp()
		return nil
	}

	sub := args[0]
	switch sub {
	case "overview":
		return handleOverview(client)
	case "nodes":
		return handleListNodes(client)
	case "list", "instances":
		return handleListInstances(client, args[1:])
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron vm get <instance-id>")
		}
		return handleGetInstance(client, args[1])
	case "create":
		return handleCreate(client, args[1:])
	case "action":
		if len(args) < 3 {
			return fmt.Errorf("usage: aeitron vm action <instance-id> <pause|resume|start|stop|reboot>")
		}
		return handleAction(client, args[1], args[2])
	case "delete", "destroy", "rm":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron vm delete <instance-id>")
		}
		return handleDelete(client, args[1])
	case "pools":
		return handleListPools(client)
	case "help", "-h", "--help":
		PrintVMHelp()
		return nil
	default:
		return fmt.Errorf("unknown vm subcommand: %s (run 'aeitron vm help')", sub)
	}
}

func handleOverview(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	overview, err := client.GetOverview(ctx)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Edge Virtualization Overview (MicroVMs & LargeVMs)")
	fmt.Println("================================================================================")
	fmt.Printf("Total Instances:          %d\n", overview.TotalVMs)
	fmt.Printf("Active MicroVMs:          %d (Firecracker / Sub-5ms Snapshot Accelerated)\n", overview.ActiveMicroVMs)
	fmt.Printf("Active LargeVMs:          %d (Cloud-Hypervisor / NUMA Pinned & Hugepages)\n", overview.ActiveLargeVMs)
	fmt.Printf("Total Allocated vCPUs:    %d vCPUs\n", overview.TotalAllocatedVCPUs)
	fmt.Printf("Total Allocated RAM:      %d MiB (%.2f GiB)\n", overview.TotalAllocatedMemoryMB, float64(overview.TotalAllocatedMemoryMB)/1024.0)
	fmt.Printf("Pre-Warmed Pool Ready:    %d standby instances\n", overview.PrewarmedPoolReady)
	fmt.Printf("Sovereign Instances:      %d (Confined to bd-dac)\n", overview.SovereignInstances)
	fmt.Printf("PQC Attestation:          %s\n", overview.PQCSovereignAttestation)
	fmt.Println("================================================================================")
	return nil
}

func handleListNodes(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nodes, err := client.ListNodes(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NODE ID\tREGION\tVCPUS (USED/TOT)\tRAM (USED/TOT)\tNUMA\tACTIVE VMS")
	for _, n := range nodes {
		fmt.Fprintf(w, "%s\t%s\t%d / %d\t%.1f / %.1f GiB\t%d\t%d\n",
			n.NodeID,
			n.Region,
			n.AllocatedVCPUs, n.TotalVCPUs,
			float64(n.AllocatedMemoryMB)/1024.0, float64(n.TotalMemoryMB)/1024.0,
			n.NumaSockets,
			n.ActiveVMCount,
		)
	}
	return w.Flush()
}

func handleListInstances(client *Client, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	pop := fs.String("pop", "", "Filter by region")
	tier := fs.String("tier", "", "Filter by tier")
	tenant := fs.String("tenant", "", "Filter by tenant")
	category := fs.String("category", "", "Filter by category (MICRO/LARGE)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	instances, err := client.ListInstances(ctx, *pop, *tier, *tenant, *category)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INSTANCE ID\tNAME\tREGION\tTIER\tCATEGORY\tVCPUS\tRAM\tSTATUS\tBOOT TIME\tSOVEREIGNTY")
	for _, inst := range instances {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d MiB\t%s\t%.1f ms\t%s\n",
			inst.ID,
			inst.Name,
			inst.Region,
			inst.Tier,
			inst.Category,
			inst.VCPUs,
			inst.MemoryMB,
			inst.Status,
			inst.BootTimeMs,
			inst.SovereignClassification,
		)
	}
	return w.Flush()
}

func handleGetInstance(client *Client, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inst, err := client.GetInstance(ctx, id)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Virtual Machine Details: %s\n", inst.ID)
	fmt.Println("================================================================================")
	fmt.Printf("Name:                 %s\n", inst.Name)
	fmt.Printf("Tenant ID:            %s\n", inst.TenantID)
	fmt.Printf("Region / PoP:         %s\n", inst.Region)
	fmt.Printf("Tier & Category:      %s (%s)\n", inst.Tier, inst.Category)
	fmt.Printf("Hypervisor Engine:    %s\n", inst.Hypervisor)
	fmt.Printf("Hardware Alloc:       %d vCPUs, %d MiB RAM (Hugepages: %d MiB)\n", inst.VCPUs, inst.MemoryMB, inst.HugepagesMB)
	fmt.Printf("NUMA Node:            Node %d (Pinned CPUs: %v)\n", inst.NUMANode, inst.PinnedCPUs)
	fmt.Printf("Networking:           %s (%s, Tap: %s)\n", inst.IPAddress, inst.MACAddress, inst.TapDevice)
	fmt.Printf("Status:               %s (Boot Time: %.2f ms)\n", inst.Status, inst.BootTimeMs)
	fmt.Printf("Kernel & Image:       %s (%s)\n", inst.KernelVersion, inst.RootfsPath)
	fmt.Printf("Sovereignty:          %s\n", inst.SovereignClassification)
	fmt.Printf("PQC Lease Token:      %s\n", inst.AttestationToken)
	fmt.Println("================================================================================")
	return nil
}

func handleCreate(client *Client, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	name := fs.String("name", "", "Instance name")
	tenant := fs.String("tenant", "default-tenant", "Tenant ID")
	workload := fs.String("workload", "wl-generic", "Workload ID")
	pop := fs.String("pop", "bd-dac", "Target Edge PoP (bd-dac, sg-sin, de-fra)")
	tier := fs.String("tier", "micro", "Tier sizing: nano, micro, small, medium, large, xlarge, 2xlarge, 4xlarge, 8xlarge, baremetal-grade")
	sovereign := fs.String("sovereign", "STANDARD", "Sovereignty mode: BANGLADESH_NDMA_2026 or STANDARD")
	if err := fs.Parse(args); err != nil {
		return err
	}

	req := CreateVMRequest{
		Name:                    *name,
		TenantID:                *tenant,
		WorkloadID:              *workload,
		Region:                  *pop,
		Tier:                    *tier,
		SovereignClassification: *sovereign,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inst, err := client.CreateVM(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to provision VM: %w", err)
	}

	fmt.Printf("Successfully provisioned %s [%s] in %s\n", inst.ID, inst.Tier, inst.Region)
	fmt.Printf("Category:      %s (Hypervisor: %s)\n", inst.Category, inst.Hypervisor)
	fmt.Printf("Allocated:     %d vCPUs, %d MiB RAM (Boot time: %.2f ms)\n", inst.VCPUs, inst.MemoryMB, inst.BootTimeMs)
	fmt.Printf("Tap Network:   %s -> %s\n", inst.TapDevice, inst.IPAddress)
	fmt.Printf("Attestation:   %s\n", inst.AttestationToken)
	return nil
}

func handleAction(client *Client, id, action string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	inst, err := client.PerformAction(ctx, id, action)
	if err != nil {
		return err
	}

	fmt.Printf("Action '%s' applied to %s. Current status: %s\n", action, inst.ID, inst.Status)
	return nil
}

func handleDelete(client *Client, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.TerminateVM(ctx, id); err != nil {
		return err
	}

	fmt.Printf("Instance %s terminated. NDMA 2026 cryptographic memory zeroization complete (0 bytes remanence).\n", id)
	return nil
}

func handleListPools(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pools, err := client.GetPools(ctx)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "REGION\tTIER\tREADY / TARGET\tFAST BOOT LATENCY")
	for _, p := range pools {
		fmt.Fprintf(w, "%s\t%s\t%d / %d\t< 5.0 ms\n", p.Region, p.Tier, p.ReadyCount, p.TargetCount)
	}
	return w.Flush()
}
