package query

import (
	"fmt"
	"strings"
)

// Builder represents a SQL query builder
type Builder struct {
	table     string
	columns   []string
	where     []string
	args      []interface{}
	orderBy   string
	limit     int
	offset    int
	operation string
}

// NewBuilder creates a new query builder for the specified table
func NewBuilder(table string) *Builder {
	return &Builder{
		table:   table,
		columns: make([]string, 0),
		where:   make([]string, 0),
		args:    make([]interface{}, 0),
	}
}

// Select sets the columns to be selected
func (b *Builder) Select(columns ...string) *Builder {
	b.operation = "SELECT"
	b.columns = columns
	return b
}

// Where adds a WHERE clause to the query
func (b *Builder) Where(condition string, args ...interface{}) *Builder {
	b.where = append(b.where, condition)
	b.args = append(b.args, args...)
	return b
}

// OrderBy adds an ORDER BY clause to the query
func (b *Builder) OrderBy(orderBy string) *Builder {
	b.orderBy = orderBy
	return b
}

// Limit adds a LIMIT clause to the query
func (b *Builder) Limit(limit int) *Builder {
	b.limit = limit
	return b
}

// Offset adds an OFFSET clause to the query
func (b *Builder) Offset(offset int) *Builder {
	b.offset = offset
	return b
}

// Build constructs and returns the SQL query and its arguments
func (b *Builder) Build() (string, []interface{}) {
	var query strings.Builder

	switch b.operation {
	case "SELECT":
		query.WriteString("SELECT ")
		if len(b.columns) == 0 {
			query.WriteString("*")
		} else {
			query.WriteString(strings.Join(b.columns, ", "))
		}
		query.WriteString(" FROM ")
		query.WriteString(b.table)
	}

	if len(b.where) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(b.where, " AND "))
	}

	if b.orderBy != "" {
		query.WriteString(" ORDER BY ")
		query.WriteString(b.orderBy)
	}

	if b.limit > 0 {
		query.WriteString(fmt.Sprintf(" LIMIT %d", b.limit))
	}

	if b.offset > 0 {
		query.WriteString(fmt.Sprintf(" OFFSET %d", b.offset))
	}

	return query.String(), b.args
}
