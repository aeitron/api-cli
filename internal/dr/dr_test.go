package dr

import (
	"testing"
)

func TestDRCLICommands_Execution(t *testing.T) {
	client := NewClient()

	commands := [][]string{
		{"help"},
		{"status", "--tenant", "tenant-dhaka-bank-01"},
		{"policy", "--tenant", "tenant-dhaka-bank-01"},
		{"rehearse", "tenant-dhaka-bank-01"},
		{"failover", "tenant-global-ai-02", "singapore"},
		{"failback", "tenant-global-ai-02"},
		{"journal", "--tenant", "tenant-global-ai-02"},
		{"profiles"},
	}

	for _, cmd := range commands {
		t.Run(cmd[0], func(t *testing.T) {
			if err := HandleDRCommand(client, cmd); err != nil {
				t.Fatalf("command failed: %v", err)
			}
		})
	}
}
