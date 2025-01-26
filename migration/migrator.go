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
	Batch     int
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
			applied INTEGER NOT NULL,
			batch INTEGER NOT NULL DEFAULT 1
		)
	`
	_, err := m.db.Exec(sql)
	return err
}

// validateSQLType checks if a SQL type is valid for SQLite
func (m *Migrator) validateSQLType(sqlType string) bool {
	validTypes := map[string]bool{
		"INTEGER": true,
		"TEXT":    true,
		"REAL":    true,
		"BLOB":    true,
	}
	return validTypes[strings.ToUpper(sqlType)]
}

// validateOperation checks if an operation is valid
func (m *Migrator) validateOperation(op Operation) error {
	switch o := op.(type) {
	case *CreateTable:
		for _, col := range o.Columns {
			if !m.validateSQLType(col.Type) {
				return fmt.Errorf("invalid SQL type %s", col.Type)
			}
		}
	case *AddColumn:
		if !m.validateSQLType(o.Column.Type) {
			return fmt.Errorf("invalid SQL type %s", o.Column.Type)
		}
	}
	return nil
}

// getNextBatchNumber gets the next batch number
func (m *Migrator) getNextBatchNumber() (int, error) {
	var batch int
	err := m.db.QueryRow("SELECT COALESCE(MAX(batch), 0) + 1 FROM migrations").Scan(&batch)
	if err != nil {
		return 0, err
	}
	return batch, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	return m.UpWithBatch(true)
}

// UpWithBatch runs all pending migrations, optionally using a transaction
func (m *Migrator) UpWithBatch(useTx bool) error {
	// Get applied migrations
	records, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	applied := make(map[string]bool)
	for _, record := range records {
		applied[record.ID] = true
	}

	// Sort migrations by timestamp
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Timestamp.Before(m.migrations[j].Timestamp)
	})

	// Get next batch number
	batch, err := m.getNextBatchNumber()
	if err != nil {
		return err
	}

	// Start transaction if requested
	var tx *sql.Tx
	if useTx {
		tx, err = m.db.Begin()
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()
	}

	// Run pending migrations
	for _, migration := range m.migrations {
		if !applied[migration.ID] {
			// Validate operations
			for _, op := range migration.Up {
				if err := m.validateOperation(op); err != nil {
					return fmt.Errorf("invalid operation in migration %s: %v", migration.Name, err)
				}
			}

			// Execute operations
			for _, op := range migration.Up {
				sql := op.SQL()
				if useTx {
					_, err = tx.Exec(sql)
				} else {
					_, err = m.db.Exec(sql)
				}
				if err != nil {
					return fmt.Errorf("failed to execute migration %s: %v", migration.Name, err)
				}
			}

			// Record migration
			now := time.Now().Unix()
			sql := `
				INSERT INTO migrations (id, name, timestamp, applied, batch)
				VALUES (?, ?, ?, ?, ?)
			`
			if useTx {
				_, err = tx.Exec(sql, migration.ID, migration.Name, migration.Timestamp.Unix(), now, batch)
			} else {
				_, err = m.db.Exec(sql, migration.ID, migration.Name, migration.Timestamp.Unix(), now, batch)
			}
			if err != nil {
				return fmt.Errorf("failed to record migration %s: %v", migration.Name, err)
			}
		}
	}

	// Commit transaction if used
	if useTx {
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}
	}

	return nil
}

// Down rolls back the last batch of migrations
func (m *Migrator) Down() error {
	return m.DownWithBatch(true)
}

// DownWithBatch rolls back the last batch of migrations, optionally using a transaction
func (m *Migrator) DownWithBatch(useTx bool) error {
	// Get applied migrations
	records, err := m.getAppliedMigrations()
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil
	}

	// Get last batch number
	lastBatch := records[len(records)-1].Batch

	// Filter migrations in last batch
	var toRollback []MigrationRecord
	for _, record := range records {
		if record.Batch == lastBatch {
			toRollback = append(toRollback, record)
		}
	}

	// Start transaction if requested
	var tx *sql.Tx
	if useTx {
		tx, err = m.db.Begin()
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()
	}

	// Roll back migrations in reverse order
	for i := len(toRollback) - 1; i >= 0; i-- {
		record := toRollback[i]

		// Find migration
		var migration *Migration
		for _, m := range m.migrations {
			if m.ID == record.ID {
				migration = m
				break
			}
		}
		if migration == nil {
			return fmt.Errorf("migration %s not found", record.ID)
		}

		// Execute down operations
		for _, op := range migration.Down {
			sql := op.SQL()
			if useTx {
				_, err = tx.Exec(sql)
			} else {
				_, err = m.db.Exec(sql)
			}
			if err != nil {
				return fmt.Errorf("failed to roll back migration %s: %v", migration.Name, err)
			}
		}

		// Remove migration record
		sql := "DELETE FROM migrations WHERE id = ?"
		if useTx {
			_, err = tx.Exec(sql, record.ID)
		} else {
			_, err = m.db.Exec(sql, record.ID)
		}
		if err != nil {
			return fmt.Errorf("failed to remove migration record %s: %v", migration.Name, err)
		}
	}

	// Commit transaction if used
	if useTx {
		err = tx.Commit()
		if err != nil {
			return fmt.Errorf("failed to commit transaction: %v", err)
		}
	}

	return nil
}

// Status returns the status of all migrations
func (m *Migrator) Status() ([]struct {
	Migration *Migration
	Applied   *time.Time
	Batch     int
}, error) {
	// Initialize migrations table if it doesn't exist
	err := m.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrations table: %v", err)
	}

	// Get applied migrations
	records, err := m.getAppliedMigrations()
	if err != nil {
		return nil, err
	}

	applied := make(map[string]struct {
		time  time.Time
		batch int
	})
	for _, record := range records {
		applied[record.ID] = struct {
			time  time.Time
			batch int
		}{
			time:  record.Applied,
			batch: record.Batch,
		}
	}

	// Build status
	var status []struct {
		Migration *Migration
		Applied   *time.Time
		Batch     int
	}

	for _, migration := range m.migrations {
		if record, ok := applied[migration.ID]; ok {
			appliedTime := record.time
			status = append(status, struct {
				Migration *Migration
				Applied   *time.Time
				Batch     int
			}{
				Migration: migration,
				Applied:   &appliedTime,
				Batch:     record.batch,
			})
		} else {
			status = append(status, struct {
				Migration *Migration
				Applied   *time.Time
				Batch     int
			}{
				Migration: migration,
				Applied:   nil,
				Batch:     0,
			})
		}
	}

	return status, nil
}

// getAppliedMigrations returns all applied migrations
func (m *Migrator) getAppliedMigrations() ([]MigrationRecord, error) {
	// Initialize migrations table if it doesn't exist
	err := m.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize migrations table: %v", err)
	}

	rows, err := m.db.Query(`
		SELECT id, name, timestamp, applied, batch
		FROM migrations
		ORDER BY timestamp ASC
	`)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		var timestamp, applied int64
		err := rows.Scan(&record.ID, &record.Name, &timestamp, &applied, &record.Batch)
		if err != nil {
			return nil, err
		}
		record.Timestamp = time.Unix(timestamp, 0)
		record.Applied = time.Unix(applied, 0)
		records = append(records, record)
	}

	return records, nil
}
