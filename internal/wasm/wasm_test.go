package wasm

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleWasmCommandList(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	err := HandleWasmCommand(client, []string{"list"})
	if err != nil {
		t.Fatalf("HandleWasmCommand(list) failed: %v", err)
	}
}

func TestHandleWasmCommandDeployAndDelete(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	// Create temporary dummy wasm bytecode file
	tmpDir := t.TempDir()
	wasmFile := filepath.Join(tmpDir, "test_plugin.wasm")
	dummyBytecode := []byte("\x00asm\x01\x00\x00\x00")
	if err := os.WriteFile(wasmFile, dummyBytecode, 0644); err != nil {
		t.Fatalf("Failed to write dummy wasm: %v", err)
	}

	err := HandleWasmCommand(client, []string{
		"deploy",
		wasmFile,
		"--name", "jwt-validator",
		"--domain", "api.example.com",
		"--phase", "RequestHeaders",
		"--memory-limit", "2097152",
		"--timeout", "100",
	})
	if err != nil {
		t.Fatalf("Deploy command failed: %v", err)
	}

	plugins, err := client.ListPlugins(context.Background())
	if err != nil {
		t.Fatalf("ListPlugins failed: %v", err)
	}

	var foundID string
	for _, p := range plugins {
		if p.Name == "jwt-validator" {
			foundID = p.ID
			break
		}
	}
	if foundID == "" {
		t.Fatalf("Deployed plugin not found in client list")
	}

	// Delete
	err = HandleWasmCommand(client, []string{"delete", foundID})
	if err != nil {
		t.Fatalf("Delete command failed: %v", err)
	}
}
