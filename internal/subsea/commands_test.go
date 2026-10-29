package subsea

import (
	"strings"
	"testing"
)

func TestHandleSubseaCommand(t *testing.T) {
	client := NewClient()

	// 1. Help
	err := HandleSubseaCommand(client, []string{"help"})
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	// 2. Status HUD
	err = HandleSubseaCommand(client, []string{"status"})
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}

	// 3. Probe SMW6
	err = HandleSubseaCommand(client, []string{"probe", "SMW6"})
	if err != nil {
		t.Fatalf("probe SMW6 failed: %v", err)
	}

	// 4. Probe ITC
	err = HandleSubseaCommand(client, []string{"probe", "ITC"})
	if err != nil {
		t.Fatalf("probe ITC failed: %v", err)
	}

	// 5. Probe Invalid Cable
	err = HandleSubseaCommand(client, []string{"probe", "NON-EXISTENT-CABLE"})
	if err == nil {
		t.Fatal("expected error on probe for non-existent cable")
	}

	// 6. Steer
	err = HandleSubseaCommand(client, []string{"steer", "SMW6", "--target", "sg-sin"})
	if err != nil {
		t.Fatalf("steer failed: %v", err)
	}

	// 7. Unknown subcommand
	err = HandleSubseaCommand(client, []string{"invalid-subcommand"})
	if err == nil || !strings.Contains(err.Error(), "unknown subsea subcommand") {
		t.Fatalf("expected unknown subcommand error, got: %v", err)
	}
}
