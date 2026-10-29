package gpu

import "time"

type GPUDevice struct {
	ID                   string    `json:"id"`
	NodeID               string    `json:"node_id"`
	Region               string    `json:"region"`
	Model                string    `json:"model"`
	TotalMemoryMB        int64     `json:"total_memory_mb"`
	AllocatedMemoryMB    int64     `json:"allocated_memory_mb"`
	AvailableMemoryMB    int64     `json:"available_memory_mb"`
	MIGEnabled           bool      `json:"mig_enabled"`
	MIGProfilesSupported []string  `json:"mig_profiles_supported,omitempty"`
	TemperatureCelsius   float64   `json:"temperature_celsius"`
	PowerWatts           float64   `json:"power_watts"`
	ClockMHz             int       `json:"clock_mhz"`
	ActiveSlices         []string  `json:"active_slices"`
	SovereignLocked      bool      `json:"sovereign_locked"`
	LastTelemetry        time.Time `json:"last_telemetry"`
}

type GPUSlice struct {
	SliceID           string    `json:"slice_id"`
	ParentDeviceID    string    `json:"parent_device_id"`
	NodeID            string    `json:"node_id"`
	Region            string    `json:"region"`
	AllocationType    string    `json:"allocation_type"`
	MIGProfile        string    `json:"mig_profile,omitempty"`
	TimeSliceFraction float64   `json:"time_slice_fraction,omitempty"`
	MemoryMB          int64     `json:"memory_mb"`
	TenantID          string    `json:"tenant_id"`
	WorkloadID        string    `json:"workload_id"`
	ModelName         string    `json:"model_name"`
	Sovereignty       string    `json:"sovereignty"`
	PQCLeaseToken     string    `json:"pqc_lease_token"`
	Status            string    `json:"status"`
	MemoryZeroized    bool      `json:"memory_zeroized"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type VLLMInstance struct {
	InstanceID           string    `json:"instance_id"`
	SliceID              string    `json:"slice_id"`
	NodeID               string    `json:"node_id"`
	Region               string    `json:"region"`
	Model                string    `json:"model"`
	Port                 int       `json:"port"`
	TensorParallelSize   int       `json:"tensor_parallel_size"`
	MaxModelLen          int       `json:"max_model_len"`
	GPUMemoryUtilization float64   `json:"gpu_memory_utilization"`
	KVMemoryMB           int64     `json:"kv_memory_mb"`
	State                string    `json:"state"`
	ThroughputTokPerSec  float64   `json:"throughput_tok_per_sec"`
	LatencyP95Ms         float64   `json:"latency_p95_ms"`
	KVCacheUsagePercent  float64   `json:"kv_cache_usage_percent"`
	LastHeartbeat        time.Time `json:"last_heartbeat"`
}

type GPUClusterOverview struct {
	TotalDevices         int     `json:"total_devices"`
	TotalMemoryMB        int64   `json:"total_memory_mb"`
	AllocatedMemoryMB    int64   `json:"allocated_memory_mb"`
	AvailableMemoryMB    int64   `json:"available_memory_mb"`
	TotalSlices          int     `json:"total_slices"`
	ActiveVLLMInstances  int     `json:"active_vllm_instances"`
	AverageTemperatureC  float64 `json:"average_temperature_c"`
	TotalPowerWatts      float64 `json:"total_power_watts"`
	NDMASovereignDevices int     `json:"ndma_sovereign_devices"`
	PQCAttestation       string  `json:"pqc_attestation"`
}

type AllocationRequest struct {
	TenantID          string  `json:"tenant_id"`
	WorkloadID        string  `json:"workload_id"`
	Region            string  `json:"region"`
	PreferredModel    string  `json:"preferred_model,omitempty"`
	AllocationType    string  `json:"allocation_type"`
	MIGProfile        string  `json:"mig_profile,omitempty"`
	TimeSliceFraction float64 `json:"time_slice_fraction,omitempty"`
	ModelName         string  `json:"model_name"`
	Sovereignty       string  `json:"sovereignty"`
}
