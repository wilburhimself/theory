package model

import (
	"reflect"
	"strings"
)

// Metadata holds the model's metadata information
type Metadata struct {
	TableName string
	Fields    []Field
}

// Field represents a model field's metadata
type Field struct {
	Name      string
	DBName    string
	Type      reflect.Type
	IsPK      bool
	IsAuto    bool
	IsNull    bool
	MaxLength int
}

// ExtractMetadata extracts metadata from a model struct using reflection
func ExtractMetadata(model interface{}) (*Metadata, error) {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return nil, ErrNotAStruct
	}

	metadata := &Metadata{
		TableName: getTableName(t),
		Fields:    make([]Field, 0),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		f := Field{
			Name:   field.Name,
			DBName: getDBFieldName(field),
			Type:   field.Type,
			IsPK:   strings.Contains(dbTag, "pk"),
			IsAuto: strings.Contains(dbTag, "auto"),
			IsNull: strings.Contains(dbTag, "null"),
		}

		metadata.Fields = append(metadata.Fields, f)
	}

	return metadata, nil
}

// getTableName extracts the table name from the model type
func getTableName(t reflect.Type) string {
	// First check if the model implements TableNamer interface
	if method, exists := t.MethodByName("TableName"); exists {
		if method.Type.NumIn() == 1 && method.Type.NumOut() == 1 && method.Type.Out(0).Kind() == reflect.String {
			// Create a new instance of the type to call the method
			v := reflect.New(t).Elem()
			result := v.Method(method.Index).Call(nil)
			return result[0].String()
		}
	}

	// Default to type name with 's' suffix
	return strings.ToLower(t.Name()) + "s"
}

// getDBFieldName extracts the database field name from struct field
func getDBFieldName(field reflect.StructField) string {
	dbTag := field.Tag.Get("db")
	if dbTag == "" {
		return strings.ToLower(field.Name)
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
