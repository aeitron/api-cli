package vm

import (
	"testing"
)

func TestCLICommands(t *testing.T) {
	client := NewClient()

	// 1. Overview
	if err := HandleVMCommand(client, []string{"overview"}); err != nil {
		t.Fatalf("HandleVMCommand overview failed: %v", err)
	}

	// 2. Nodes
	if err := HandleVMCommand(client, []string{"nodes"}); err != nil {
		t.Fatalf("HandleVMCommand nodes failed: %v", err)
	}

	// 3. List
	if err := HandleVMCommand(client, []string{"list"}); err != nil {
		t.Fatalf("HandleVMCommand list failed: %v", err)
	}

	// 4. Pools
	if err := HandleVMCommand(client, []string{"pools"}); err != nil {
		t.Fatalf("HandleVMCommand pools failed: %v", err)
	}

	// 5. Create
	if err := HandleVMCommand(client, []string{"create", "--name", "cli-test-vm", "--tier", "micro", "--pop", "bd-dac"}); err != nil {
		t.Fatalf("HandleVMCommand create failed: %v", err)
	}

	// 6. Action
	if err := HandleVMCommand(client, []string{"action", "vm-micro-4a81f", "pause"}); err != nil {
		t.Fatalf("HandleVMCommand action failed: %v", err)
	}

	// 7. Delete
	if err := HandleVMCommand(client, []string{"delete", "vm-micro-4a81f"}); err != nil {
		t.Fatalf("HandleVMCommand delete failed: %v", err)
	}
}
