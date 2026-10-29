package network

import (
	"context"
	"testing"
)

func TestMeshCommands_MockWorkflow(t *testing.T) {
	client := NewClient()

	// 1. Test GetMeshStatus
	ctx := context.Background()
	summary, err := client.GetMeshStatus(ctx)
	if err != nil {
		t.Fatalf("failed to get mesh status: %v", err)
	}
	if summary.LocalPoPID != "bd-dac" || summary.ActivePeersCount != 4 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	// 2. Test GetMeshPeers
	peers, err := client.GetMeshPeers(ctx)
	if err != nil {
		t.Fatalf("failed to get mesh peers: %v", err)
	}
	if len(peers) != 4 {
		t.Fatalf("expected 4 peers, got %d", len(peers))
	}

	// 3. Test RekeyMeshPeer
	rekey, err := client.RekeyMeshPeer(ctx, "sg-sin", "operator-manual-test")
	if err != nil {
		t.Fatalf("rekey failed: %v", err)
	}
	if rekey.Status != "SUCCESS" || rekey.PeerID != "sg-sin" {
		t.Fatalf("unexpected rekey response: %+v", rekey)
	}

	// 4. Test HandleMeshCommand handlers
	if err := HandleMeshCommand(client, []string{"status"}); err != nil {
		t.Fatalf("status command failed: %v", err)
	}
	if err := HandleMeshCommand(client, []string{"peers"}); err != nil {
		t.Fatalf("peers command failed: %v", err)
	}
	if err := HandleMeshCommand(client, []string{"rekey", "sg-sin", "--reason", "audit-check"}); err != nil {
		t.Fatalf("rekey command failed: %v", err)
	}
	if err := HandleMeshCommand(client, []string{"topology"}); err != nil {
		t.Fatalf("topology command failed: %v", err)
	}
	if err := HandleMeshCommand(client, []string{"help"}); err != nil {
		t.Fatalf("help command failed: %v", err)
	}
}

func TestMeshCommands_UnknownSubcommand(t *testing.T) {
	client := NewClient()
	err := HandleMeshCommand(client, []string{"invalid-subcommand"})
	if err == nil {
		t.Fatalf("expected error on invalid subcommand, got nil")
	}
}
