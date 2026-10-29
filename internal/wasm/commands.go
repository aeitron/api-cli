package wasm

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
)

// PrintWasmHelp prints available WASM plugin management subcommands.
func PrintWasmHelp() {
	fmt.Println("Aeitron Edge-Native WASM Extensibility Manager")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron wasm <subcommand> [args] [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  deploy <path/to/module.wasm> [flags]  Compile and deploy a sandboxed WASM plugin to edge PoPs")
	fmt.Println("  list                                  List all active WASM edge plugins")
	fmt.Println("  delete <plugin-id>                    Remove a WASM edge plugin")
	fmt.Println()
	fmt.Println("Flags for 'deploy':")
	fmt.Println("  --name <string>             Human-readable plugin name")
	fmt.Println("  --domain <string>           Domain filter (e.g. '*.aeitron.io' or 'api.example.com') (default: '*')")
	fmt.Println("  --phase <phase>             Execution phase: RequestHeaders, RequestBody, ResponseHeaders (default: RequestHeaders)")
	fmt.Println("  --memory-limit <bytes>      Linear memory cap in bytes (default: 4194304 / 4MB)")
	fmt.Println("  --timeout <micros>          Execution budget in microseconds (default: 500)")
}

// HandleWasmCommand routes CLI subcommands for WASM.
func HandleWasmCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintWasmHelp()
		return nil
	}

	sub := args[0]
	rest := args[1:]

	switch sub {
	case "deploy":
		if len(rest) == 0 {
			return fmt.Errorf("missing path to WASM module file: aeitron wasm deploy <path.wasm>")
		}
		wasmPath := rest[0]
		fs := flag.NewFlagSet("wasm deploy", flag.ContinueOnError)
		name := fs.String("name", "", "Plugin name")
		domain := fs.String("domain", "*", "Domain filter")
		phase := fs.String("phase", "RequestHeaders", "Execution phase")
		memLimit := fs.Int("memory-limit", 4*1024*1024, "Memory limit in bytes")
		timeout := fs.Uint64("timeout", 500, "Timeout in microseconds")

		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}

		bytecode, err := os.ReadFile(wasmPath)
		if err != nil {
			return fmt.Errorf("failed to read WASM file %q: %w", wasmPath, err)
		}

		if *name == "" {
			*name = wasmPath
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		plugin, err := client.DeployPlugin(ctx, DeployPluginRequest{
			Name:             *name,
			DomainFilter:     *domain,
			Phase:            PluginPhase(*phase),
			MemoryLimitBytes: *memLimit,
			TimeoutMicros:    *timeout,
			WasmBytecode:     bytecode,
		})
		if err != nil {
			return fmt.Errorf("failed to deploy WASM plugin: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Printf(" WASM Edge Plugin Deployed Successfully: %s\n", plugin.ID)
		fmt.Println("================================================================================")
		fmt.Printf("  Name:              %s\n", plugin.Name)
		fmt.Printf("  Domain Filter:     %s\n", plugin.DomainFilter)
		fmt.Printf("  Execution Phase:   %s\n", plugin.Phase)
		fmt.Printf("  Memory Quota:      %d bytes (%.1f MB)\n", plugin.MemoryLimitBytes, float64(plugin.MemoryLimitBytes)/(1024*1024))
		fmt.Printf("  Latency Budget:    %d µs\n", plugin.TimeoutMicros)
		fmt.Printf("  Bytecode SHA-256:  %s\n", plugin.BytecodeSha256)
		fmt.Println("================================================================================")
		return nil

	case "list":
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		plugins, err := client.ListPlugins(ctx)
		if err != nil {
			return fmt.Errorf("failed to list plugins: %w", err)
		}

		fmt.Println("================================================================================")
		fmt.Println(" Aeitron Edge-Native WASM Extensibility Modules")
		fmt.Println("================================================================================")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "PLUGIN ID\tNAME\tDOMAIN\tPHASE\tMEMORY CAP\tBUDGET\tSTATUS")
		for _, p := range plugins {
			status := "ENABLED"
			if !p.Enabled {
				status = "DISABLED"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%dKB\t%dµs\t%s\n",
				p.ID, p.Name, p.DomainFilter, p.Phase, p.MemoryLimitBytes/1024, p.TimeoutMicros, status)
		}
		w.Flush()
		fmt.Println("================================================================================")
		return nil

	case "delete":
		if len(rest) == 0 {
			return fmt.Errorf("missing plugin ID: aeitron wasm delete <plugin-id>")
		}
		pluginID := rest[0]
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.DeletePlugin(ctx, pluginID); err != nil {
			return fmt.Errorf("failed to delete plugin %q: %w", pluginID, err)
		}
		fmt.Printf("WASM plugin %q deleted successfully.\n", pluginID)
		return nil

	default:
		PrintWasmHelp()
		return fmt.Errorf("unknown wasm subcommand %q", sub)
	}
}
