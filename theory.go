package theory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/wilburhimself/theory/model"
)

// DB represents a Theory database instance
type DB struct {
	conn *sql.DB
}

// Config holds the database configuration
type Config struct {
	Driver string
	DSN    string
}

// Connect establishes a connection to the database using the provided configuration
func Connect(driver, dsn string) (*DB, error) {
	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}

	if err := conn.Ping(); err != nil {
		return nil, err
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Create inserts a new record into the database
func (db *DB) Create(ctx context.Context, model interface{}) error {
	metadata, err := extractModelMetadata(model)
	if err != nil {
		return fmt.Errorf("failed to extract model metadata: %w", err)
	}

	query, args := db.buildInsertQuery(metadata, model)
	result, err := db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute insert query: %w", err)
	}

	// If the model has an auto-increment primary key, update it
	if id, err := result.LastInsertId(); err == nil {
		if pk := metadata.PrimaryKey(); pk != nil && pk.IsAuto {
			reflect.ValueOf(model).Elem().FieldByName(pk.Name).SetInt(id)
		}
	}

	return nil
}

// Find retrieves records from the database
func (db *DB) Find(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	return db.scanRows(rows, dest)
}

// Update updates records in the database
func (db *DB) Update(ctx context.Context, model interface{}) error {
	metadata, err := extractModelMetadata(model)
	if err != nil {
		return fmt.Errorf("failed to extract model metadata: %w", err)
	}

	query, args := db.buildUpdateQuery(metadata, model)
	_, err = db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute update query: %w", err)
	}

	return nil
}

// Delete removes records from the database
func (db *DB) Delete(ctx context.Context, model interface{}) error {
	metadata, err := extractModelMetadata(model)
	if err != nil {
		return fmt.Errorf("failed to extract model metadata: %w", err)
	}

	query, args := db.buildDeleteQuery(metadata, model)
	_, err = db.conn.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute delete query: %w", err)
	}

	return nil
}

// Helper methods for SQL generation

func (db *DB) buildInsertQuery(metadata *model.Metadata, model interface{}) (string, []interface{}) {
	val := reflect.ValueOf(model).Elem()
	var columns []string
	var placeholders []string
	var args []interface{}

	for _, field := range metadata.Fields {
		// Skip auto-increment primary keys
		if field.IsPK && field.IsAuto {
			continue
		}

		columns = append(columns, field.DBName)
		placeholders = append(placeholders, "?")
		args = append(args, val.FieldByName(field.Name).Interface())
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		metadata.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return query, args
}

func (db *DB) buildUpdateQuery(metadata *model.Metadata, model interface{}) (string, []interface{}) {
	val := reflect.ValueOf(model).Elem()
	var setColumns []string
	var args []interface{}

	// Build SET clause
	for _, field := range metadata.Fields {
		if field.IsPK {
			continue
		}
		setColumns = append(setColumns, fmt.Sprintf("%s = ?", field.DBName))
		args = append(args, val.FieldByName(field.Name).Interface())
	}

	// Add WHERE clause for primary key
	var whereClause string
	if pk := metadata.PrimaryKey(); pk != nil {
		whereClause = fmt.Sprintf("WHERE %s = ?", pk.DBName)
		args = append(args, val.FieldByName(pk.Name).Interface())
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s %s",
		metadata.TableName,
		strings.Join(setColumns, ", "),
		whereClause,
	)

	return query, args
}

func (db *DB) buildDeleteQuery(metadata *model.Metadata, model interface{}) (string, []interface{}) {
	val := reflect.ValueOf(model).Elem()
	var args []interface{}

	// Build WHERE clause for primary key
	var whereClause string
	if pk := metadata.PrimaryKey(); pk != nil {
		whereClause = fmt.Sprintf("WHERE %s = ?", pk.DBName)
		args = append(args, val.FieldByName(pk.Name).Interface())
	}

	query := fmt.Sprintf("DELETE FROM %s %s", metadata.TableName, whereClause)
	return query, args
}

func (db *DB) scanRows(rows *sql.Rows, dest interface{}) error {
	destVal := reflect.ValueOf(dest)
	if destVal.Kind() != reflect.Ptr {
		return errors.New("destination must be a pointer")
	}
	destVal = destVal.Elem()

	// Get the type of the slice elements
	sliceType := destVal.Type()
	if sliceType.Kind() != reflect.Slice {
		return errors.New("destination must be a pointer to slice")
	}
	elemType := sliceType.Elem()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get column names: %w", err)
	}

	for rows.Next() {
		// Create a new element
		elem := reflect.New(elemType).Elem()

		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		for i := range values {
			values[i] = new(interface{})
		}

		// Scan the row into the interface{} slice
		if err := rows.Scan(values...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Set the values on the struct fields
		for i, col := range columns {
			field := elem.FieldByName(toFieldName(col))
			if field.IsValid() {
				val := reflect.ValueOf(values[i]).Elem().Interface()
				if val != nil {
					field.Set(reflect.ValueOf(val))
				}
			}
		}

		// Append the element to the result slice
		destVal.Set(reflect.Append(destVal, elem))
	}

	return rows.Err()
}

// Helper function to convert database column names to struct field names
func toFieldName(column string) string {
	parts := strings.Split(column, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
}

// Helper function to extract model metadata
func extractModelMetadata(m interface{}) (*model.Metadata, error) {
	return model.ExtractMetadata(m)
}
