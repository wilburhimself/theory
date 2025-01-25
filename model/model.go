package model

import (
	"reflect"
	"strings"
	"unicode"
)

// Model represents a database model
type Model interface {
	TableName() string
	PrimaryKey() *Field
}

// Metadata holds the model's metadata information
type Metadata struct {
	TableName string
	Fields    []Field
}

// Field represents a model field's metadata
type Field struct {
	Name       string
	DBName     string
	Type       reflect.Type
	IsPK       bool
	IsAuto     bool
	IsNull     bool
	MaxLength  int
	IsPKHandled bool // Internal flag to track if PK is handled by Model interface
}

// MetadataProvider is an interface that models can implement to provide their own metadata
type MetadataProvider interface {
	ExtractMetadata() (*Metadata, error)
}

// ExtractMetadata extracts metadata from a model struct using reflection
func ExtractMetadata(m interface{}) (*Metadata, error) {
	if m == nil {
		return nil, &Error{Message: "nil model provided"}
	}

	// First check if the model implements MetadataProvider
	if provider, ok := m.(MetadataProvider); ok {
		return provider.ExtractMetadata()
	}

	t := reflect.TypeOf(m)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, ErrNotAStruct
	}

	metadata := &Metadata{
		TableName: getTableName(t, m),
		Fields:    make([]Field, 0),
	}

	// If the model implements Model interface, use its PrimaryKey method
	if model, ok := m.(Model); ok {
		if pk := model.PrimaryKey(); pk != nil {
			metadata.Fields = append(metadata.Fields, *pk)
			// Add a flag to indicate that primary key is already handled
			metadata.Fields[len(metadata.Fields)-1].IsPKHandled = true
		}
	}

	// Extract fields using reflection
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip if the field is already added (like primary key)
		if containsField(metadata.Fields, field.Name) {
			continue
		}

		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		f := Field{
			Name:   field.Name,
			DBName: getDBFieldName(field),
			Type:   field.Type,
		}

		// Parse db tag options
		if dbTag != "" {
			parts := strings.Split(dbTag, ",")
			for _, part := range parts[1:] { // Skip the first part (field name)
				switch part {
				case "pk":
					// If primary key is already handled, do not set IsPK to true
					if !containsPKHandledField(metadata.Fields, field.Name) {
						f.IsPK = true
					}
				case "auto":
					f.IsAuto = true
				case "null":
					f.IsNull = true
				}
			}
		}

		metadata.Fields = append(metadata.Fields, f)
	}

	return metadata, nil
}

// Helper function to check if a field name already exists in the fields slice
func containsField(fields []Field, name string) bool {
	for _, f := range fields {
		if f.Name == name {
			return true
		}
	}
	return false
}

// Helper function to check if a field with IsPKHandled flag exists in the fields slice
func containsPKHandledField(fields []Field, name string) bool {
	for _, f := range fields {
		if f.Name == name && f.IsPKHandled {
			return true
		}
	}
	return false
}

// PrimaryKey returns the primary key field of the model, if any
func (m *Metadata) PrimaryKey() *Field {
	for _, field := range m.Fields {
		if field.IsPK {
			return &field
		}
	}
	return nil
}

// getTableName extracts the table name from the model type
func getTableName(t reflect.Type, m interface{}) string {
	// First check if the model implements Model interface
	if model, ok := m.(Model); ok {
		return model.TableName()
	}

	// Convert CamelCase to snake_case
	name := t.Name()
	var result strings.Builder
	for i, r := range name {
		if i > 0 && 'A' <= r && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteByte(byte(unicode.ToLower(r)))
	}
	
	return result.String()
}

// getDBFieldName extracts the database field name from struct field
func getDBFieldName(field reflect.StructField) string {
	dbTag := field.Tag.Get("db")
	if dbTag == "" {
		// Convert field name to snake_case
		var result strings.Builder
		name := field.Name
		for i, r := range name {
			if i > 0 && 'A' <= r && r <= 'Z' {
				result.WriteByte('_')
			}
			result.WriteByte(byte(unicode.ToLower(r)))
		}
		return result.String()
	}

	parts := strings.Split(dbTag, ",")
	if parts[0] != "" {
		return parts[0]
	}

	return strings.ToLower(field.Name)
}

// Common errors
var (
	ErrNotAStruct = &Error{Message: "model must be a struct"}
)

// Error represents a model error
type Error struct {
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
