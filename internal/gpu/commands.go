package gpu

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
)

// PrintGPUHelp displays available GPU management commands.
func PrintGPUHelp() {
	fmt.Println("Aeitron Edge GPU Slicing & vLLM Cluster Controller")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron gpu <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  nodes, list         List all physical GPU accelerators across Edge PoPs")
	fmt.Println("  overview            Display aggregate GPU cluster capacity, power, and PQC attestation")
	fmt.Println("  slices              List active MIG partitions and time-sliced vGPU allocations")
	fmt.Println("  allocate [flags]    Allocate a physical MIG slice or temporal compute fraction")
	fmt.Println("  release <slice-id>  Decommission a GPU slice with forced HBM memory zeroization")
	fmt.Println("  vllm                Inspect supervised vLLM distributed inference engine instances")
	fmt.Println()
	fmt.Println("Flags for allocate:")
	fmt.Println("  --tenant <id>       Tenant ID owning the allocation")
	fmt.Println("  --workload <id>     Unique workload or service identifier")
	fmt.Println("  --pop <region>      Target Edge PoP (bd-dac, sg-sin, de-fra, is-north-1, us-iad)")
	fmt.Println("  --type <type>       Allocation type: MIG_HARDWARE or TEMPORAL_TIME_SLICE (default: MIG_HARDWARE)")
	fmt.Println("  --profile <name>    NVIDIA MIG profile: 1g.10gb, 2g.20gb, 3g.40gb, 7g.80gb (default: 1g.10gb)")
	fmt.Println("  --fraction <val>    Time-slice compute fraction (0.10 to 1.00) (default: 0.25)")
	fmt.Println("  --model <name>      Target AI model name (e.g. 'deepseek-ai/DeepSeek-R1-Distill-Qwen-8B')")
	fmt.Println("  --sovereignty <val> Sovereignty mode: DATA_JURISDICTION_BANGLADESH or DATA_JURISDICTION_GLOBAL")
}

// HandleGPUCommand routes GPU subcommands.
func HandleGPUCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintGPUHelp()
		return nil
	}

	sub := args[0]
	switch sub {
	case "nodes", "list":
		return handleListNodes(client)
	case "overview":
		return handleOverview(client)
	case "slices":
		return handleListSlices(client)
	case "allocate":
		return handleAllocate(client, args[1:])
	case "release":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron gpu release <slice-id>")
		}
		return handleRelease(client, args[1])
	case "vllm":
		return handleListVLLM(client)
	case "--help", "-h", "help":
		PrintGPUHelp()
		return nil
	default:
		return fmt.Errorf("unknown gpu subcommand '%s'. Run 'aeitron gpu --help'", sub)
	}
}

func handleOverview(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	overview, err := client.GetOverview(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch GPU overview: %w", err)
	}

	fmt.Println("=========================================================================================")
	fmt.Println(" Aeitron Edge GPU Accelerator & vLLM Cluster Overview")
	fmt.Println("=========================================================================================")
	fmt.Printf("Total GPU Devices:          %d accelerators\n", overview.TotalDevices)
	fmt.Printf("Total VRAM Memory:          %d MB (%.1f GB)\n", overview.TotalMemoryMB, float64(overview.TotalMemoryMB)/1024.0)
	fmt.Printf("Allocated VRAM:             %d MB (%.1f GB)\n", overview.AllocatedMemoryMB, float64(overview.AllocatedMemoryMB)/1024.0)
	fmt.Printf("Available VRAM:             %d MB (%.1f GB)\n", overview.AvailableMemoryMB, float64(overview.AvailableMemoryMB)/1024.0)
	fmt.Printf("Active Slices:              %d active partitions\n", overview.TotalSlices)
	fmt.Printf("Supervised vLLM Engines:    %d serving instances\n", overview.ActiveVLLMInstances)
	fmt.Printf("Average Temperature:        %.1f °C\n", overview.AverageTemperatureC)
	fmt.Printf("Total Draw Power:           %.1f W\n", overview.TotalPowerWatts)
	fmt.Printf("NDMA 2026 Sovereign Nodes:  %d pinned\n", overview.NDMASovereignDevices)
	fmt.Printf("PQC Lease Attestation:      %s (ML-KEM-1024)\n", overview.PQCAttestation)
	fmt.Println("=========================================================================================")
	return nil
}

func handleListNodes(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	devices, err := client.ListDevices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list GPU nodes: %w", err)
	}

	fmt.Println("=========================================================================================")
	fmt.Println(" Aeitron Edge PoP Physical GPU Accelerators")
	fmt.Println("=========================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEVICE ID\tNODE\tPOP\tMODEL\tMEMORY (MB)\tMIG\tTEMP\tPOWER\tSOVEREIGNTY")
	fmt.Fprintln(w, "---------\t----\t---\t-----\t-----------\t---\t----\t-----\t-----------")

	for _, d := range devices {
		migStr := "DISABLED"
		if d.MIGEnabled {
			migStr = "ENABLED"
		}
		sovStr := "GLOBAL"
		if d.SovereignLocked {
			sovStr = "PINNED (bd-dac)"
		}
		memStr := fmt.Sprintf("%d / %d", d.AllocatedMemoryMB, d.TotalMemoryMB)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%.1fC\t%.0fW\t%s\n",
			d.ID, d.NodeID, d.Region, d.Model, memStr, migStr, d.TemperatureCelsius, d.PowerWatts, sovStr)
	}
	w.Flush()
	return nil
}

func handleListSlices(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slices, err := client.ListSlices(ctx)
	if err != nil {
		return fmt.Errorf("failed to list GPU slices: %w", err)
	}

	fmt.Println("=========================================================================================")
	fmt.Println(" Aeitron Active GPU Slices & Partitions (NVIDIA MIG & Time-Slicing)")
	fmt.Println("=========================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SLICE ID\tPARENT DEVICE\tPOP\tTYPE\tPROFILE/FRAC\tMEMORY\tTENANT\tMODEL\tSTATUS")
	fmt.Fprintln(w, "--------\t-------------\t---\t----\t------------\t------\t------\t-----\t------")

	for _, s := range slices {
		profStr := s.MIGProfile
		if s.AllocationType == "TEMPORAL_TIME_SLICE" {
			profStr = fmt.Sprintf("%.0f%% compute", s.TimeSliceFraction*100)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%dMB\t%s\t%s\t%s\n",
			s.SliceID, s.ParentDeviceID, s.Region, s.AllocationType, profStr, s.MemoryMB, s.TenantID, s.ModelName, s.Status)
	}
	w.Flush()
	return nil
}

func handleAllocate(client *Client, args []string) error {
	fs := flag.NewFlagSet("allocate", flag.ContinueOnError)
	tenant := fs.String("tenant", "tenant-default", "Tenant ID")
	workload := fs.String("workload", "wl-inference-01", "Workload ID")
	pop := fs.String("pop", "bd-dac", "Edge PoP (bd-dac, sg-sin, de-fra, is-north-1, us-iad)")
	allocType := fs.String("type", "MIG_HARDWARE", "MIG_HARDWARE or TEMPORAL_TIME_SLICE")
	profile := fs.String("profile", "1g.10gb", "MIG profile")
	fraction := fs.Float64("fraction", 0.25, "Time slice compute fraction")
	model := fs.String("model", "deepseek-ai/DeepSeek-R1-Distill-Qwen-8B", "Target AI model")
	sov := fs.String("sovereignty", "DATA_JURISDICTION_BANGLADESH", "Sovereignty jurisdiction")

	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	slice, err := client.AllocateSlice(ctx, AllocationRequest{
		TenantID:          *tenant,
		WorkloadID:        *workload,
		Region:            *pop,
		AllocationType:    *allocType,
		MIGProfile:        *profile,
		TimeSliceFraction: *fraction,
		ModelName:         *model,
		Sovereignty:       *sov,
	})
	if err != nil {
		return fmt.Errorf("slice allocation failed: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" GPU Slice Allocated Successfully")
	fmt.Println("================================================================================")
	fmt.Printf("  Slice ID:       %s\n", slice.SliceID)
	fmt.Printf("  Parent Device:  %s (PoP: %s)\n", slice.ParentDeviceID, slice.Region)
	fmt.Printf("  Type:           %s\n", slice.AllocationType)
	if slice.AllocationType == "MIG_HARDWARE" {
		fmt.Printf("  MIG Profile:    %s\n", slice.MIGProfile)
	} else {
		fmt.Printf("  Time Fraction:  %.1f%% compute\n", slice.TimeSliceFraction*100)
	}
	fmt.Printf("  Dedicated VRAM: %d MB (%.1f GB)\n", slice.MemoryMB, float64(slice.MemoryMB)/1024.0)
	fmt.Printf("  Tenant ID:      %s\n", slice.TenantID)
	fmt.Printf("  Model:          %s\n", slice.ModelName)
	fmt.Printf("  PQC Attest:     %s (Verified ML-KEM-1024)\n", slice.PQCLeaseToken)
	fmt.Printf("  Status:         %s\n", slice.Status)
	fmt.Println("================================================================================")
	return nil
}

func handleRelease(client *Client, sliceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.ReleaseSlice(ctx, sliceID); err != nil {
		return fmt.Errorf("failed to release slice '%s': %w", sliceID, err)
	}

	fmt.Println("================================================================================")
	fmt.Printf(" GPU Slice '%s' Decommissioned\n", sliceID)
	fmt.Println("================================================================================")
	fmt.Println("  Status:          RELEASED")
	fmt.Println("  Memory Security: ZEROIZED (High-Bandwidth Memory purged under NDMA 2026)")
	fmt.Println("  Device VRAM:     Successfully reclaimed to cluster free pool")
	fmt.Println("================================================================================")
	return nil
}

func handleListVLLM(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	instances, err := client.ListVLLMInstances(ctx)
	if err != nil {
		return fmt.Errorf("failed to list vLLM instances: %w", err)
	}

	fmt.Println("=========================================================================================")
	fmt.Println(" Aeitron Edge vLLM Distributed Serving Instances")
	fmt.Println("=========================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INSTANCE ID\tSLICE ID\tPOP\tMODEL\tPORT\tKV-CACHE\tTOKENS/SEC\tP95 LATENCY\tSTATE")
	fmt.Fprintln(w, "-----------\t--------\t---\t-----\t----\t--------\t----------\t-----------\t-----")

	for _, inst := range instances {
		kvStr := fmt.Sprintf("%.1f%% (%dMB)", inst.KVCacheUsagePercent, inst.KVMemoryMB)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%.1f tok/s\t%.1f ms\t%s\n",
			inst.InstanceID, inst.SliceID, inst.Region, inst.Model, inst.Port, kvStr,
			inst.ThroughputTokPerSec, inst.LatencyP95Ms, inst.State)
	}
	w.Flush()
	return nil
}
