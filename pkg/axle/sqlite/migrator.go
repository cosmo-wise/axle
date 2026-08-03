package sqlite

import (
	"context"
	"fmt"

	internalsqlite "github.com/cosmo-wise/axle/internal/sqlite"
)

func validateMigrationSets(sets []MigrationSet) error {
	seen := map[string]bool{}
	for _, set := range sets {
		if seen[set.Component] {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("duplicate component %q", set.Component)}
		}
		seen[set.Component] = true
		if set.MinCompatible < 0 || set.MinCompatible > set.TargetVersion {
			return &MigrationError{Kind: StatusInvalidSet, Message: fmt.Sprintf("component %s has invalid min compatible version %d for target %d", set.Component, set.MinCompatible, set.TargetVersion)}
		}
		if err := internalsqlite.ValidateSet(set.Component, toInternalDefs(set), set.TargetVersion); err != nil {
			return err
		}
	}
	return nil
}

func componentMigrationStatus(set MigrationSet, defs []internalsqlite.MigrationDef, records []internalsqlite.MigrationRecord) (int, StatusKind) {
	current := internalsqlite.MaxMigrationVersion(records)
	recordMap := map[int]string{}
	for _, record := range records {
		recordMap[record.Version] = record.Checksum
	}
	for _, migration := range defs {
		if migration.Version > current {
			break
		}
		checksum, ok := recordMap[migration.Version]
		if !ok || checksum != migration.Checksum {
			return current, StatusChecksumDrift
		}
	}
	if current > set.TargetVersion {
		return current, StatusDatabaseTooNew
	}
	if current > 0 && current < set.MinCompatible {
		return current, StatusDatabaseTooOld
	}
	if current < set.TargetVersion {
		return current, StatusPending
	}
	return current, StatusCurrent
}

func toInternalDefs(set MigrationSet) []internalsqlite.MigrationDef {
	defs := make([]internalsqlite.MigrationDef, 0, len(set.Migrations))
	for _, m := range set.Migrations {
		defs = append(defs, internalsqlite.MigrationDef{
			Component: m.Component,
			Version:   m.Version,
			Name:      m.Name,
			Checksum:  m.Checksum,
			Up:        m.Up,
			Verify:    m.Verify,
		})
	}
	return defs
}

func (d *Database) MigrationPlan(ctx context.Context, sets ...MigrationSet) (Plan, error) {
	if err := validateMigrationSets(sets); err != nil {
		return Plan{}, err
	}
	var plan Plan
	for _, set := range sets {
		defs := toInternalDefs(set)
		records, err := internalsqlite.LoadRecords(ctx, d.db, set.Component)
		if err != nil {
			return Plan{}, err
		}
		current, kind := componentMigrationStatus(set, defs, records)
		plan.Components = append(plan.Components, ComponentPlan{
			Component: set.Component, CurrentVersion: current, TargetVersion: set.TargetVersion, Status: kind,
		})
		if kind != StatusCurrent && kind != StatusPending {
			return plan, &MigrationError{Kind: kind, Message: fmt.Sprintf("component %s cannot be migrated from version %d to %d", set.Component, current, set.TargetVersion)}
		}
		for _, m := range defs {
			if m.Version > current {
				plan.Entries = append(plan.Entries, PlanEntry{
					Component: m.Component,
					Version:   m.Version,
					Name:      m.Name,
					Checksum:  m.Checksum,
				})
			}
		}
	}
	return plan, nil
}

func (d *Database) MigrationStatus(ctx context.Context, sets ...MigrationSet) (Status, error) {
	if err := validateMigrationSets(sets); err != nil {
		return Status{}, err
	}
	var status Status
	for _, set := range sets {
		defs := toInternalDefs(set)
		records, err := internalsqlite.LoadRecords(ctx, d.db, set.Component)
		if err != nil {
			return Status{}, err
		}
		current, kind := componentMigrationStatus(set, defs, records)
		status.Components = append(status.Components, ComponentStatus{
			Component:      set.Component,
			CurrentVersion: current,
			TargetVersion:  set.TargetVersion,
			Status:         kind,
		})
	}
	return status, nil
}

func (d *Database) ApplyMigrations(ctx context.Context, sets ...MigrationSet) error {
	if err := validateMigrationSets(sets); err != nil {
		return err
	}
	for _, set := range sets {
		defs := toInternalDefs(set)
		if err := internalsqlite.ApplyMigrations(ctx, d.migrationDB, set.Component, defs, set.MinCompatible, set.TargetVersion); err != nil {
			return err
		}
	}
	return nil
}
