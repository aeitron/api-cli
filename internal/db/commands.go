package db

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintDBHelp displays available database migration and seeder commands.
func PrintDBHelp() {
	fmt.Println("Aeitron Distributed Database Migration & Schema Seeder Engine")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  aeitron db <subcommand> [flags]")
	fmt.Println("  aeitron migrate <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  status                         Inspect migration state, versions, and checksum validity")
	fmt.Println("  migrate, up [flags]            Apply forward zero-downtime migrations")
	fmt.Println("  rollback, down [flags]         Roll back applied migrations in reverse dependency order")
	fmt.Println("  seed [flags]                   Populate synthetic seed records with sovereign tagging")
	fmt.Println("  verify                         Verify migration SHA-256 checksum integrity against disk")
	fmt.Println()
	fmt.Println("Flags for migrate / up:")
	fmt.Println("  --dialect <postgres|clickhouse> Target database dialect (default: postgres)")
	fmt.Println("  --version <n>                   Target migration version (0 = all pending)")
	fmt.Println()
	fmt.Println("Flags for rollback / down:")
	fmt.Println("  --dialect <postgres|clickhouse> Target database dialect (default: postgres)")
	fmt.Println("  --steps <n>                     Number of migration steps to rollback (default: 1)")
	fmt.Println()
	fmt.Println("Flags for seed:")
	fmt.Println("  --scale <minimal|standard|benchmark|full> Seeding volume scale (default: standard)")
}

// HandleDBCommand routes database and migration CLI subcommands.
func HandleDBCommand(client *Client, args []string) error {
	if len(args) == 0 {
		PrintDBHelp()
		return nil
	}

	sub := args[0]
	switch sub {
	case "status":
		return handleStatus(client)
	case "migrate", "up":
		return handleMigrate(client, args[1:])
	case "rollback", "down":
		return handleRollback(client, args[1:])
	case "seed":
		return handleSeed(client, args[1:])
	case "verify":
		return handleVerify(client)
	case "help", "-h", "--help":
		PrintDBHelp()
		return nil
	default:
		return fmt.Errorf("unknown db subcommand: %s (run 'aeitron db help')", sub)
	}
}

func handleStatus(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch migration status: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Println(" Aeitron Distributed Database Migration Status (CockroachDB & ClickHouse)")
	fmt.Println("================================================================================")

	if resp.Postgres != nil {
		p := resp.Postgres
		chkStr := "VALID (SHA-256 Verified)"
		if !p.ChecksumsValid {
			chkStr = "TAMPERED / MISMATCH"
		}
		fmt.Printf("PostgreSQL / CockroachDB (Transactional):\n")
		fmt.Printf("  Current Version:  %d\n", p.CurrentVersion)
		fmt.Printf("  Applied / Total:  %d / %d migrations\n", p.AppliedCount, p.TotalMigrations)
		fmt.Printf("  Pending Count:    %d\n", p.PendingCount)
		fmt.Printf("  Dirty State:      %t\n", p.Dirty)
		fmt.Printf("  Checksums:        %s\n", chkStr)
		if len(p.Steps) > 0 {
			fmt.Println("\nApplied Steps (PostgreSQL):")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "  VERSION\tDOMAIN\tNAME\tDIRTY\tAPPLIED AT")
			for _, s := range p.Steps {
				fmt.Fprintf(w, "  %d\t%s\t%s\t%t\t%s\n",
					s.Version, s.Domain, s.Name, s.Dirty, s.AppliedAt.Format("2006-01-02 15:04:05"))
			}
			w.Flush()
		}
	}

	fmt.Println()

	if resp.ClickHouse != nil {
		c := resp.ClickHouse
		chkStr := "VALID (SHA-256 Verified)"
		if !c.ChecksumsValid {
			chkStr = "TAMPERED / MISMATCH"
		}
		fmt.Printf("ClickHouse (Analytical Line-Rate):\n")
		fmt.Printf("  Current Version:  %d\n", c.CurrentVersion)
		fmt.Printf("  Applied / Total:  %d / %d migrations\n", c.AppliedCount, c.TotalMigrations)
		fmt.Printf("  Pending Count:    %d\n", c.PendingCount)
		fmt.Printf("  Dirty State:      %t\n", c.Dirty)
		fmt.Printf("  Checksums:        %s\n", chkStr)
		if len(c.Steps) > 0 {
			fmt.Println("\nApplied Steps (ClickHouse):")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "  VERSION\tDOMAIN\tNAME\tDIRTY\tAPPLIED AT")
			for _, s := range c.Steps {
				fmt.Fprintf(w, "  %d\t%s\t%s\t%t\t%s\n",
					s.Version, s.Domain, s.Name, s.Dirty, s.AppliedAt.Format("2006-01-02 15:04:05"))
			}
			w.Flush()
		}
	}

	fmt.Println("================================================================================")
	return nil
}

func handleMigrate(client *Client, args []string) error {
	fs := flag.NewFlagSet("db migrate", flag.ContinueOnError)
	dialectStr := fs.String("dialect", "postgres", "Target database dialect (postgres or clickhouse)")
	targetVersion := fs.Int("version", 0, "Target migration version (0 = all pending)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	dialect := DialectPostgres
	if *dialectStr == "clickhouse" {
		dialect = DialectClickHouse
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Applying forward zero-downtime migrations (Dialect: %s, Target Version: %d)...\n", dialect, *targetVersion)
	report, err := client.MigrateUp(ctx, dialect, *targetVersion)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("Migration completed successfully:")
	fmt.Printf("  Dialect:          %s\n", report.Dialect)
	fmt.Printf("  Applied Count:    %d\n", report.AppliedCount)
	fmt.Printf("  Target Version:   %d\n", report.TargetVersion)
	fmt.Printf("  Checksums Valid:  %t\n", report.ChecksumsValid)
	fmt.Printf("  Duration:         %dms\n", report.DurationMs)
	return nil
}

func handleRollback(client *Client, args []string) error {
	fs := flag.NewFlagSet("db rollback", flag.ContinueOnError)
	dialectStr := fs.String("dialect", "postgres", "Target database dialect (postgres or clickhouse)")
	steps := fs.Int("steps", 1, "Number of migration steps to rollback")

	if err := fs.Parse(args); err != nil {
		return err
	}

	dialect := DialectPostgres
	if *dialectStr == "clickhouse" {
		dialect = DialectClickHouse
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Rolling back migrations (Dialect: %s, Steps: %d)...\n", dialect, *steps)
	report, err := client.MigrateDown(ctx, dialect, *steps)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Println("Rollback completed successfully:")
	fmt.Printf("  Dialect:          %s\n", report.Dialect)
	fmt.Printf("  Rollback Count:   %d\n", report.RollbackCount)
	fmt.Printf("  Checksums Valid:  %t\n", report.ChecksumsValid)
	fmt.Printf("  Duration:         %dms\n", report.DurationMs)
	return nil
}

func handleSeed(client *Client, args []string) error {
	fs := flag.NewFlagSet("db seed", flag.ContinueOnError)
	scaleStr := fs.String("scale", "standard", "Seeding scale: minimal, standard, benchmark, full")

	if err := fs.Parse(args); err != nil {
		return err
	}

	scale := SeedScale(strings.ToUpper(*scaleStr))
	switch scale {
	case SeedScaleMinimal, SeedScaleStandard, SeedScaleBenchmark, SeedScaleFull:
	default:
		scale = SeedScaleStandard
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("Seeding synthetic data with sovereign tagging (Scale: %s)...\n", scale)
	report, err := client.Seed(ctx, scale)
	if err != nil {
		return fmt.Errorf("seeding failed: %w", err)
	}

	fmt.Println("================================================================================")
	fmt.Printf(" Aeitron Sovereign Database Seeder Engine (Scale: %s)\n", report.Scale)
	fmt.Println("================================================================================")
	fmt.Printf("Total Entities Seeded:     %d\n", report.TotalEntities)
	fmt.Printf("Sovereign Tagged Entities: %d (Bangladesh NDMA 2026 Confined)\n", report.SovereignEntities)
	fmt.Printf("Status:                    %s\n", report.Status)
	fmt.Printf("Execution Time:            %dms\n", report.DurationMs)
	fmt.Printf("Attestation:               %s\n", report.Attestation)
	fmt.Println()
	fmt.Println("Domain Breakdown:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "  DOMAIN\tTABLE\tCOUNT\tSOVEREIGN")
	for _, d := range report.Domains {
		fmt.Fprintf(w, "  %s\t%s\t%d\t%d\n", d.Domain, d.TableName, d.Count, d.SovereignCount)
	}
	w.Flush()
	fmt.Println("================================================================================")
	return nil
}

func handleVerify(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetStatus(ctx)
	if err != nil {
		return fmt.Errorf("verification query failed: %w", err)
	}

	pgOk := resp.Postgres != nil && resp.Postgres.ChecksumsValid
	chOk := resp.ClickHouse != nil && resp.ClickHouse.ChecksumsValid

	if pgOk && chOk {
		fmt.Println("Verification SUCCESS: All SHA-256 migration checksums match authoritative disk state.")
		return nil
	}

	return fmt.Errorf("verification FAILED: Checksum tamper detected (Postgres Valid: %t, ClickHouse Valid: %t)", pgOk, chOk)
}
