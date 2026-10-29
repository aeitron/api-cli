package workload

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ParseManifestFile reads and parses a workload manifest from disk (JSON or YAML).
func ParseManifestFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}
	return ParseManifestBytes(data)
}

// ParseManifestBytes parses raw manifest data supporting JSON and lightweight YAML.
func ParseManifestBytes(data []byte) (*Manifest, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, errors.New("empty manifest content")
	}

	// 1. Try standard JSON first
	if strings.HasPrefix(trimmed, "{") {
		var m Manifest
		if err := json.Unmarshal([]byte(trimmed), &m); err != nil {
			return nil, fmt.Errorf("invalid manifest JSON: %w", err)
		}
		if err := ValidateManifest(&m); err != nil {
			return nil, err
		}
		return &m, nil
	}

	// 2. Parse YAML line-by-line into structured manifest
	m, err := parseYAMLToManifest(trimmed)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML manifest: %w", err)
	}

	if err := ValidateManifest(m); err != nil {
		return nil, err
	}
	return m, nil
}

// parseYAMLToManifest implements robust zero-dependency YAML parsing for Workload manifests.
func parseYAMLToManifest(content string) (*Manifest, error) {
	m := &Manifest{
		Metadata: MetadataSpec{
			Labels: make(map[string]string),
		},
		Spec: WorkloadSpec{
			Type:         TypeGPUInference,
			Jurisdiction: JurisdictionAny,
			Objective:    ObjectiveBalanced,
		},
	}

	currentSection := ""
	lines := strings.Split(content, "\n")

	for lineNum, rawLine := range lines {
		line := strings.TrimRight(rawLine, "\r\t ")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		trimmedLine := strings.TrimSpace(line)
		parts := strings.SplitN(trimmedLine, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		// Detect section headers
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			switch key {
			case "apiVersion":
				m.APIVersion = val
				currentSection = ""
				continue
			case "kind":
				m.Kind = val
				currentSection = ""
				continue
			case "metadata":
				currentSection = "metadata"
				continue
			case "spec":
				currentSection = "spec"
				continue
			}
		}

		// Parse fields within sections
		switch currentSection {
		case "metadata":
			switch key {
			case "name":
				m.Metadata.Name = val
			case "tenant_id", "tenantId":
				m.Metadata.TenantID = val
			case "project_id", "projectId":
				m.Metadata.ProjectID = val
			}
		case "spec":
			switch key {
			case "workload_id", "workloadId":
				m.Spec.WorkloadID = val
			case "type":
				m.Spec.Type = WorkloadType(strings.ToUpper(val))
			case "jurisdiction":
				m.Spec.Jurisdiction = DataJurisdiction(strings.ToUpper(val))
			case "strict_sovereignty", "strictSovereignty":
				m.Spec.StrictSovereignty = parseBool(val)
			case "max_latency_p95_ms", "maxLatencyP95Ms":
				if n, err := strconv.ParseUint(val, 10, 32); err == nil {
					m.Spec.MaxLatencyP95Ms = uint32(n)
				}
			case "max_cost_budget_usd", "maxCostBudgetUSD":
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					m.Spec.MaxCostBudgetUSD = f
				}
			case "cpu_millicores", "cpuMillicores":
				if n, err := strconv.ParseUint(val, 10, 32); err == nil {
					m.Spec.CPUMillicores = uint32(n)
				}
			case "memory_bytes", "memoryBytes":
				if n, err := strconv.ParseUint(val, 10, 64); err == nil {
					m.Spec.MemoryBytes = n
				}
			case "gpu_count", "gpuCount":
				if n, err := strconv.ParseUint(val, 10, 32); err == nil {
					m.Spec.GPUCount = uint32(n)
				}
			case "required_gpu_model", "requiredGpuModel":
				m.Spec.RequiredGPUModel = val
			case "objective":
				m.Spec.Objective = OptimizationObjective(strings.ToUpper(val))
			case "client_origin_region", "clientOriginRegion":
				m.Spec.ClientOriginRegion = strings.ToLower(val)
			case "idempotency_key", "idempotencyKey":
				m.Spec.IdempotencyKey = val
			default:
				_ = lineNum
			}
		}
	}

	// Propagate metadata to spec if missing
	if m.Spec.WorkloadID == "" && m.Metadata.Name != "" {
		m.Spec.WorkloadID = m.Metadata.Name
	}

	return m, nil
}

func parseBool(val string) bool {
	v := strings.ToLower(strings.TrimSpace(val))
	return v == "true" || v == "yes" || v == "1" || v == "on"
}

// ValidateManifest verifies syntactic invariants, field bounds, and regulatory rules.
func ValidateManifest(m *Manifest) error {
	if m == nil {
		return errors.New("manifest is nil")
	}

	if m.Kind != "" && !strings.EqualFold(m.Kind, "Workload") {
		return fmt.Errorf("unsupported kind: %q (expected 'Workload')", m.Kind)
	}

	if m.Spec.WorkloadID == "" && m.Metadata.Name == "" {
		return errors.New("workload requires either spec.workload_id or metadata.name")
	}
	if m.Spec.WorkloadID == "" {
		m.Spec.WorkloadID = m.Metadata.Name
	}
	if m.Metadata.Name == "" {
		m.Metadata.Name = m.Spec.WorkloadID
	}

	if m.Metadata.TenantID == "" {
		return errors.New("metadata.tenant_id is required for multi-tenant isolation (Rule 54)")
	}

	if m.Spec.GPUCount > 64 {
		return fmt.Errorf("spec.gpu_count (%d) exceeds maximum limit of 64", m.Spec.GPUCount)
	}

	if m.Spec.StrictSovereignty && m.Spec.Jurisdiction == "" {
		m.Spec.Jurisdiction = JurisdictionBangladesh
	}

	return nil
}

// GenerateTemplate outputs a canonical YAML manifest template for rapid developer bootstrapping.
func GenerateTemplate(targetType string) string {
	switch strings.ToLower(targetType) {
	case "sovereign", "bd", "banking":
		return `# Aeitron Sovereign Banking & Critical Infrastructure Workload Manifest
# Enforces Bangladesh National Data Management Act 2026 (NDMA 2026)
apiVersion: aeitron.io/v1
kind: Workload
metadata:
  name: nid-verification-gateway
  tenant_id: tenant-financial-services
  project_id: proj-sovereign-core
spec:
  workload_id: nid-verification-gateway
  type: API_GATEWAY
  jurisdiction: BANGLADESH
  strict_sovereignty: true
  max_latency_p95_ms: 25
  max_cost_budget_usd: 5.00
  cpu_millicores: 4000
  memory_bytes: 8589934592 # 8 GiB
  gpu_count: 0
  objective: LOW_LATENCY
  client_origin_region: dhaka
`
	case "container", "microservice":
		return `# Aeitron Universal Container Workload Manifest
apiVersion: aeitron.io/v1
kind: Workload
metadata:
  name: checkout-microservice
  tenant_id: tenant-ecommerce
  project_id: proj-checkout
spec:
  workload_id: checkout-microservice
  type: CONTAINER_POD
  jurisdiction: ANY
  strict_sovereignty: false
  max_latency_p95_ms: 50
  max_cost_budget_usd: 2.50
  cpu_millicores: 2000
  memory_bytes: 4294967296 # 4 GiB
  gpu_count: 0
  objective: BALANCED
  client_origin_region: singapore
`
	case "gpu", "inference", "ai":
		fallthrough
	default:
		return `# Aeitron Universal Workload Manifest: Distributed GPU Inference
apiVersion: aeitron.io/v1
kind: Workload
metadata:
  name: deepseek-r1-inference
  tenant_id: tenant-ai-labs
  project_id: proj-reasoning-fleet
spec:
  workload_id: deepseek-r1-inference
  type: GPU_INFERENCE
  jurisdiction: ANY
  strict_sovereignty: false
  max_latency_p95_ms: 60
  max_cost_budget_usd: 8.00
  cpu_millicores: 16000
  memory_bytes: 68719476736 # 64 GiB
  gpu_count: 4
  required_gpu_model: H100
  objective: LOW_LATENCY
  client_origin_region: dhaka
`
	}
}
