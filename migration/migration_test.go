package migration

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

type TestUser struct {
	ID        int       `db:"id,pk,auto"`
	Name      string    `db:"name"`
	Email     string    `db:"email,null"`
	CreatedAt time.Time `db:"created_at"`
}

func TestCreateTableFromModel(t *testing.T) {
	user := &TestUser{}
	op, err := CreateTableFromModel(user)
	if err != nil {
		t.Fatalf("CreateTableFromModel() error = %v", err)
	}

	expectedColumns := []Column{
		{Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
		{Name: "name", Type: "TEXT"},
		{Name: "email", Type: "TEXT", IsNull: true},
		{Name: "created_at", Type: "INTEGER"},
	}

	if !reflect.DeepEqual(op.Columns, expectedColumns) {
		t.Errorf("CreateTableFromModel() got = %v, want %v", op.Columns, expectedColumns)
	}
}

func TestOperationSQL(t *testing.T) {
	tests := []struct {
		name      string
		operation Operation
		wantSQL   string
	}{
		{
			name: "create table",
			operation: &CreateTable{
				Name: "users",
				Columns: []Column{
					{Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
					{Name: "name", Type: "TEXT"},
				},
			},
			wantSQL: "CREATE TABLE users (\n\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\tname TEXT NOT NULL\n)",
		},
		{
			name: "drop table",
			operation: &DropTable{
				Name: "users",
			},
			wantSQL: "DROP TABLE users",
		},
		{
			name: "add column",
			operation: &AddColumn{
				Table:  "users",
				Column: Column{Name: "age", Type: "INTEGER"},
			},
			wantSQL: "ALTER TABLE users ADD COLUMN age INTEGER NOT NULL",
		},
		{
			name: "modify column",
			operation: &ModifyColumn{
				Table:     "users",
				OldColumn: "name",
				NewColumn: Column{Name: "full_name", Type: "TEXT"},
			},
			wantSQL: "ALTER TABLE users RENAME COLUMN name TO full_name",
		},
		{
			name: "create index",
			operation: &CreateIndex{
				Table: "users",
				Index: Index{
					Name:     "idx_users_email",
					Columns:  []string{"email"},
					IsUnique: true,
				},
			},
			wantSQL: "CREATE UNIQUE INDEX idx_users_email ON users (email)",
		},
		{
			name: "drop index",
			operation: &DropIndex{
				Table: "users",
				Name:  "idx_users_email",
			},
			wantSQL: "DROP INDEX idx_users_email",
		},
		{
			name: "add foreign key",
			operation: &AddForeignKey{
				Table: "posts",
				ForeignKey: ForeignKey{
					Columns:    []string{"user_id"},
					RefTable:   "users",
					RefColumns: []string{"id"},
					OnDelete:   "CASCADE",
				},
			},
			wantSQL: "ALTER TABLE posts ADD CONSTRAINT posts_user_id_fk FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE",
		},
		{
			name: "create table with foreign key",
			operation: &CreateTable{
				Name: "posts",
				Columns: []Column{
					{Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
					{Name: "title", Type: "TEXT"},
					{Name: "user_id", Type: "INTEGER"},
				},
				ForeignKeys: []ForeignKey{
					{
						Columns:    []string{"user_id"},
						RefTable:   "users",
						RefColumns: []string{"id"},
						OnDelete:   "CASCADE",
					},
				},
			},
			wantSQL: "CREATE TABLE posts (\n\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\ttitle TEXT NOT NULL,\n\tuser_id INTEGER NOT NULL,\n\tFOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE\n)",
		},
		{
			name: "create table with index",
			operation: &CreateTable{
				Name: "users",
				Columns: []Column{
					{Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
					{Name: "email", Type: "TEXT"},
				},
				Indexes: []Index{
					{
						Name:     "idx_users_email",
						Columns:  []string{"email"},
						IsUnique: true,
					},
				},
			},
			wantSQL: "CREATE TABLE users (\n\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\temail TEXT NOT NULL\n);\nCREATE UNIQUE INDEX idx_users_email ON users (email)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.operation.SQL(); got != tt.wantSQL {
				t.Errorf("SQL() = %v, want %v", got, tt.wantSQL)
			}
		})
	}
}

func TestNewMigration(t *testing.T) {
	name := "test_migration"
	m := NewMigration(name)

	if m.Name != name {
		t.Errorf("NewMigration() name = %v, want %v", m.Name, name)
	}

	if !strings.HasSuffix(m.ID, name) {
		t.Errorf("NewMigration() id = %v, want suffix %v", m.ID, name)
	}

	if len(m.Up) != 0 {
		t.Errorf("NewMigration() up = %v, want empty", m.Up)
	}

	if len(m.Down) != 0 {
		t.Errorf("NewMigration() down = %v, want empty", m.Down)
	}
}

func TestBatchMigrations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	migrator := NewMigrator(db)
	err := migrator.Initialize()
	if err != nil {
		t.Fatalf("failed to initialize migrator: %v", err)
	}

	// Create first batch of migrations
	m1 := NewMigration("create_users")
	m1.Up = []Operation{
		&CreateTable{
			Name: "users",
			Columns: []Column{
				{Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
				{Name: "name", Type: "TEXT"},
			},
		},
	}
	m1.Down = []Operation{
		&DropTable{Name: "users"},
	}

	m2 := NewMigration("add_user_email")
	m2.Up = []Operation{
		&AddColumn{
			Table:  "users",
			Column: Column{Name: "email", Type: "TEXT"},
		},
	}
	m2.Down = []Operation{
		&DropColumn{Table: "users", Column: "email"},
	}

	// Add and run first batch
	migrator.Add(m1)
	migrator.Add(m2)
	err = migrator.UpWithBatch(true)
	if err != nil {
		t.Fatalf("failed to run first batch: %v", err)
	}

	// Verify first batch
	status, err := migrator.Status()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
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

	// Create second batch of migrations
	m3 := NewMigration("add_user_index")
	m3.Up = []Operation{
		&CreateIndex{
			Table: "users",
			Index: Index{
				Name:     "idx_users_email",
				Columns:  []string{"email"},
				IsUnique: true,
			},
		},
	}
	m3.Down = []Operation{
		&DropIndex{Table: "users", Name: "idx_users_email"},
	}

	// Add and run second batch
	migrator.Add(m3)
	err = migrator.UpWithBatch(true)
	if err != nil {
		t.Fatalf("failed to run second batch: %v", err)
	}

	// Verify second batch
	status, err = migrator.Status()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
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

	// Roll back last batch
	err = migrator.DownWithBatch(true)
	if err != nil {
		t.Fatalf("failed to roll back: %v", err)
	}

	// Verify rollback
	status, err = migrator.Status()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
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
