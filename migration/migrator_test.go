package migration

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestMigrator(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrator := NewMigrator(db)

	// Test Initialize
	err := migrator.Initialize()
	if err != nil {
		t.Fatalf("Migrator.Initialize() error = %v", err)
	}

	// Create test migrations
	migration1 := NewMigration("create_users")
	migration1.Up = []Operation{
		&CreateTable{
			Name: "users",
			Columns: []Column{
				{Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
				{Name: "name", Type: "TEXT"},
			},
		},
	}
	migration1.Down = []Operation{
		&DropTable{Name: "users"},
	}

	migration2 := NewMigration("add_email_to_users")
	migration2.Up = []Operation{
		&AddColumn{
			Table:  "users",
			Column: Column{Name: "email", Type: "TEXT"},
		},
	}
	migration2.Down = []Operation{
		&DropColumn{
			Table:  "users",
			Column: "email",
		},
	}

	// Add migrations
	migrator.Add(migration1)
	migrator.Add(migration2)

	// Test Up
	err = migrator.Up()
	if err != nil {
		t.Fatalf("Migrator.Up() error = %v", err)
	}

	// Check if migrations were applied
	applied, err := migrator.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("Migrator.GetAppliedMigrations() error = %v", err)
	}

	if len(applied) != 2 {
		t.Errorf("len(applied) = %v, want %v", len(applied), 2)
	}

	// Test Down
	err = migrator.Down()
	if err != nil {
		t.Fatalf("Migrator.Down() error = %v", err)
	}

	// Check if last migration was rolled back
	applied, err = migrator.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("Migrator.GetAppliedMigrations() error = %v", err)
	}

	if len(applied) != 1 {
		t.Errorf("len(applied) = %v, want %v", len(applied), 1)
	}

	// Test Status
	status, err := migrator.Status()
	if err != nil {
		t.Fatalf("Migrator.Status() error = %v", err)
	}

	if len(status) != 2 {
		t.Errorf("len(status) = %v, want %v", len(status), 2)
	}

	if status[0].Migration.ID != migration1.ID {
		t.Errorf("status[0].Migration.ID = %v, want %v", status[0].Migration.ID, migration1.ID)
	}

	if status[1].Migration.ID != migration2.ID {
		t.Errorf("status[1].Migration.ID = %v, want %v", status[1].Migration.ID, migration2.ID)
	}
}

func TestMigratorErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	migrator := NewMigrator(db)

	// Test Initialize
	err := migrator.Initialize()
	if err != nil {
		t.Fatalf("Migrator.Initialize() error = %v", err)
	}

	// Create an invalid migration
	migration := NewMigration("invalid_migration")
	migration.Up = []Operation{
		&CreateTable{
			Name: "users",
			Columns: []Column{
				{Name: "id", Type: "INVALID_TYPE", IsPK: true},
			},
		},
	}

	migrator.Add(migration)

	// Test Up with invalid migration
	err = migrator.Up()
	if err == nil {
		t.Error("Migrator.Up() error = nil, want error")
	}

	// Check that no migrations were applied
	applied, err := migrator.GetAppliedMigrations()
	if err != nil {
		t.Fatalf("Migrator.GetAppliedMigrations() error = %v", err)
	}

	if len(applied) != 0 {
		t.Errorf("len(applied) = %v, want %v", len(applied), 0)
	}
}
