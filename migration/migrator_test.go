package migration

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db, func() {
		db.Close()
	}
}

func TestMigrator(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

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
		&DropColumn{Table: "users", Column: "email"},
	}

	// Test Add
	migrator.Add(migration1)
	migrator.Add(migration2)

	// Test Up with batch
	err = migrator.UpWithBatch(true)
	if err != nil {
		t.Fatalf("Migrator.UpWithBatch() error = %v", err)
	}

	// Verify migrations were applied
	status, err := migrator.Status()
	if err != nil {
		t.Fatalf("Migrator.Status() error = %v", err)
	}

	if len(status) != 2 {
		t.Errorf("got %d migrations, want 2", len(status))
	}

	for _, s := range status {
		if s.Applied == nil {
			t.Errorf("migration %s not applied", s.Migration.Name)
		}
		if s.Batch != 1 {
			t.Errorf("migration %s batch = %d, want 1", s.Migration.Name, s.Batch)
		}
	}

	// Create and run second batch
	migration3 := NewMigration("add_user_index")
	migration3.Up = []Operation{
		&CreateIndex{
			Table: "users",
			Index: Index{
				Name:     "idx_users_email",
				Columns:  []string{"email"},
				IsUnique: true,
			},
		},
	}
	migration3.Down = []Operation{
		&DropIndex{Table: "users", Name: "idx_users_email"},
	}

	migrator.Add(migration3)
	err = migrator.UpWithBatch(true)
	if err != nil {
		t.Fatalf("Migrator.UpWithBatch() error = %v", err)
	}

	// Verify second batch
	status, err = migrator.Status()
	if err != nil {
		t.Fatalf("Migrator.Status() error = %v", err)
	}

	if len(status) != 3 {
		t.Errorf("got %d migrations, want 3", len(status))
	}

	found := false
	for _, s := range status {
		if s.Migration.Name == "add_user_index" {
			found = true
			if s.Applied == nil {
				t.Error("migration add_user_index not applied")
			}
			if s.Batch != 2 {
				t.Errorf("migration add_user_index batch = %d, want 2", s.Batch)
			}
		}
	}

	if !found {
		t.Error("migration add_user_index not found")
	}

	// Test Down with batch
	err = migrator.DownWithBatch(true)
	if err != nil {
		t.Fatalf("Migrator.DownWithBatch() error = %v", err)
	}

	// Verify rollback
	status, err = migrator.Status()
	if err != nil {
		t.Fatalf("Migrator.Status() error = %v", err)
	}

	for _, s := range status {
		if s.Migration.Name == "add_user_index" {
			if s.Applied != nil {
				t.Error("migration add_user_index still applied after rollback")
			}
		} else {
			if s.Applied == nil {
				t.Errorf("migration %s rolled back unexpectedly", s.Migration.Name)
			}
		}
	}
}

func TestMigratorErrorHandling(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)

	// Test Initialize error handling
	err := migrator.Initialize()
	if err != nil {
		t.Fatalf("Migrator.Initialize() error = %v", err)
	}

	// Verify migrations table exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}

	if count > 0 {
		t.Error("migrations table contains rows after initialization")
	}

	// Test invalid SQL in migration
	invalidMigration := NewMigration("invalid_migration")
	invalidMigration.Up = []Operation{
		&CreateTable{
			Name: "users",
			Columns: []Column{
				{Name: "id", Type: "INVALID_TYPE"},
			},
		},
	}

	migrator.Add(invalidMigration)
	err = migrator.UpWithBatch(true)
	if err == nil {
		t.Error("Migrator.UpWithBatch() expected error for invalid SQL")
	}

	// Test transaction rollback
	status, err := migrator.Status()
	if err != nil {
		t.Fatalf("Migrator.Status() error = %v", err)
	}

	for _, s := range status {
		if s.Applied != nil {
			t.Error("migration applied despite error")
		}
	}

	// Verify migrations table is empty
	err = db.QueryRow("SELECT COUNT(*) FROM migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query migrations table: %v", err)
	}

	if count > 0 {
		t.Error("migrations table contains rows after failed migration")
	}
}
