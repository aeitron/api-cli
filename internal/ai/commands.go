package ai

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintAIHelp prints the available AI subcommands.
func PrintAIHelp() {
	fmt.Println("Aeitron AI / Compute Traffic Controller")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron ai <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  route              Intelligently route an AI prompt using multi-objective KV & carbon engine")
	fmt.Println("  clusters           Display live telemetry, VRAM, and carbon intensity of GPU clusters")
	fmt.Println("  inject-thermal     Inject live GPU die temperature and power draw to test thermal throttling")
	fmt.Println("  cluster-health     Query detailed thermal status and throttle factor of a cluster")
	fmt.Println("  providers list     List all registered upstream AI compute providers")
	fmt.Println("  providers add      Register a new AI provider in the Control Plane Key Vault")
	fmt.Println("  providers delete   Remove an AI provider by ID")
	fmt.Println("  analytics          Display live token volume, costs, and arbitrage dollar savings")
	fmt.Println("  test               Execute a test prompt to verify multi-strategy routing and failover")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  Run 'aeitron ai <subcommand> --help' for details on specific flags.")
}

// HandleAICommand routes ai subcommands to appropriate handlers.
func HandleAICommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintAIHelp()
		return nil
	}

	subcommand := args[0]
	rest := args[1:]

	switch subcommand {
	case "route":
		return handleRoute(client, rest)
	case "clusters":
		return handleClusters(client, rest)
	case "inject-thermal":
		return handleInjectThermal(client, rest)
	case "cluster-health":
		return handleClusterHealth(client, rest)
	case "providers":
		return handleProviders(client, rest)
	case "analytics":
		return handleAnalytics(client, rest)
	case "test":
		return handleTest(client, rest)
	case "help", "--help", "-h":
		PrintAIHelp()
		return nil
	default:
		return fmt.Errorf("unknown ai subcommand %q; run 'aeitron ai help'", subcommand)
	}
}

func handleProviders(client *Client, args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: aeitron ai providers <list|add|delete> [flags]")
		return nil
	}

	action := args[0]
	rest := args[1:]

	switch action {
	case "list":
		fs := flag.NewFlagSet("ai providers list", flag.ContinueOnError)
		projectID := fs.String("project-id", "proj-default", "Project identifier")
		if err := fs.Parse(rest); err != nil {
			return err
		}

		ctx := context.Background()
		providers, err := client.ListProviders(ctx, *projectID)
		if err != nil {
			return fmt.Errorf("failed to list providers: %w", err)
		}

		fmt.Printf("Registered AI Compute Providers (Project: %s, Total: %d)\n\n", *projectID, len(providers))
		if len(providers) == 0 {
			fmt.Println("No providers registered. Use 'aeitron ai providers add' to register an upstream.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tTYPE\tENDPOINT\tCOST/1M\tJURISDICTION\tPRIORITY\tSTATUS")
		for _, p := range providers {
			status := "ENABLED"
			if !p.Enabled {
				status = "DISABLED"
			}
			jur := p.SovereigntyJurisdiction
			if jur == "" {
				jur = "GLOBAL"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%s\t%d\t%s\n",
				p.ID, p.Name, p.ProviderType, p.Endpoint, p.CostPerMTokens, jur, p.Priority, status)
		}
		return w.Flush()

	case "add":
		fs := flag.NewFlagSet("ai providers add", flag.ContinueOnError)
		id := fs.String("id", "", "Unique provider identifier (e.g. coreweave-spot-01) [required]")
		name := fs.String("name", "", "Human-readable provider name [required]")
		endpoint := fs.String("endpoint", "", "Upstream base URL endpoint [required]")
		providerType := fs.String("type", "", "Provider type: openai, azure, coreweave, vllm, onprem, aws [required]")
		apiKey := fs.String("key", "", "API Key for provider authentication (optional)")
		cost := fs.Float64("cost", 1.0, "Cost per 1M tokens in USD")
		priority := fs.Int("priority", 1, "Priority tier (lower number = higher priority)")
		jurisdiction := fs.String("jurisdiction", "", "Sovereignty jurisdiction code (e.g. BD, US, EU)")
		cooldown := fs.Int("cooldown", 10, "Circuit breaker 429 cooldown period in seconds")
		projectID := fs.String("project-id", "proj-default", "Project identifier")

		if err := fs.Parse(rest); err != nil {
			return err
		}

		if *id == "" || *name == "" || *endpoint == "" || *providerType == "" {
			return fmt.Errorf("missing required flags: --id, --name, --endpoint, and --type are required")
		}

		req := CreateAIProviderRequest{
			ID:                      *id,
			ProjectID:               *projectID,
			Name:                    *name,
			Endpoint:                *endpoint,
			ProviderType:            *providerType,
			APIKey:                  *apiKey,
			CostPerMTokens:          *cost,
			Priority:                *priority,
			SovereigntyJurisdiction: *jurisdiction,
			FailoverCooldownSecs:    *cooldown,
			Enabled:                 true,
		}

		ctx := context.Background()
		created, err := client.AddProvider(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to register provider: %w", err)
		}

		fmt.Printf("Provider registered successfully in Key Vault:\n")
		fmt.Printf("  ID:           %s\n", created.ID)
		fmt.Printf("  Name:         %s\n", created.Name)
		fmt.Printf("  Type:         %s\n", created.ProviderType)
		fmt.Printf("  Endpoint:     %s\n", created.Endpoint)
		fmt.Printf("  Cost/1M:      $%.2f\n", created.CostPerMTokens)
		fmt.Printf("  Jurisdiction: %s\n", created.SovereigntyJurisdiction)
		fmt.Printf("  Priority:     %d\n", created.Priority)
		return nil

	case "delete":
		fs := flag.NewFlagSet("ai providers delete", flag.ContinueOnError)
		id := fs.String("id", "", "Provider ID to remove [required]")
		if err := fs.Parse(rest); err != nil {
			return err
		}

		if *id == "" {
			return fmt.Errorf("missing required flag: --id is required")
		}

		ctx := context.Background()
		if err := client.DeleteProvider(ctx, *id); err != nil {
			return fmt.Errorf("failed to delete provider: %w", err)
		}
		fmt.Printf("Provider %q deleted successfully from Key Vault.\n", *id)
		return nil

	default:
		return fmt.Errorf("unknown action %q for providers; run 'aeitron ai providers'", action)
	}
}

func handleAnalytics(client *Client, args []string) error {
	ctx := context.Background()
	analytics, err := client.GetAnalytics(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve live analytics: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Universal AI Traffic Director - Live Economics & Health")
	fmt.Println("================================================================================")
	fmt.Printf("Status:               %s\n", analytics.Status)
	fmt.Printf("Director Engine:      %s\n", analytics.Engine)
	fmt.Printf("Snapshot Timestamp:   %s\n", analytics.Timestamp)
	fmt.Println()
	fmt.Println("--- Real-Time Economics Summary ---")
	fmt.Printf("  Total Tokens Routed:       %d\n", analytics.Economics.TotalTokensRouted)
	fmt.Printf("  Prompt / Completion:       %d / %d\n", analytics.Economics.PromptTokens, analytics.Economics.CompletionTokens)
	fmt.Printf("  Actual Cost Spent:         $%.4f USD\n", analytics.Economics.CostSpentUSD)
	fmt.Printf("  Arbitraged Dollar Savings: $%.4f USD\n", analytics.Economics.CostSavedUSD)
	fmt.Printf("  Cost Savings Percentage:   %.1f%%\n", analytics.Economics.SavingsPercentage)
	fmt.Println()

	fmt.Println("--- Connected Compute Providers Telemetry ---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tJURISDICTION\tREQS\t429S\tFAILOVERS\tTTFT(ms)\tCOST/1M\tHEALTH")
	for _, p := range analytics.Providers {
		health := "HEALTHY"
		if !p.IsAvailable {
			health = "BACKOFF_COOLDOWN"
		}
		jur := p.Jurisdiction
		if jur == "" {
			jur = "GLOBAL"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t%.1f\t$%.2f\t%s\n",
			p.ID, p.Type, jur, p.TotalRequests, p.RateLimited429, p.FailoversTriggered, p.AvgTTFTMs, p.CostPerMTokens, health)
	}
	w.Flush()
	fmt.Println("================================================================================")
	return nil
}

func handleTest(client *Client, args []string) error {
	fs := flag.NewFlagSet("ai test", flag.ContinueOnError)
	model := fs.String("model", "llama-3.3-70b", "Model identifier to request")
	prompt := fs.String("prompt", "Where should this AI workload run right now?", "Input prompt to test")
	strategy := fs.String("strategy", "cost", "Routing strategy: cost, latency, balanced, priority")
	jurisdiction := fs.String("jurisdiction", "", "Sovereignty jurisdiction filter (e.g. BD, US, EU)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	fmt.Println("Executing test inference dispatch via Aeitron Universal AI Director...")
	fmt.Printf("  Model:        %s\n", *model)
	fmt.Printf("  Strategy:     %s\n", *strategy)
	if *jurisdiction != "" {
		fmt.Printf("  Jurisdiction: %s\n", *jurisdiction)
	}
	fmt.Printf("  Prompt:       %q\n", *prompt)
	fmt.Println()

	ctx := context.Background()
	result, err := client.TestInference(ctx, *model, *prompt, *strategy, *jurisdiction)
	if err != nil {
		return fmt.Errorf("test dispatch failed: %w", err)
	}

	fmt.Println("--- Inference Dispatch Results ---")
	fmt.Printf("  HTTP Status:               %d\n", result.StatusCode)
	fmt.Printf("  Selected Compute Provider: %s\n", result.ProviderID)
	fmt.Printf("  Active Routing Strategy:   %s\n", result.RoutingStrategy)
	if result.CostPerMTokens != "" {
		fmt.Printf("  Provider Cost / 1M Tokens: $%s\n", result.CostPerMTokens)
	}
	if result.TokensTotal != "" {
		fmt.Printf("  Total Tokens Extracted:    %s\n", result.TokensTotal)
	}
	if result.EstimatedSavingsUSD != "" {
		fmt.Printf("  Estimated Dollar Savings:  $%s USD\n", result.EstimatedSavingsUSD)
	}
	fmt.Printf("  Time To First Token (TTFT): %s ms\n", result.TTFTMs)
	fmt.Printf("  Transparent Failovers:     %s\n", result.FailoverCount)
	fmt.Printf("  Total End-to-End Duration: %s ms\n", result.TotalDurationMs)
	fmt.Println()
	fmt.Println("--- Upstream Response Preview ---")
	fmt.Println(result.ResponseContent)
	fmt.Println("---------------------------------")
	return nil
}

func handleRoute(client *Client, args []string) error {
	fs := flag.NewFlagSet("ai route", flag.ContinueOnError)
	prompt := fs.String("prompt", "", "Prompt to intelligently route [required]")
	model := fs.String("model", "llama-3.3-70b", "Requested model architecture")
	origin := fs.String("origin", "dhaka", "Client origin PoP region (dhaka, singapore, frankfurt, virginia)")
	strategy := fs.String("strategy", "balanced", "Routing strategy: balanced, latency, cost, green, kv_cache")
	jurisdiction := fs.String("jurisdiction", "", "Sovereignty jurisdiction filter (e.g. BD, US, EU)")
	green := fs.Bool("green", false, "Shortcut to prioritize green clean power")
	maxLat := fs.Int("max-latency", 0, "Max acceptable latency P95 in ms")
	maxCost := fs.Float64("max-cost", 0.0, "Max acceptable hourly cost in USD")
	tenantID := fs.String("tenant-id", "tenant-default", "Tenant identifier")
	projectID := fs.String("project-id", "proj-default", "Project identifier")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *prompt == "" {
		return fmt.Errorf("missing required flag: --prompt is required")
	}

	activeStrategy := *strategy
	if *green {
		activeStrategy = "green"
	}

	req := AIRouteRequest{
		Model:                *model,
		Prompt:               *prompt,
		ClientOrigin:         *origin,
		RequiredJurisdiction: *jurisdiction,
		Strategy:             activeStrategy,
		MaxLatencyP95Ms:      *maxLat,
		MaxCostPerHour:       *maxCost,
		TenantID:             *tenantID,
		ProjectID:            *projectID,
	}

	ctx := context.Background()
	resp, err := client.RouteInference(ctx, req)
	if err != nil {
		return fmt.Errorf("ai routing failed: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Multi-Objective AI Inference & Energy Router")
	fmt.Println("================================================================================")
	fmt.Printf("Status:               %s\n", resp.Status)
	if resp.Status == "UNSCHEDULABLE" {
		fmt.Printf("Rejection Reason:     %s\n", resp.Reason)
		fmt.Println("================================================================================")
		return nil
	}

	fmt.Printf("Assigned Cluster:     %s (%s)\n", resp.AssignedClusterID, resp.ClusterName)
	fmt.Printf("Region / Provider:    %s / %s\n", resp.Region, resp.Provider)
	fmt.Printf("Data Jurisdiction:    %s\n", resp.Jurisdiction)
	fmt.Printf("Measured RTT:         %d ms\n", resp.MeasuredRTTMs)
	fmt.Printf("Estimated TTFT:       %.1f ms\n", resp.EstimatedTTFTMs)
	fmt.Println()
	fmt.Println("--- KV-Cache Locality ---")
	fmt.Printf("  KV-Cache Hit:              %t\n", resp.KVCacheHit)
	fmt.Printf("  Cache Warmth Factor:       %.2f\n", resp.KVCacheWarmth)
	fmt.Printf("  Prefill Latency Reduction: %.1f%%\n", resp.PrefillLatencyReductionPercent)
	fmt.Println()
	fmt.Println("--- Power, Carbon & Economics ---")
	fmt.Printf("  Grid Carbon Intensity:     %.1f gCO2/kWh\n", resp.CarbonGCO2PerKWh)
	fmt.Printf("  Power Classification:      %s\n", resp.CarbonClassification)
	fmt.Printf("  Hourly GPU Compute Cost:   $%.2f USD/hour\n", resp.HourlyCost)
	fmt.Printf("  Multi-Objective Score:     %.2f\n", resp.OptimizationScore)
	fmt.Printf("  Decision Rationale:        %s\n", resp.Reason)
	fmt.Println("================================================================================")
	return nil
}

func handleClusters(client *Client, args []string) error {
	fs := flag.NewFlagSet("ai clusters", flag.ContinueOnError)
	region := fs.String("region", "", "Filter by region")
	health := fs.Bool("health", false, "Display detailed thermal and health telemetry")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ctx := context.Background()
	clusters, err := client.ListClusters(ctx)
	if err != nil {
		return fmt.Errorf("failed to list GPU clusters: %w", err)
	}

	fmt.Printf("Aeitron Global Edge GPU Clusters (Total: %d)\n\n", len(clusters))
	if len(clusters) == 0 {
		fmt.Println("No GPU clusters registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if *health {
		fmt.Fprintln(w, "ID\tNAME\tREGION\tSTATUS\tTEMP\tPOWER\tMAX-TDP\tTHROTTLE\tLAST-CHECK")
		for _, c := range clusters {
			if *region != "" && !strings.EqualFold(c.Region, *region) {
				continue
			}
			status := c.HealthStatus
			if status == "" {
				status = "NOMINAL"
			}
			throttleStr := fmt.Sprintf("%.2fx", c.ThrottleFactor)
			if c.ThrottleFactor == 0 {
				throttleStr = "1.00x"
			}
			lastCheck := "-"
			if !c.LastHealthCheck.IsZero() {
				lastCheck = c.LastHealthCheck.Format("15:04:05")
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.1f°C\t%.0fW\t%.0fW\t%s\t%s\n",
				c.ID, c.Name, c.Region, status, c.TemperatureCelsius, c.PowerWatts, c.MaxTDPWatts, throttleStr, lastCheck)
		}
		return w.Flush()
	}

	fmt.Fprintln(w, "ID\tNAME\tREGION\tJUR\tGPUS\tCOST/H\tCARBON\tKV-USED\tHEALTH\tMODELS")
	for _, c := range clusters {
		if *region != "" && !strings.EqualFold(c.Region, *region) {
			continue
		}
		models := strings.Join(c.ActiveModels, ",")
		if len(models) > 24 {
			models = models[:21] + "..."
		}
		kvMB := fmt.Sprintf("%d/%d MB", c.KVCacheUsedMB, c.KVCacheCapacityMB)
		healthStatus := c.HealthStatus
		if healthStatus == "" {
			healthStatus = "NOMINAL"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d/%d\t$%.2f\t%.0f g\t%s\t%s\t%s\n",
			c.ID, c.Name, c.Region, c.Jurisdiction, c.AvailableGPUs, c.TotalGPUs, c.CostPerGPUHour, c.CarbonIntensity, kvMB, healthStatus, models)
	}
	return w.Flush()
}

func handleInjectThermal(client *Client, args []string) error {
	fs := flag.NewFlagSet("ai inject-thermal", flag.ContinueOnError)
	clusterID := fs.String("cluster", "", "Target GPU cluster ID (e.g. bd-dhaka-dgx01, us-cw-h100-iad) [required]")
	temp := fs.Float64("temp", 0, "GPU die temperature in degrees Celsius [required]")
	power := fs.Float64("power", 0, "GPU power draw in Watts [required]")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *clusterID == "" {
		return fmt.Errorf("missing required flag: --cluster <id>")
	}
	if *temp <= 0 {
		return fmt.Errorf("missing or invalid temperature: --temp must be > 0 (e.g. --temp 86.5)")
	}
	if *power <= 0 {
		return fmt.Errorf("missing or invalid power draw: --power must be > 0 (e.g. --power 720.0)")
	}

	ctx := context.Background()
	resp, err := client.IngestClusterTelemetry(ctx, *clusterID, *temp, *power)
	if err != nil {
		return fmt.Errorf("failed to inject thermal telemetry: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Edge Cluster Thermal & Power Telemetry Injection")
	fmt.Println("================================================================================")
	fmt.Printf("Cluster ID:            %s\n", resp.ClusterID)
	fmt.Printf("Die Temperature:       %.1f °C\n", resp.TemperatureCelsius)
	fmt.Printf("Power Consumption:     %.1f W\n", resp.PowerWatts)
	fmt.Printf("Thermal Health State:  %s\n", resp.HealthStatus)
	fmt.Printf("Clock Throttle Factor: %.2fx (Nominal: 1.00x)\n", resp.ThrottleFactor)
	fmt.Printf("Audit Event Recorded:  %t\n", resp.EventRecorded)
	switch resp.HealthStatus {
	case "NOMINAL":
		fmt.Println("State Description:     Operating within thermal envelope. Full clock speeds active.")
	case "DEGRADED":
		fmt.Println("State Description:     Minor thermal warning. 15% clock step-down, +30 routing penalty.")
	case "THROTTLED":
		fmt.Println("State Description:     AGGRESSIVE THERMAL THROTTLE! 35% clock step-down, +150 routing penalty.")
		fmt.Println("Action:                Traffic Router auto-drains inference requests to cooler global clusters.")
	case "EMERGENCY_SHUTDOWN":
		fmt.Println("State Description:     CRITICAL THERMAL RUNAWAY! Emergency hardware protection trip.")
		fmt.Println("Action:                Cluster completely removed from global routing table.")
	}
	fmt.Println("================================================================================")
	return nil
}

func handleClusterHealth(client *Client, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: aeitron ai cluster-health <cluster-id>")
	}
	clusterID := args[0]

	ctx := context.Background()
	h, err := client.GetClusterHealth(ctx, clusterID)
	if err != nil {
		return fmt.Errorf("failed to query cluster health: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Edge GPU Cluster Thermal State & Telemetry")
	fmt.Println("================================================================================")
	fmt.Printf("Cluster ID:            %s\n", h.ClusterID)
	fmt.Printf("Health State:          %s\n", h.HealthStatus)
	fmt.Printf("Die Temperature:       %.1f °C\n", h.TemperatureCelsius)
	if h.MaxTDPWatts > 0 {
		fmt.Printf("Power Draw:            %.1f W / %.1f W Max TDP (%.1f%%)\n",
			h.PowerWatts, h.MaxTDPWatts, (h.PowerWatts/h.MaxTDPWatts)*100.0)
	} else {
		fmt.Printf("Power Draw:            %.1f W\n", h.PowerWatts)
	}
	fmt.Printf("Clock Throttle Factor: %.2fx\n", h.ThrottleFactor)
	if !h.LastHealthCheck.IsZero() {
		fmt.Printf("Last Evaluated:        %s\n", h.LastHealthCheck.Format(time.RFC3339))
	}
	fmt.Println("================================================================================")
	return nil
}
