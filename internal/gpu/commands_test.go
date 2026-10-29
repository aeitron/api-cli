package gpu

import (
	"testing"
)

func TestHandleGPUCommand_Overview(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	if err := HandleGPUCommand(client, []string{"overview"}); err != nil {
		t.Fatalf("unexpected error running 'overview': %v", err)
	}
}

func TestHandleGPUCommand_Nodes(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	if err := HandleGPUCommand(client, []string{"nodes"}); err != nil {
		t.Fatalf("unexpected error running 'nodes': %v", err)
	}
}

func TestHandleGPUCommand_Slices(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	if err := HandleGPUCommand(client, []string{"slices"}); err != nil {
		t.Fatalf("unexpected error running 'slices': %v", err)
	}
}

func TestHandleGPUCommand_AllocateAndRelease(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	err := HandleGPUCommand(client, []string{
		"allocate",
		"--tenant", "tenant-test",
		"--workload", "wl-test",
		"--pop", "bd-dac",
		"--profile", "1g.10gb",
		"--model", "qwen-2.5-coder-7b",
	})
	if err != nil {
		t.Fatalf("unexpected error allocating slice: %v", err)
	}

	err = HandleGPUCommand(client, []string{"release", "slice-m-90a1"})
	if err != nil {
		t.Fatalf("unexpected error releasing slice: %v", err)
	}
}

func TestHandleGPUCommand_VLLM(t *testing.T) {
	client := NewClient()
	client.MockMode = true

	if err := HandleGPUCommand(client, []string{"vllm"}); err != nil {
		t.Fatalf("unexpected error running 'vllm': %v", err)
	}
}

func TestHandleGPUCommand_Help(t *testing.T) {
	client := NewClient()
	if err := HandleGPUCommand(client, []string{"--help"}); err != nil {
		t.Fatalf("unexpected error running '--help': %v", err)
	}
}
