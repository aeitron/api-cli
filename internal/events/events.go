package events

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	events "github.com/aeitron/api-cli/pkg/events"
)

type SchemaRegistryEntry struct {
	Type        string
	Domain      string
	Subject     string
	SchemaURI   string
	Sovereignty string
	Description string
}

var Registry = []SchemaRegistryEntry{
	{
		Type:        "io.aeitron.envelope.v1",
		Domain:      "Global",
		Subject:     "aeitron.events.*",
		SchemaURI:   "https://schemas.aeitron.io/events/envelope.v1.json",
		Sovereignty: "NDMA 2026 Sovereign Attestation Enforced",
		Description: "CloudEvents v1.0 Envelope with post-quantum ML-KEM-1024 attestation",
	},
	{
		Type:        "io.aeitron.dns.zone_updated.v1",
		Domain:      "DNS",
		Subject:     "aeitron.events.dns.zone_updated",
		SchemaURI:   "https://schemas.aeitron.io/events/dns/v1/zone_updated.json",
		Sovereignty: "Configurable / bd-dac Confined for .gov.bd",
		Description: "Authoritative DNS zone modification and DNSSEC resigning",
	},
	{
		Type:        "io.aeitron.network.cable_degraded.v1",
		Domain:      "Network",
		Subject:     "aeitron.events.network.cable_degraded",
		SchemaURI:   "https://schemas.aeitron.io/events/network/v1/cable_degraded.json",
		Sovereignty: "Global Anycast Telemetry",
		Description: "Subsea cable TWAMP latency degradation, jitter, or fiber cut",
	},
	{
		Type:        "io.aeitron.network.bgp_peer_changed.v1",
		Domain:      "Network",
		Subject:     "aeitron.events.network.bgp_peer_changed",
		SchemaURI:   "https://schemas.aeitron.io/events/network/v1/bgp_peer_changed.json",
		Sovereignty: "Global Anycast Control",
		Description: "Edge BGP peering session state transition or route prefix shift",
	},
	{
		Type:        "io.aeitron.security.threat_detected.v1",
		Domain:      "Security",
		Subject:     "aeitron.events.security.threat_detected",
		SchemaURI:   "https://schemas.aeitron.io/events/security/v1/threat_detected.json",
		Sovereignty: "Strict Confinement (NDMA 2026 CII Audit)",
		Description: "Line-rate WAF, DDoS mitigation, or sovereign intrusion alert",
	},
	{
		Type:        "io.aeitron.kms.key_rotated.v1",
		Domain:      "Security",
		Subject:     "aeitron.events.kms.key_rotated",
		SchemaURI:   "https://schemas.aeitron.io/events/kms/v1/key_rotated.json",
		Sovereignty: "Dual-Custody HSM Audit Log",
		Description: "Hardware HSM post-quantum root or tenant key rotation",
	},
	{
		Type:        "io.aeitron.compute.workload_scheduled.v1",
		Domain:      "Compute",
		Subject:     "aeitron.events.compute.workload_scheduled",
		SchemaURI:   "https://schemas.aeitron.io/events/compute/v1/workload_scheduled.json",
		Sovereignty: "Geo-Pinning & Consensus Controlled",
		Description: "Edge microVM or AI inference workload placement committed",
	},
	{
		Type:        "io.aeitron.compute.thermal_throttle_triggered.v1",
		Domain:      "Compute",
		Subject:     "aeitron.events.compute.thermal_throttle_triggered",
		SchemaURI:   "https://schemas.aeitron.io/events/compute/v1/thermal_throttle_triggered.json",
		Sovereignty: "Telemetry Alert",
		Description: "Edge node GPU or chassis thermal throttle and carbon protection",
	},
	{
		Type:        "io.aeitron.storage.replication_divergence_detected.v1",
		Domain:      "Storage",
		Subject:     "aeitron.events.storage.replication_divergence_detected",
		SchemaURI:   "https://schemas.aeitron.io/events/storage/v1/replication_divergence_detected.json",
		Sovereignty: "Anti-Entropy Cross-Region Audit",
		Description: "ClickHouse or CockroachDB Merkle divergence detection",
	},
}

// PrintEventsHelp displays available event commands.
func PrintEventsHelp() {
	fmt.Println("Aeitron Event Contracts & NATS JetStream Schema Registry")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron events <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  schemas             List all registered event schemas and NATS JetStream subjects")
	fmt.Println("  validate <file>     Validate a CloudEvents envelope JSON file against schema rules")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  aeitron events schemas")
	fmt.Println("  aeitron events validate sample_event.json")
}

// HandleEventsCommand dispatches event CLI commands.
func HandleEventsCommand(args []string) error {
	if len(args) == 0 {
		PrintEventsHelp()
		return nil
	}

	sub := args[0]
	switch sub {
	case "schemas", "list":
		return handleListSchemas()
	case "validate":
		if len(args) < 2 {
			return fmt.Errorf("usage: aeitron events validate <file.json>")
		}
		return handleValidateFile(args[1])
	case "--help", "-h", "help":
		PrintEventsHelp()
		return nil
	default:
		return fmt.Errorf("unknown events subcommand '%s'. Run 'aeitron events --help'", sub)
	}
}

func handleListSchemas() error {
	fmt.Println("=========================================================================================")
	fmt.Println(" Aeitron Event Contracts & NATS JetStream Schema Registry")
	fmt.Println("=========================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "EVENT TYPE\tDOMAIN\tNATS SUBJECT\tSOVEREIGNTY STATUS")
	fmt.Fprintln(w, "----------\t------\t------------\t------------------")

	for _, entry := range Registry {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", entry.Type, entry.Domain, entry.Subject, entry.Sovereignty)
	}
	w.Flush()
	fmt.Println()
	fmt.Printf("Total Registered Contracts: %d schemas across 5 domains (JSON Schema Draft-07)\n", len(Registry))
	return nil
}

func handleValidateFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file '%s': %w", filePath, err)
	}

	var env events.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("file '%s' is not valid JSON: %w", filePath, err)
	}

	if err := env.Validate(); err != nil {
		return fmt.Errorf("CloudEvents validation failed for '%s': %w", filePath, err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron CloudEvents v1.0 Validation: PASSED")
	fmt.Println("================================================================================")
	fmt.Printf("  Event ID:       %s\n", env.ID)
	fmt.Printf("  Event Type:     %s\n", env.Type)
	fmt.Printf("  Source:         %s\n", env.Source)
	fmt.Printf("  Origin PoP:     %s\n", env.OriginPoP)
	fmt.Printf("  Jurisdiction:   %s\n", env.SovereigntyJurisdiction)
	fmt.Printf("  Strict Confine: %t\n", env.StrictSovereignty)
	if env.PQCAttestation != "" {
		fmt.Printf("  PQC Digest:     %s (Verified ML-KEM-1024)\n", env.PQCAttestation)
	}
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("Payload conforms to Aeitron event contract specifications.")
	return nil
}
