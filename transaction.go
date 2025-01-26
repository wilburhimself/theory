package theory

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/wilburhimself/theory/model"
)

// TxOptions wraps sql.TxOptions to provide additional transaction configuration
type TxOptions struct {
	sql.TxOptions
	// Nested determines if this should create a nested transaction using savepoints
	Nested bool
}

// Transaction represents a database transaction
type Transaction struct {
	db        *DB
	tx        *sql.Tx
	savepoint string
	mu        sync.RWMutex
	options   *TxOptions
	parent    *Transaction
}

// NewTransaction creates a new transaction instance
func NewTransaction(db *DB) *Transaction {
	return &Transaction{
		db: db,
	}
}

// Begin starts a new transaction. If there's an existing transaction and Nested is true,
// it creates a savepoint instead.
func (t *Transaction) Begin(ctx context.Context, opts *TxOptions) (*Transaction, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tx != nil {
		if opts != nil && opts.Nested {
			// Create a nested transaction using savepoint
			savepoint := fmt.Sprintf("sp_%p", t)
			_, err := t.tx.ExecContext(ctx, fmt.Sprintf("SAVEPOINT %s", savepoint))
			if err != nil {
				return nil, fmt.Errorf("failed to create savepoint: %w", err)
			}

			return &Transaction{
				db:        t.db,
				tx:        t.tx,
				savepoint: savepoint,
				options:   opts,
				parent:    t,
			}, nil
		}
		return nil, fmt.Errorf("transaction already started")
	}

	tx, err := t.db.conn.BeginTx(ctx, &opts.TxOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &Transaction{
		db:      t.db,
		tx:      tx,
		options: opts,
	}, nil
}

// Commit commits the transaction or releases the savepoint if this is a nested transaction
func (t *Transaction) Commit(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}

	if t.savepoint != "" {
		// Release savepoint for nested transaction
		_, err := t.tx.ExecContext(ctx, fmt.Sprintf("RELEASE SAVEPOINT %s", t.savepoint))
		if err != nil {
			return fmt.Errorf("failed to release savepoint: %w", err)
		}
		t.tx = nil
		return nil
	}

	err := t.tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	t.tx = nil
	return nil
}

// Rollback rolls back the transaction or rolls back to the savepoint if this is a nested transaction
func (t *Transaction) Rollback(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}

	if t.savepoint != "" {
		// Rollback to savepoint for nested transaction
		_, err := t.tx.ExecContext(ctx, fmt.Sprintf("ROLLBACK TO SAVEPOINT %s", t.savepoint))
		if err != nil {
			return fmt.Errorf("failed to rollback to savepoint: %w", err)
		}
		t.tx = nil
		return nil
	}

	err := t.tx.Rollback()
	if err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	t.tx = nil
	return nil
}

// ExecContext executes a query within the transaction
func (t *Transaction) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tx == nil {
		return nil, fmt.Errorf("no transaction in progress")
	}

	return t.tx.ExecContext(ctx, query, args...)
}

// QueryContext executes a query that returns rows within the transaction
func (t *Transaction) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tx == nil {
		return nil, fmt.Errorf("no transaction in progress")
	}

	return t.tx.QueryContext(ctx, query, args...)
}

// QueryRowContext executes a query that returns a single row within the transaction
func (t *Transaction) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tx == nil {
		return nil
	}

	return t.tx.QueryRowContext(ctx, query, args...)
}

// Create creates a new record in the database within the transaction
func (t *Transaction) Create(ctx context.Context, m interface{}) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.tx == nil {
		return fmt.Errorf("no transaction in progress")
	}

	// Get metadata from model
	metadata, err := model.ExtractMetadata(m)
	if err != nil {
		return err
	}

	// Build query
	var columns []string
	var placeholders []string
	var values []interface{}

	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for _, field := range metadata.Fields {
		if !field.IsAuto {
			columns = append(columns, field.DBName)
			placeholders = append(placeholders, "?")
			values = append(values, v.FieldByName(field.Name).Interface())
		}
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		metadata.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	// Execute query
	result, err := t.ExecContext(ctx, sql, values...)
	if err != nil {
		return err
	}

	// Get last insert ID if available
	if id, err := result.LastInsertId(); err == nil {
		for _, field := range metadata.Fields {
			if field.IsAuto {
				v.FieldByName(field.Name).SetInt(id)
				break
			}
		}
	}

	return nil
}

// InTransaction returns true if there is an active transaction
func (t *Transaction) InTransaction() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.tx != nil
}

// IsNested returns true if this is a nested transaction
func (t *Transaction) IsNested() bool {
	return t.savepoint != ""
}

// GetParent returns the parent transaction if this is a nested transaction
func (t *Transaction) GetParent() *Transaction {
	return t.parent
}
