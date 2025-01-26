package theory

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/wilburhimself/theory/migration"
	"github.com/wilburhimself/theory/model"
)

// DB represents a Theory database instance
type DB struct {
	conn     *sql.DB
	driver   string
	migrator *migration.Migrator
}

// Config holds database connection configuration
type Config struct {
	Driver string
	DSN    string
}

// ErrRecordNotFound is returned when a record is not found
var ErrRecordNotFound = fmt.Errorf("record not found")

// Connect establishes a database connection
func Connect(cfg Config) (*DB, error) {
	conn, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	err = conn.Ping()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{
		conn:   conn,
		driver: cfg.Driver,
	}

	// Initialize migrator
	db.migrator = migration.NewMigrator(conn)
	err = db.migrator.Initialize()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize migrator: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Migrator returns the database migrator
func (db *DB) Migrator() *migration.Migrator {
	return db.migrator
}

// AutoMigrate creates or updates database tables based on the given models
func (db *DB) AutoMigrate(models ...interface{}) error {
	for _, m := range models {
		// Create migration
		metadata, err := model.ExtractMetadata(m)
		if err != nil {
			return err
		}

		// Create table operation
		createTable := &migration.CreateTable{
			Name:    metadata.TableName,
			Columns: make([]migration.Column, 0),
		}

		// Convert model fields to columns
		for _, field := range metadata.Fields {
			col := migration.Column{
				Name:   field.DBName,
				Type:   migration.SqlType(field.Type),
				IsPK:   field.IsPK,
				IsAuto: field.IsAuto,
				IsNull: field.IsNull,
			}
			createTable.Columns = append(createTable.Columns, col)
		}

		// Create migration
		mig := migration.NewMigration(fmt.Sprintf("create_%s", metadata.TableName))
		mig.Up = []migration.Operation{createTable}
		mig.Down = []migration.Operation{
			&migration.DropTable{Name: metadata.TableName},
		}

		// Add and run migration
		db.migrator.Add(mig)
		err = db.migrator.Up()
		if err != nil {
			return err
		}
	}

	return nil
}

// Create inserts a new record into the database
func (db *DB) Create(ctx context.Context, m interface{}) error {
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
	result, err := db.conn.ExecContext(ctx, sql, values...)
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

// Find retrieves records from the database
func (db *DB) Find(ctx context.Context, dest interface{}, where string, args ...interface{}) error {
	// Get metadata from destination type
	destType := reflect.TypeOf(dest)
	if destType.Kind() != reflect.Ptr {
		return fmt.Errorf("destination must be a pointer")
	}

	elemType := destType.Elem()
	isSlice := elemType.Kind() == reflect.Slice
	if isSlice {
		elemType = elemType.Elem()
	}

	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	metadata, err := model.ExtractMetadata(reflect.New(elemType).Interface())
	if err != nil {
		return err
	}

	// Build query
	sql := fmt.Sprintf("SELECT * FROM %s", metadata.TableName)
	if where != "" {
		sql += " WHERE " + where
	}

	// Execute query
	rows, err := db.conn.QueryContext(ctx, sql, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	var results reflect.Value
	if isSlice {
		results = reflect.MakeSlice(reflect.SliceOf(elemType), 0, 0)
	}

	found := false
	for rows.Next() {
		found = true
		// Create a new instance of the model
		modelInstance := reflect.New(elemType)
		if modelInstance.Kind() == reflect.Ptr {
			modelInstance = modelInstance.Elem()
		}

		// Create a slice of pointers to scan into
		var scanDest []interface{}
		for _, field := range metadata.Fields {
			scanDest = append(scanDest, modelInstance.FieldByName(field.Name).Addr().Interface())
		}

		// Scan row into model
		err := rows.Scan(scanDest...)
		if err != nil {
			return err
		}

		if isSlice {
			results = reflect.Append(results, modelInstance)
		} else {
			reflect.ValueOf(dest).Elem().Set(modelInstance)
			break
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	if !isSlice && !found {
		return ErrRecordNotFound
	}

	if isSlice {
		reflect.ValueOf(dest).Elem().Set(results)
	}

	return nil
}

// First retrieves the first record matching the given ID
func (db *DB) First(ctx context.Context, dest interface{}, id interface{}) error {
	metadata, err := model.ExtractMetadata(dest)
	if err != nil {
		return err
	}

	// Find primary key field
	var pkField *model.Field
	for i := range metadata.Fields {
		if metadata.Fields[i].IsPK {
			pkField = &metadata.Fields[i]
			break
		}
	}

	if pkField == nil {
		return fmt.Errorf("no primary key field found")
	}

	err = db.Find(ctx, dest, fmt.Sprintf("%s = ?", pkField.DBName), id)
	if err == ErrRecordNotFound {
		return ErrRecordNotFound
	}
	return err
}

// Update updates a record in the database
func (db *DB) Update(ctx context.Context, m interface{}) error {
	metadata, err := model.ExtractMetadata(m)
	if err != nil {
		return err
	}

	// Build query
	var setColumns []string
	var values []interface{}
	var pkField *model.Field
	var pkValue interface{}

	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i := range metadata.Fields {
		field := &metadata.Fields[i]
		if field.IsPK {
			pkField = field
			pkValue = v.FieldByName(field.Name).Interface()
		} else {
			setColumns = append(setColumns, fmt.Sprintf("%s = ?", field.DBName))
			values = append(values, v.FieldByName(field.Name).Interface())
		}
	}

	if pkField == nil {
		return fmt.Errorf("no primary key field found")
	}

	// Add primary key value to values
	values = append(values, pkValue)

	sql := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?",
		metadata.TableName,
		strings.Join(setColumns, ", "),
		pkField.DBName,
	)

	// Execute query
	_, err = db.conn.ExecContext(ctx, sql, values...)
	return err
}

// Delete deletes a record from the database
func (db *DB) Delete(ctx context.Context, m interface{}) error {
	metadata, err := model.ExtractMetadata(m)
	if err != nil {
		return err
	}

	// Find primary key
	var pkField *model.Field
	var pkValue interface{}

	v := reflect.ValueOf(m)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	for i := range metadata.Fields {
		field := &metadata.Fields[i]
		if field.IsPK {
			pkField = field
			pkValue = v.FieldByName(field.Name).Interface()
			break
		}
	}

	if pkField == nil {
		return fmt.Errorf("no primary key field found")
	}

	sql := fmt.Sprintf("DELETE FROM %s WHERE %s = ?",
		metadata.TableName,
		pkField.DBName,
	)

	// Execute query
	_, err = db.conn.ExecContext(ctx, sql, pkValue)
	return err
}

// BeginTx starts a new transaction with the given options
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.conn.BeginTx(ctx, opts)
}

// Begin starts a new transaction with default options
func (db *DB) Begin(ctx context.Context) (*sql.Tx, error) {
	return db.conn.BeginTx(ctx, nil)
}

// BeginTx starts a new transaction with the given options
func (db *DB) BeginTxWithTxOptions(ctx context.Context, opts *TxOptions) (*Transaction, error) {
	tx := NewTransaction(db)
	return tx.Begin(ctx, opts)
}

// Begin starts a new transaction with default options
func (db *DB) BeginWithTxOptions(ctx context.Context) (*Transaction, error) {
	return db.BeginTxWithTxOptions(ctx, &TxOptions{})
}
