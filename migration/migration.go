package migration

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/wilburhimself/theory/model"
)

// Migration represents a database migration
type Migration struct {
	ID        string
	Timestamp time.Time
	Name      string
	Up        []Operation
	Down      []Operation
}

// Operation represents a migration operation
type Operation interface {
	SQL() string
	Args() []interface{}
}

// CreateTable operation creates a new table
type CreateTable struct {
	Name       string
	Columns    []Column
	ForeignKeys []ForeignKey
	Indexes    []Index
}

// Column represents a table column
type Column struct {
	Name      string
	Type      string
	IsPK      bool
	IsAuto    bool
	IsNull    bool
	MaxLength int
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Columns      []string
	RefTable     string
	RefColumns   []string
	OnDelete     string // CASCADE, SET NULL, RESTRICT, NO ACTION
	OnUpdate     string // CASCADE, SET NULL, RESTRICT, NO ACTION
}

// Index represents a table index
type Index struct {
	Name      string
	Columns   []string
	IsUnique  bool
}

// DropTable operation drops a table
type DropTable struct {
	Name string
}

// AddColumn operation adds a column to a table
type AddColumn struct {
	Table  string
	Column Column
}

// DropColumn operation drops a column from a table
type DropColumn struct {
	Table  string
	Column string
}

// ModifyColumn operation modifies a column in a table
type ModifyColumn struct {
	Table     string
	OldColumn string
	NewColumn Column
}

// CreateIndex operation creates an index
type CreateIndex struct {
	Table  string
	Index  Index
}

// DropIndex operation drops an index
type DropIndex struct {
	Table string
	Name  string
}

// AddForeignKey operation adds a foreign key constraint
type AddForeignKey struct {
	Table      string
	ForeignKey ForeignKey
}

// DropForeignKey operation drops a foreign key constraint
type DropForeignKey struct {
	Table string
	Name  string
}

// SQL generates SQL for CreateTable operation
func (op *CreateTable) SQL() string {
	var cols []string
	for _, col := range op.Columns {
		def := fmt.Sprintf("%s %s", col.Name, col.Type)
		if col.IsPK {
			if col.IsAuto {
				def += " PRIMARY KEY AUTOINCREMENT"
			} else {
				def += " PRIMARY KEY"
			}
		}
		if !col.IsPK && !col.IsNull {
			def += " NOT NULL"
		}
		cols = append(cols, def)
	}

	// Add foreign key constraints
	for _, fk := range op.ForeignKeys {
		def := fmt.Sprintf("FOREIGN KEY (%s) REFERENCES %s (%s)",
			strings.Join(fk.Columns, ", "),
			fk.RefTable,
			strings.Join(fk.RefColumns, ", "))
		
		if fk.OnDelete != "" {
			def += fmt.Sprintf(" ON DELETE %s", fk.OnDelete)
		}
		if fk.OnUpdate != "" {
			def += fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate)
		}
		cols = append(cols, def)
	}

	sql := fmt.Sprintf("CREATE TABLE %s (\n\t%s\n)", op.Name, strings.Join(cols, ",\n\t"))

	// Create indexes
	var indexes []string
	for _, idx := range op.Indexes {
		idxSQL := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
			map[bool]string{true: "UNIQUE ", false: ""}[idx.IsUnique],
			idx.Name,
			op.Name,
			strings.Join(idx.Columns, ", "))
		indexes = append(indexes, idxSQL)
	}

	if len(indexes) > 0 {
		sql += ";\n" + strings.Join(indexes, ";\n")
	}

	return sql
}

func (c *CreateTable) Args() []interface{} {
	return nil
}

// SQL generates SQL for DropTable operation
func (d *DropTable) SQL() string {
	return fmt.Sprintf("DROP TABLE %s", d.Name)
}

func (d *DropTable) Args() []interface{} {
	return nil
}

// SQL generates SQL for AddColumn operation
func (a *AddColumn) SQL() string {
	def := fmt.Sprintf("%s %s", a.Column.Name, a.Column.Type)
	if !a.Column.IsNull {
		def += " NOT NULL"
	}
	return fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", a.Table, def)
}

func (a *AddColumn) Args() []interface{} {
	return nil
}

// SQL generates SQL for DropColumn operation
func (d *DropColumn) SQL() string {
	return fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", d.Table, d.Column)
}

func (d *DropColumn) Args() []interface{} {
	return nil
}

// SQL generates SQL for ModifyColumn operation
func (m *ModifyColumn) SQL() string {
	return fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", m.Table, m.OldColumn, m.NewColumn.Name)
}

func (m *ModifyColumn) Args() []interface{} {
	return nil
}

// SQL generates SQL for CreateIndex operation
func (c *CreateIndex) SQL() string {
	return fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
		map[bool]string{true: "UNIQUE ", false: ""}[c.Index.IsUnique],
		c.Index.Name,
		c.Table,
		strings.Join(c.Index.Columns, ", "))
}

func (c *CreateIndex) Args() []interface{} {
	return nil
}

// SQL generates SQL for DropIndex operation
func (d *DropIndex) SQL() string {
	return fmt.Sprintf("DROP INDEX %s", d.Name)
}

func (d *DropIndex) Args() []interface{} {
	return nil
}

// SQL generates SQL for AddForeignKey operation
func (a *AddForeignKey) SQL() string {
	sql := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s_%s_fk FOREIGN KEY (%s) REFERENCES %s (%s)",
		a.Table,
		a.Table,
		strings.Join(a.ForeignKey.Columns, "_"),
		strings.Join(a.ForeignKey.Columns, ", "),
		a.ForeignKey.RefTable,
		strings.Join(a.ForeignKey.RefColumns, ", "))

	if a.ForeignKey.OnDelete != "" {
		sql += fmt.Sprintf(" ON DELETE %s", a.ForeignKey.OnDelete)
	}
	if a.ForeignKey.OnUpdate != "" {
		sql += fmt.Sprintf(" ON UPDATE %s", a.ForeignKey.OnUpdate)
	}

	return sql
}

func (a *AddForeignKey) Args() []interface{} {
	return nil
}

// SQL generates SQL for DropForeignKey operation
func (d *DropForeignKey) SQL() string {
	return fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT %s", d.Table, d.Name)
}

func (d *DropForeignKey) Args() []interface{} {
	return nil
}

// NewMigration creates a new migration with the given name
func NewMigration(name string) *Migration {
	return &Migration{
		ID:        fmt.Sprintf("%d_%s", time.Now().Unix(), name),
		Timestamp: time.Now(),
		Name:      name,
		Up:        make([]Operation, 0),
		Down:      make([]Operation, 0),
	}
}

// SqlType converts a Go type to SQL type
func SqlType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "INTEGER"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.String:
		return "TEXT"
	case reflect.Bool:
		return "INTEGER"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "BLOB"
		}
	case reflect.Struct:
		if t == reflect.TypeOf(time.Time{}) {
			return "INTEGER" // Store as Unix timestamp
		}
	}
	return "TEXT"
}

// CreateTableFromModel creates a CreateTable operation from a model
func CreateTableFromModel(m interface{}) (*CreateTable, error) {
	metadata, err := model.ExtractMetadata(m)
	if err != nil {
		return nil, err
	}

	var columns []Column
	for _, field := range metadata.Fields {
		col := Column{
			Name:   field.DBName,
			Type:   SqlType(field.Type),
			IsPK:   field.IsPK,
			IsAuto: field.IsAuto,
			IsNull: field.IsNull,
		}
		columns = append(columns, col)
	}

	return &CreateTable{
		Name:    metadata.TableName,
		Columns: columns,
	}, nil
}

// generateID generates a unique ID for a migration
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
