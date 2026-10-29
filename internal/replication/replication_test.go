package replication

import (
	"testing"
)

func TestHandleReplicationCommandStatus(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	err := HandleReplicationCommand(client, []string{"status"})
	if err != nil {
		t.Fatalf("HandleReplicationCommand(status) failed: %v", err)
	}
}

func TestHandleReplicationCommandSync(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	err := HandleReplicationCommand(client, []string{"sync", "--peer", "sg-sin"})
	if err != nil {
		t.Fatalf("HandleReplicationCommand(sync) failed: %v", err)
	}
}

func TestHandleReplicationCommandTopology(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	err := HandleReplicationCommand(client, []string{"topology"})
	if err != nil {
		t.Fatalf("HandleReplicationCommand(topology) failed: %v", err)
	}
}

func TestHandleDatabaseCommandStatus(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	err := HandleReplicationCommand(client, []string{"database", "status"})
	if err != nil {
		t.Fatalf("HandleReplicationCommand(database, status) failed: %v", err)
	}
}

func TestHandleDatabaseCommandSync(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	err := HandleReplicationCommand(client, []string{"database", "sync", "--peer", "sg-sin"})
	if err != nil {
		t.Fatalf("HandleReplicationCommand(database, sync) failed: %v", err)
	}
}
