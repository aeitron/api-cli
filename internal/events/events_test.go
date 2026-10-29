package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	events "github.com/aeitron/api-cli/pkg/events"
)

func TestHandleEventsCommandSchemas(t *testing.T) {
	err := HandleEventsCommand([]string{"schemas"})
	if err != nil {
		t.Fatalf("unexpected error running 'schemas': %v", err)
	}
}

func TestHandleEventsCommandValidateValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	validEventFile := filepath.Join(tmpDir, "valid_event.json")

	env, err := events.NewEnvelope(
		"io.aeitron.security.threat_detected.v1",
		"urn:aeitron:pop:bd-dac:service:security",
		events.PoPDhaka,
		events.JurisdictionBangladesh,
		true,
		map[string]any{
			"threat_id": "a1b2c3d4-e5f6-4000-8000-123456789abc",
		},
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create envelope: %v", err)
	}

	bytes, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	if err := os.WriteFile(validEventFile, bytes, 0644); err != nil {
		t.Fatalf("failed to write tmp event file: %v", err)
	}

	if err := HandleEventsCommand([]string{"validate", validEventFile}); err != nil {
		t.Fatalf("expected validation to pass, got: %v", err)
	}
}

func TestHandleEventsCommandValidateInvalidFile(t *testing.T) {
	tmpDir := t.TempDir()
	invalidEventFile := filepath.Join(tmpDir, "invalid_event.json")

	badData := `{"specversion": "1.0", "id": "not-a-uuid"}`
	if err := os.WriteFile(invalidEventFile, []byte(badData), 0644); err != nil {
		t.Fatalf("failed to write invalid file: %v", err)
	}

	err := HandleEventsCommand([]string{"validate", invalidEventFile})
	if err == nil {
		t.Fatalf("expected validation error for invalid file, got nil")
	}
}

func TestHandleEventsCommandHelp(t *testing.T) {
	if err := HandleEventsCommand([]string{"--help"}); err != nil {
		t.Fatalf("unexpected error running '--help': %v", err)
	}
}
