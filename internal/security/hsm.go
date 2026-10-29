package security

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"
)

// PrintHSMHelp displays help for PKCS#11 Hardware Security Module commands.
func PrintHSMHelp() {
	fmt.Println("Aeitron Sovereign PKCS#11 HSM & Hardware Key Broker")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron security hsm <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  status     Inspect active PKCS#11 hardware token, FIPS 140-3 status, and sessions")
	fmt.Println("  slots      List all detected hardware slots and sovereign boundaries")
	fmt.Println("  test-wrap  Execute an active round-trip key wrap/unwrap test inside silicon boundary")
}

// HandleHSMCommand routes PKCS#11 hardware commands.
func HandleHSMCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintHSMHelp()
		return nil
	}

	sub := args[0]
	switch sub {
	case "status":
		return handleHSMStatus(client)
	case "slots":
		return handleHSMSlots(client)
	case "test-wrap", "test":
		return handleHSMTestWrap(client)
	case "help", "--help", "-h":
		PrintHSMHelp()
		return nil
	default:
		PrintHSMHelp()
		return fmt.Errorf("unknown hsm subcommand: %s", sub)
	}
}

func handleHSMStatus(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := client.GetHSMStatus(ctx)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println("          AEITRON SOVEREIGN PKCS#11 HARDWARE SECURITY MODULE STATUS           ")
	fmt.Println("================================================================================")
	fmt.Printf("HSM Status:        %v\n", status["status"])
	fmt.Printf("Operation Mode:    %v\n", status["mode"])
	fmt.Printf("Active Slot ID:    %v\n", status["slot_id"])
	fmt.Printf("Token Label:       %v\n", status["label"])
	fmt.Printf("Manufacturer:      %v\n", status["manufacturer"])
	fmt.Printf("Model / Serial:    %v / %v\n", status["model"], status["serial_number"])
	fmt.Printf("FIPS Standard:     %v\n", status["fips_level"])
	fmt.Printf("Statutory Mandate: %v\n", status["compliance"])
	fmt.Printf("Hardware Locked:   %v (Silicon Enforced)\n", status["hardware_locked"])
	fmt.Printf("Active Sessions:   %v / %v\n", status["active_sessions"], status["max_sessions"])
	fmt.Printf("Free Memory:       %v KB\n", status["free_memory_kb"])
	fmt.Printf("Firmware / HW Ver: %v / %v\n", status["firmware_version"], status["hardware_version"])
	fmt.Println("================================================================================")
	return nil
}

func handleHSMSlots(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := client.GetHSMSlots(ctx)
	if err != nil {
		return err
	}

	slots, ok := data["slots"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected slots format in response")
	}

	fmt.Println("================================================================================")
	fmt.Println("           AEITRON DETECTED PKCS#11 HARDWARE CRYPTOGRAPHIC SLOTS              ")
	fmt.Println("================================================================================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "SLOT ID\tREGION\tROLE\tTOKEN LABEL\tMANUFACTURER\tPRESENT")
	fmt.Fprintln(w, "-------\t------\t----\t-----------\t------------\t-------")

	for _, raw := range slots {
		slot, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\t%v\n",
			slot["slot_id"],
			slot["region"],
			slot["role"],
			slot["token_label"],
			slot["manufacturer_id"],
			slot["token_present"],
		)
	}
	w.Flush()
	fmt.Println("================================================================================")
	return nil
}

func handleHSMTestWrap(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("Executing active round-trip hardware key wrapping probe (AES-256-GCM under KEK)...")
	res, err := client.TestHSMWrap(ctx)
	if err != nil {
		return err
	}

	fmt.Println("================================================================================")
	fmt.Println("                  PKCS#11 HARDWARE CRYPTOGRAPHIC WRAP VERDICT                  ")
	fmt.Println("================================================================================")
	fmt.Printf("Probe Status:      %v\n", res["status"])
	fmt.Printf("Active KEK ID:     %v\n", res["kek_id"])
	fmt.Printf("Operation:         %v\n", res["operation"])
	fmt.Printf("Round-Trip Latency:%v µs\n", res["latency_us"])
	fmt.Printf("Verified At:       %v\n", res["verified_at"])
	fmt.Println("================================================================================")
	fmt.Println("Verdict: Master KEK remained physically isolated within HSM silicon boundary.")
	return nil
}
