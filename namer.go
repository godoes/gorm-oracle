package oracle

import (
	"strings"

	"gorm.io/gorm/schema"
)

type Namer struct {
	schema.NamingStrategy

	CaseSensitive bool // whether naming is case-sensitive
}

// Deprecated: As of v1.5.0, use the Namer.ConvertNameToFormat instead.
//
//goland:noinspection GoUnusedExportedFunction
func ConvertNameToFormat(x string) string {
	return (Namer{}).ConvertNameToFormat(x)
}

func (n Namer) ConvertNameToFormat(x string) string {
	if n.CaseSensitive {
		return x
	}
	return strings.ToUpper(x)
}

func (n Namer) TableName(table string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.TableName(table))
}

func (n Namer) ColumnName(table, column string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.ColumnName(table, column))
}

func (n Namer) JoinTableName(table string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.JoinTableName(table))
}

func (n Namer) RelationshipFKName(relationship schema.Relationship) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.RelationshipFKName(relationship))
}

func (n Namer) CheckerName(table, column string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.CheckerName(table, column))
}

func (n Namer) IndexName(table, column string) (name string) {
	return n.ConvertNameToFormat(n.NamingStrategy.IndexName(table, column))
}
