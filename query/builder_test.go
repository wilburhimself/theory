package query

import (
	"reflect"
	"testing"
)

func TestBuilder_Select(t *testing.T) {
	tests := []struct {
		name       string
		table      string
		columns    []string
		where      []string
		whereArgs  []interface{}
		orderBy    string
		limit      int
		offset     int
		wantQuery  string
		wantArgs   []interface{}
	}{
		{
			name:      "select all columns",
			table:     "users",
			wantQuery: "SELECT * FROM users",
			wantArgs:  []interface{}{},
		},
		{
			name:      "select specific columns",
			table:     "users",
			columns:   []string{"id", "name", "email"},
			wantQuery: "SELECT id, name, email FROM users",
			wantArgs:  []interface{}{},
		},
		{
			name:      "select with where clause",
			table:     "users",
			columns:   []string{"id", "name"},
			where:     []string{"id = ?"},
			whereArgs: []interface{}{1},
			wantQuery: "SELECT id, name FROM users WHERE id = ?",
			wantArgs:  []interface{}{1},
		},
		{
			name:      "select with multiple where clauses",
			table:     "users",
			where:     []string{"id > ?", "name LIKE ?"},
			whereArgs: []interface{}{1, "%john%"},
			wantQuery: "SELECT * FROM users WHERE id > ? AND name LIKE ?",
			wantArgs:  []interface{}{1, "%john%"},
		},
		{
			name:      "select with order by",
			table:     "users",
			orderBy:   "name DESC",
			wantQuery: "SELECT * FROM users ORDER BY name DESC",
			wantArgs:  []interface{}{},
		},
		{
			name:      "select with limit",
			table:     "users",
			limit:     10,
			wantQuery: "SELECT * FROM users LIMIT 10",
			wantArgs:  []interface{}{},
		},
		{
			name:      "select with offset",
			table:     "users",
			offset:    5,
			wantQuery: "SELECT * FROM users OFFSET 5",
			wantArgs:  []interface{}{},
		},
		{
			name:      "select with limit and offset",
			table:     "users",
			limit:     10,
			offset:    5,
			wantQuery: "SELECT * FROM users LIMIT 10 OFFSET 5",
			wantArgs:  []interface{}{},
		},
		{
			name:      "select with all clauses",
			table:     "users",
			columns:   []string{"id", "name"},
			where:     []string{"age > ?", "status = ?"},
			whereArgs: []interface{}{18, "active"},
			orderBy:   "name ASC",
			limit:     10,
			offset:    20,
			wantQuery: "SELECT id, name FROM users WHERE age > ? AND status = ? ORDER BY name ASC LIMIT 10 OFFSET 20",
			wantArgs:  []interface{}{18, "active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewBuilder(tt.table)
			if len(tt.columns) > 0 {
				b.Select(tt.columns...)
			} else {
				b.Select()
			}

			for i, condition := range tt.where {
				b.Where(condition, tt.whereArgs[i])
			}

			if tt.orderBy != "" {
				b.OrderBy(tt.orderBy)
			}

			if tt.limit > 0 {
				b.Limit(tt.limit)
			}

			if tt.offset > 0 {
				b.Offset(tt.offset)
			}

			gotQuery, gotArgs := b.Build()
			if gotQuery != tt.wantQuery {
				t.Errorf("Builder.Build() gotQuery = %v, want %v", gotQuery, tt.wantQuery)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("Builder.Build() gotArgs = %v, want %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestBuilder_Chaining(t *testing.T) {
	// Test method chaining
	b := NewBuilder("users").
		Select("id", "name").
		Where("age > ?", 18).
		Where("status = ?", "active").
		OrderBy("name ASC").
		Limit(10).
		Offset(20)

	wantQuery := "SELECT id, name FROM users WHERE age > ? AND status = ? ORDER BY name ASC LIMIT 10 OFFSET 20"
	wantArgs := []interface{}{18, "active"}

	gotQuery, gotArgs := b.Build()
	if gotQuery != wantQuery {
		t.Errorf("Builder chaining gotQuery = %v, want %v", gotQuery, wantQuery)
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Errorf("Builder chaining gotArgs = %v, want %v", gotArgs, wantArgs)
	}
}
