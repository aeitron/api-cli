package db

import "time"

// Dialect represents database engines.
type Dialect string

const (
	DialectPostgres   Dialect = "POSTGRES"
	DialectClickHouse Dialect = "CLICKHOUSE"
)

// SeedScale specifies seeder dataset volume.
type SeedScale string

const (
	SeedScaleMinimal   SeedScale = "MINIMAL"
	SeedScaleStandard  SeedScale = "STANDARD"
	SeedScaleBenchmark SeedScale = "BENCHMARK"
	SeedScaleFull      SeedScale = "FULL"
)

// AppliedMigration details an individual schema migration step.
type AppliedMigration struct {
	Version    int       `json:"version"`
	Domain     string    `json:"domain"`
	Dialect    Dialect   `json:"dialect"`
	Name       string    `json:"name"`
	Checksum   string    `json:"checksum"`
	Dirty      bool      `json:"dirty"`
	AppliedAt  time.Time `json:"applied_at"`
	DurationMs int64     `json:"duration_ms"`
}

// MigrationReport summarizes migration execution.
type MigrationReport struct {
	Dialect        Dialect            `json:"dialect"`
	AppliedCount   int                `json:"applied_count"`
	RollbackCount  int                `json:"rollback_count"`
	TargetVersion  int                `json:"target_version"`
	ChecksumsValid bool               `json:"checksums_valid"`
	DurationMs     int64              `json:"duration_ms"`
	AppliedSteps   []AppliedMigration `json:"applied_steps"`
}

// MigrationStatusReport details current database migration state.
type MigrationStatusReport struct {
	DatabaseDialect Dialect            `json:"database_dialect"`
	CurrentVersion  int                `json:"current_version"`
	TotalMigrations int                `json:"total_migrations"`
	AppliedCount    int                `json:"applied_count"`
	PendingCount    int                `json:"pending_count"`
	Dirty           bool               `json:"dirty"`
	ChecksumsValid  bool               `json:"checksums_valid"`
	Steps           []AppliedMigration `json:"steps"`
}

// SeedEntityCount details count of generated records per table.
type SeedEntityCount struct {
	Domain         string `json:"domain"`
	TableName      string `json:"table_name"`
	Count          int    `json:"count"`
	SovereignCount int    `json:"sovereign_count"`
}

// SeedReport summarizes automated database seeding.
type SeedReport struct {
	Scale             SeedScale         `json:"scale"`
	TotalEntities     int               `json:"total_entities"`
	SovereignEntities int               `json:"sovereign_entities"`
	Domains           []SeedEntityCount `json:"domains"`
	DurationMs        int64             `json:"duration_ms"`
	Status            string            `json:"status"`
	Attestation       string            `json:"attestation"`
}
