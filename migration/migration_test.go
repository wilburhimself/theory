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
	Age       int       `db:"age"`
	CreatedAt time.Time `db:"created_at"`
}

func TestCreateTableFromModel(t *testing.T) {
	user := &TestUser{}
	op, err := CreateTableFromModel(user)
	if err != nil {
		t.Fatalf("CreateTableFromModel() error = %v", err)
	}

	// Check table name
	if op.Name != "test_user" {
		t.Errorf("CreateTableFromModel() table name = %v, want %v", op.Name, "test_user")
	}

	// Check columns
	wantColumns := []Column{
		{Name: "id", Type: "INTEGER", IsPK: true, IsAuto: true},
		{Name: "name", Type: "TEXT"},
		{Name: "email", Type: "TEXT", IsNull: true},
		{Name: "age", Type: "INTEGER"},
		{Name: "created_at", Type: "INTEGER"},
	}

	if len(op.Columns) != len(wantColumns) {
		t.Errorf("CreateTableFromModel() columns count = %v, want %v", len(op.Columns), len(wantColumns))
	}

	for i, want := range wantColumns {
		got := op.Columns[i]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("CreateTableFromModel() column[%d] = %v, want %v", i, got, want)
		}
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
					{Name: "email", Type: "TEXT", IsNull: true},
				},
			},
			wantSQL: "CREATE TABLE users (\n\tid INTEGER PRIMARY KEY AUTOINCREMENT,\n\tname TEXT NOT NULL,\n\temail TEXT\n)",
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
				Column: Column{Name: "age", Type: "INTEGER", IsNull: false},
			},
			wantSQL: "ALTER TABLE users ADD COLUMN age INTEGER NOT NULL",
		},
		{
			name: "drop column",
			operation: &DropColumn{
				Table:  "users",
				Column: "age",
			},
			wantSQL: "ALTER TABLE users DROP COLUMN age",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL := tt.operation.SQL()
			if gotSQL != tt.wantSQL {
				t.Errorf("SQL() = %v, want %v", gotSQL, tt.wantSQL)
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

	if m.ID == "" {
		t.Error("NewMigration() id is empty")
	}

	if !strings.HasSuffix(m.ID, name) {
		t.Errorf("NewMigration() id = %v, want suffix %v", m.ID, name)
	}

	if m.Timestamp.IsZero() {
		t.Error("NewMigration() timestamp is zero")
	}

	if len(m.Up) != 0 {
		t.Errorf("NewMigration() up = %v, want empty", m.Up)
	}

	if len(m.Down) != 0 {
		t.Errorf("NewMigration() down = %v, want empty", m.Down)
	}
}
