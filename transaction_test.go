package theory

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestTransaction(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	tx := NewTransaction(db)

	// Verify table exists
	var tableName string
	row := db.conn.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='test_users'")
	err := row.Scan(&tableName)
	if err != nil {
		t.Fatalf("failed to check table existence: %v", err)
	}
	if tableName != "test_users" {
		t.Fatal("test_users table does not exist")
	}

	t.Run("Basic Transaction", func(t *testing.T) {
		// Begin transaction
		txn, err := tx.Begin(ctx, &TxOptions{})
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		// Insert data
		user := &TestUser{
			Name:  "test1",
			Email: "test1@example.com",
		}
		err = txn.Create(ctx, user)
		if err != nil {
			t.Fatalf("failed to insert data: %v", err)
		}

		// Verify data was inserted using transaction's connection
		var count int
		rows, err := txn.QueryContext(ctx, "SELECT COUNT(*) FROM test_users WHERE name = ?", "test1")
		if err != nil {
			t.Fatalf("failed to query data: %v", err)
		}
		defer rows.Close()

		if rows.Next() {
			err = rows.Scan(&count)
			if err != nil {
				t.Fatalf("failed to scan count: %v", err)
			}
		}

		if count != 1 {
			t.Errorf("expected 1 row, got %d", count)
		}

		// Commit transaction
		err = txn.Commit(ctx)
		if err != nil {
			t.Fatalf("failed to commit transaction: %v", err)
		}
	})

	t.Run("Rollback Transaction", func(t *testing.T) {
		// Begin transaction
		txn, err := tx.Begin(ctx, &TxOptions{})
		if err != nil {
			t.Fatalf("failed to begin transaction: %v", err)
		}

		// Insert data
		user := &TestUser{
			Name:  "test2",
			Email: "test2@example.com",
		}
		err = txn.Create(ctx, user)
		if err != nil {
			t.Fatalf("failed to insert data: %v", err)
		}

		// Verify data was inserted using transaction's connection
		var count int
		rows, err := txn.QueryContext(ctx, "SELECT COUNT(*) FROM test_users WHERE name = ?", "test2")
		if err != nil {
			t.Fatalf("failed to query data: %v", err)
		}
		defer rows.Close()

		if rows.Next() {
			err = rows.Scan(&count)
			if err != nil {
				t.Fatalf("failed to scan count: %v", err)
			}
		}

		if count != 1 {
			t.Errorf("expected 1 row, got %d", count)
		}

		// Rollback transaction
		err = txn.Rollback(ctx)
		if err != nil {
			t.Fatalf("failed to rollback transaction: %v", err)
		}

		// Verify data was not inserted using DB connection since transaction is rolled back
		rows, err = db.conn.QueryContext(ctx, "SELECT COUNT(*) FROM test_users WHERE name = ?", "test2")
		if err != nil {
			t.Fatalf("failed to query data: %v", err)
		}
		defer rows.Close()

		if rows.Next() {
			err = rows.Scan(&count)
			if err != nil {
				t.Fatalf("failed to scan count: %v", err)
			}
		}

		if count != 0 {
			t.Errorf("expected 0 rows, got %d", count)
		}
	})

	t.Run("Nested Transaction", func(t *testing.T) {
		// Begin parent transaction
		parent, err := tx.Begin(ctx, &TxOptions{})
		if err != nil {
			t.Fatalf("failed to begin parent transaction: %v", err)
		}

		// Insert in parent
		parentUser := &TestUser{
			Name:  "parent",
			Email: "parent@example.com",
		}
		err = parent.Create(ctx, parentUser)
		if err != nil {
			t.Fatalf("failed to insert parent data: %v", err)
		}

		// Begin nested transaction
		nested, err := parent.Begin(ctx, &TxOptions{Nested: true})
		if err != nil {
			t.Fatalf("failed to begin nested transaction: %v", err)
		}

		// Insert in nested
		nestedUser := &TestUser{
			Name:  "nested",
			Email: "nested@example.com",
		}
		err = nested.Create(ctx, nestedUser)
		if err != nil {
			t.Fatalf("failed to insert nested data: %v", err)
		}

		// Verify nested data was inserted using nested transaction's connection
		var count int
		rows, err := nested.QueryContext(ctx, "SELECT COUNT(*) FROM test_users WHERE name = ?", "nested")
		if err != nil {
			t.Fatalf("failed to query nested data: %v", err)
		}
		defer rows.Close()

		if rows.Next() {
			err = rows.Scan(&count)
			if err != nil {
				t.Fatalf("failed to scan count: %v", err)
			}
		}

		if count != 1 {
			t.Errorf("expected 1 nested row, got %d", count)
		}

		// Rollback nested transaction
		err = nested.Rollback(ctx)
		if err != nil {
			t.Fatalf("failed to rollback nested transaction: %v", err)
		}

		// Verify nested data was rolled back using parent transaction's connection
		rows, err = parent.QueryContext(ctx, "SELECT COUNT(*) FROM test_users WHERE name = ?", "nested")
		if err != nil {
			t.Fatalf("failed to query nested data: %v", err)
		}
		defer rows.Close()

		if rows.Next() {
			err = rows.Scan(&count)
			if err != nil {
				t.Fatalf("failed to scan count: %v", err)
			}
		}

		if count != 0 {
			t.Errorf("expected 0 nested rows, got %d", count)
		}

		// Verify parent data still exists
		rows, err = parent.QueryContext(ctx, "SELECT COUNT(*) FROM test_users WHERE name = ?", "parent")
		if err != nil {
			t.Fatalf("failed to query parent data: %v", err)
		}
		defer rows.Close()

		if rows.Next() {
			err = rows.Scan(&count)
			if err != nil {
				t.Fatalf("failed to scan count: %v", err)
			}
		}

		if count != 1 {
			t.Errorf("expected 1 parent row, got %d", count)
		}

		// Commit parent transaction
		err = parent.Commit(ctx)
		if err != nil {
			t.Fatalf("failed to commit parent transaction: %v", err)
		}
	})
}
