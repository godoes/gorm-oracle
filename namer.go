package oracle

import (
	"strings"

	"gorm.io/gorm/schema"
)

// Namer implement gorm schema namer interface
type Namer struct {
	// NamingStrategy use custom naming strategy in gorm.Config on initialize
	NamingStrategy schema.Namer
	// CaseSensitive determines whether naming is case-sensitive
	CaseSensitive bool
}

// Deprecated: As of v1.5.0, use the Namer.ConvertNameToFormat instead.
//
//goland:noinspection GoUnusedExportedFunction
func ConvertNameToFormat(x string) string {
	return (Namer{}).ConvertNameToFormat(x)
}

// ConvertNameToFormat return appropriate capitalization name based on CaseSensitive
func (n Namer) ConvertNameToFormat(x string) string {
	if n.CaseSensitive {
		return x
	}
	return strings.ToUpper(x)
}

// TableName convert string to table name
func (n Namer) TableName(table string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.TableName(table))
}

// SchemaName generate schema name from table name, don't guarantee it is the reverse value of TableName
func (n Namer) SchemaName(table string) string {
	return n.ConvertNameToFormat(n.NamingStrategy.SchemaName(table))
}

// ColumnName convert string to column name
func (n Namer) ColumnName(table, column string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.ColumnName(table, column))
}

// JoinTableName convert string to join table name
func (n Namer) JoinTableName(table string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.JoinTableName(table))
}

// RelationshipFKName generate fk name for relation
func (n Namer) RelationshipFKName(relationship schema.Relationship) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.RelationshipFKName(relationship))
}

// CheckerName generate checker name
func (n Namer) CheckerName(table, column string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.CheckerName(table, column))
}

// IndexName generate index name
func (n Namer) IndexName(table, column string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.IndexName(table, column))
}

// UniqueName generate unique constraint name
func (n Namer) UniqueName(table, column string) string {
	return n.ConvertNameToFormat(n.NamingStrategy.UniqueName(table, column))
}
