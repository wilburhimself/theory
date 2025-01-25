package migration

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Migrator handles database migrations
type Migrator struct {
	db         *sql.DB
	migrations []*Migration
}

// MigrationRecord represents a migration record in the database
type MigrationRecord struct {
	ID        string
	Name      string
	Timestamp time.Time
	Applied   time.Time
}

// NewMigrator creates a new migrator instance
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db:         db,
		migrations: make([]*Migration, 0),
	}
}

// Add adds a migration to the migrator
func (m *Migrator) Add(migration *Migration) {
	m.migrations = append(m.migrations, migration)
}

// Initialize creates the migrations table if it doesn't exist
func (m *Migrator) Initialize() error {
	sql := `
		CREATE TABLE IF NOT EXISTS migrations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			timestamp INTEGER NOT NULL,
			applied INTEGER NOT NULL
		)
	`
	_, err := m.db.Exec(sql)
	return err
}

// GetAppliedMigrations returns all applied migrations
func (m *Migrator) GetAppliedMigrations() ([]MigrationRecord, error) {
	rows, err := m.db.Query("SELECT id, name, timestamp, applied FROM migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		var timestamp, applied int64
		err := rows.Scan(&record.ID, &record.Name, &timestamp, &applied)
		if err != nil {
			return nil, err
		}
		record.Timestamp = time.Unix(timestamp, 0)
		record.Applied = time.Unix(applied, 0)
		records = append(records, record)
	}

	return records, rows.Err()
}

// validateSQLType checks if a SQL type is valid for SQLite
func (m *Migrator) validateSQLType(sqlType string) bool {
	validTypes := map[string]bool{
		"INTEGER": true,
		"REAL":    true,
		"TEXT":    true,
		"BLOB":    true,
		"NULL":    true,
	}
	return validTypes[strings.ToUpper(sqlType)]
}

// validateOperation checks if an operation is valid
func (m *Migrator) validateOperation(op Operation) error {
	switch o := op.(type) {
	case *CreateTable:
		for _, col := range o.Columns {
			if !m.validateSQLType(col.Type) {
				return fmt.Errorf("invalid SQL type '%s' for column '%s'", col.Type, col.Name)
			}
		}
	case *AddColumn:
		if !m.validateSQLType(o.Column.Type) {
			return fmt.Errorf("invalid SQL type '%s' for column '%s'", o.Column.Type, o.Column.Name)
		}
	case *ModifyColumn:
		if !m.validateSQLType(o.NewColumn.Type) {
			return fmt.Errorf("invalid SQL type '%s' for column '%s'", o.NewColumn.Type, o.NewColumn.Name)
		}
	}
	return nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	// Get applied migrations
	records, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	// Create a map of applied migration IDs
	applied := make(map[string]bool)
	for _, record := range records {
		applied[record.ID] = true
	}

	// Sort migrations by timestamp
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Timestamp.Before(m.migrations[j].Timestamp)
	})

	// Run pending migrations
	for _, migration := range m.migrations {
		if !applied[migration.ID] {
			// Validate operations before starting transaction
			for _, op := range migration.Up {
				if err := m.validateOperation(op); err != nil {
					return fmt.Errorf("invalid operation in migration %s: %w", migration.Name, err)
				}
			}

			// Start transaction
			tx, err := m.db.Begin()
			if err != nil {
				return err
			}

			// Run migration operations
			for _, op := range migration.Up {
				_, err := tx.Exec(op.SQL(), op.Args()...)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to run migration %s: %w", migration.Name, err)
				}
			}

			// Record migration
			_, err = tx.Exec(
				"INSERT INTO migrations (id, name, timestamp, applied) VALUES (?, ?, ?, ?)",
				migration.ID,
				migration.Name,
				migration.Timestamp.Unix(),
				time.Now().Unix(),
			)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to record migration %s: %w", migration.Name, err)
			}

			// Commit transaction
			err = tx.Commit()
			if err != nil {
				return fmt.Errorf("failed to commit migration %s: %w", migration.Name, err)
			}
		}
	}

	return nil
}

// Down rolls back the last migration
func (m *Migrator) Down() error {
	// Get applied migrations
	records, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil
	}

	// Find the last applied migration
	var lastRecord MigrationRecord
	for _, record := range records {
		if lastRecord.Applied.Before(record.Applied) {
			lastRecord = record
		}
	}

	// Find the migration
	var migration *Migration
	for _, m := range m.migrations {
		if m.ID == lastRecord.ID {
			migration = m
			break
		}
	}

	if migration == nil {
		return fmt.Errorf("migration %s not found", lastRecord.ID)
	}

	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	// Run migration operations
	for _, op := range migration.Down {
		_, err := tx.Exec(op.SQL(), op.Args()...)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to rollback migration %s: %w", migration.Name, err)
		}
	}

	// Remove migration record
	_, err = tx.Exec("DELETE FROM migrations WHERE id = ?", migration.ID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record %s: %w", migration.Name, err)
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit rollback of migration %s: %w", migration.Name, err)
	}

	return nil
}

// Status returns the status of all migrations
func (m *Migrator) Status() ([]struct {
	Migration *Migration
	Applied   *time.Time
}, error) {
	// Get applied migrations
	records, err := m.GetAppliedMigrations()
	if err != nil {
		return nil, err
	}

	// Create a map of applied migrations
	applied := make(map[string]time.Time)
	for _, record := range records {
		applied[record.ID] = record.Applied
	}

	// Sort migrations by timestamp
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Timestamp.Before(m.migrations[j].Timestamp)
	})

	// Create status list
	var status []struct {
		Migration *Migration
		Applied   *time.Time
	}

	for _, migration := range m.migrations {
		if appliedTime, ok := applied[migration.ID]; ok {
			status = append(status, struct {
				Migration *Migration
				Applied   *time.Time
			}{
				Migration: migration,
				Applied:   &appliedTime,
			})
		} else {
			status = append(status, struct {
				Migration *Migration
				Applied   *time.Time
			}{
				Migration: migration,
				Applied:   nil,
			})
		}
	}

	return status, nil
}
