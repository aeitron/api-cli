package db

import (
	"testing"
)

func TestHandleDBCommandHelp(t *testing.T) {
	client := NewClient()

	if err := HandleDBCommand(client, []string{}); err != nil {
		t.Fatalf("HandleDBCommand() without args returned error: %v", err)
	}

	if err := HandleDBCommand(client, []string{"help"}); err != nil {
		t.Fatalf("HandleDBCommand(help) returned error: %v", err)
	}
}

func TestHandleDBCommandStatus(t *testing.T) {
	client := NewClient()

	if err := HandleDBCommand(client, []string{"status"}); err != nil {
		t.Fatalf("HandleDBCommand(status) returned error: %v", err)
	}
}

func TestHandleDBCommandMigrate(t *testing.T) {
	client := NewClient()

	if err := HandleDBCommand(client, []string{"migrate", "--dialect", "postgres", "--version", "1"}); err != nil {
		t.Fatalf("HandleDBCommand(migrate postgres) returned error: %v", err)
	}

	if err := HandleDBCommand(client, []string{"up", "--dialect", "clickhouse", "--version", "1"}); err != nil {
		t.Fatalf("HandleDBCommand(up clickhouse) returned error: %v", err)
	}
}

func TestHandleDBCommandRollback(t *testing.T) {
	client := NewClient()

	if err := HandleDBCommand(client, []string{"rollback", "--dialect", "postgres", "--steps", "1"}); err != nil {
		t.Fatalf("HandleDBCommand(rollback postgres) returned error: %v", err)
	}

	if err := HandleDBCommand(client, []string{"down", "--dialect", "clickhouse", "--steps", "1"}); err != nil {
		t.Fatalf("HandleDBCommand(down clickhouse) returned error: %v", err)
	}
}

func TestHandleDBCommandSeed(t *testing.T) {
	client := NewClient()

	if err := HandleDBCommand(client, []string{"seed", "--scale", "standard"}); err != nil {
		t.Fatalf("HandleDBCommand(seed standard) returned error: %v", err)
	}

	if err := HandleDBCommand(client, []string{"seed", "--scale", "benchmark"}); err != nil {
		t.Fatalf("HandleDBCommand(seed benchmark) returned error: %v", err)
	}
}

func TestHandleDBCommandVerify(t *testing.T) {
	client := NewClient()

	if err := HandleDBCommand(client, []string{"verify"}); err != nil {
		t.Fatalf("HandleDBCommand(verify) returned error: %v", err)
	}
}

func TestHandleDBCommandUnknown(t *testing.T) {
	client := NewClient()

	if err := HandleDBCommand(client, []string{"invalid_command"}); err == nil {
		t.Fatalf("HandleDBCommand(invalid_command) should have failed but succeeded")
	}
}
