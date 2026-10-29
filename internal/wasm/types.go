package wasm

import "time"

// PluginPhase denotes when the WASM hook runs during request processing.
type PluginPhase string

const (
	PhaseRequestHeaders  PluginPhase = "RequestHeaders"
	PhaseRequestBody     PluginPhase = "RequestBody"
	PhaseResponseHeaders PluginPhase = "ResponseHeaders"
)

// WasmPlugin represents a registered edge-native WebAssembly extension.
type WasmPlugin struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	DomainFilter     string      `json:"domain_filter"`
	Phase            PluginPhase `json:"phase"`
	MemoryLimitBytes int         `json:"memory_limit_bytes"`
	TimeoutMicros    uint64      `json:"timeout_micros"`
	BytecodeSha256   string      `json:"bytecode_sha256"`
	Enabled          bool        `json:"enabled"`
	CreatedAt        time.Time   `json:"created_at"`
}

// DeployPluginRequest contains parameters for deploying a new WASM module.
type DeployPluginRequest struct {
	Name             string      `json:"name"`
	DomainFilter     string      `json:"domain_filter"`
	Phase            PluginPhase `json:"phase"`
	MemoryLimitBytes int         `json:"memory_limit_bytes"`
	TimeoutMicros    uint64      `json:"timeout_micros"`
	WasmBytecode     []byte      `json:"-"`
}
