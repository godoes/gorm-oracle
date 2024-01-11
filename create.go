package oracle

import (
	"database/sql"
	"reflect"

	"github.com/sijms/go-ora/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
)

func Create(db *gorm.DB) {
	if db.Error != nil {
		return
	}

	stmt := db.Statement
	if stmt == nil {
		return
	}

	stmtSchema := stmt.Schema
	if stmtSchema == nil {
		return
	}

	if !stmt.Unscoped {
		for _, c := range stmtSchema.CreateClauses {
			stmt.AddClause(c)
		}
	}

	if stmt.SQL.Len() == 0 {
		var (
			createValues            = callbacks.ConvertToCreateValues(stmt)
			onConflict, hasConflict = stmt.Clauses["ON CONFLICT"].Expression.(clause.OnConflict)
		)

		if hasConflict {
			if len(stmtSchema.PrimaryFields) > 0 {
				columnsMap := map[string]bool{}
				for _, column := range createValues.Columns {
					columnsMap[column.Name] = true
				}

				for _, field := range stmtSchema.PrimaryFields {
					if _, ok := columnsMap[field.DBName]; !ok {
						hasConflict = false
					}
				}
			} else {
				hasConflict = false
			}
		}

		hasDefaultValues := len(stmtSchema.FieldsWithDefaultDBValue) > 0
		if hasConflict {
			MergeCreate(db, onConflict, createValues)
		} else {
			stmt.AddClauseIfNotExists(clause.Insert{Table: clause.Table{Name: stmt.Schema.Table}})
			stmt.AddClause(clause.Values{Columns: createValues.Columns, Values: [][]interface{}{createValues.Values[0]}})

			if hasDefaultValues {
				columns := make([]clause.Column, len(stmtSchema.FieldsWithDefaultDBValue))
				for idx, field := range stmtSchema.FieldsWithDefaultDBValue {
					columns[idx] = clause.Column{Name: field.DBName}
				}
				stmt.AddClauseIfNotExists(clause.Returning{Columns: columns})
			}
			stmt.Build("INSERT", "VALUES", "RETURNING")

			if hasDefaultValues {
				_, _ = stmt.WriteString(" INTO ")
				for idx, field := range stmtSchema.FieldsWithDefaultDBValue {
					if idx > 0 {
						_ = stmt.WriteByte(',')
					}
					stmt.AddVar(stmt, sql.Out{Dest: reflect.New(field.FieldType).Interface()})
				}
				_, _ = stmt.WriteString(" /*-sql.Out{}-*/")
			}
		}

		if !db.DryRun && db.Error == nil {
			if hasConflict {
				for i, val := range stmt.Vars {
					// HACK: replace values one by one, assuming its value layout will be the same all the time, i.e. aligned
					stmt.Vars[i] = convertValue(val)
				}

				result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
				if db.AddError(err) == nil {
					db.RowsAffected, _ = result.RowsAffected()
					// TODO: get merged returning
				}
			} else {
				for idx, values := range createValues.Values {
					for i, val := range values {
						// HACK: replace values one by one, assuming its value layout will be the same all the time, i.e. aligned
						stmt.Vars[i] = convertValue(val)
					}

					result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
					if db.AddError(err) == nil {
						db.RowsAffected, _ = result.RowsAffected()

						if hasDefaultValues {
							getDefaultValues(db, idx)
						}
					}
				}
			}
		}
	}
}

func MergeCreate(db *gorm.DB, onConflict clause.OnConflict, values clause.Values) {
	var dummyTable string
	switch d := ptrDereference(db.Dialector).(type) {
	case Dialector:
		dummyTable = d.DummyTableName()
	default:
		dummyTable = "DUAL"
	}

	_, _ = db.Statement.WriteString("MERGE INTO ")
	db.Statement.WriteQuoted(db.Statement.Table)
	_, _ = db.Statement.WriteString(" USING (")

	for idx, value := range values.Values {
		if idx > 0 {
			_, _ = db.Statement.WriteString(" UNION ALL ")
		}

		_, _ = db.Statement.WriteString("SELECT ")
		for i, v := range value {
			if i > 0 {
				_ = db.Statement.WriteByte(',')
			}
			column := values.Columns[i]
			db.Statement.AddVar(db.Statement, v)
			_, _ = db.Statement.WriteString(" AS ")
			db.Statement.WriteQuoted(column.Name)
		}
		_, _ = db.Statement.WriteString(" FROM ")
		_, _ = db.Statement.WriteString(dummyTable)
	}

	_, _ = db.Statement.WriteString(`) `)
	db.Statement.WriteQuoted("excluded")
	_, _ = db.Statement.WriteString(" ON (")

	var where clause.Where
	for _, field := range db.Statement.Schema.PrimaryFields {
		where.Exprs = append(where.Exprs, clause.Eq{
			Column: clause.Column{Table: db.Statement.Table, Name: field.DBName},
			Value:  clause.Column{Table: "excluded", Name: field.DBName},
		})
	}
	where.Build(db.Statement)
	_ = db.Statement.WriteByte(')')

	if len(onConflict.DoUpdates) > 0 {
		_, _ = db.Statement.WriteString(" WHEN MATCHED THEN UPDATE SET ")
		onConflict.DoUpdates.Build(db.Statement)
	}

	_, _ = db.Statement.WriteString(" WHEN NOT MATCHED THEN INSERT (")

	written := false
	for _, column := range values.Columns {
		if db.Statement.Schema.PrioritizedPrimaryField == nil || !db.Statement.Schema.PrioritizedPrimaryField.AutoIncrement || db.Statement.Schema.PrioritizedPrimaryField.DBName != column.Name {
			if written {
				_ = db.Statement.WriteByte(',')
			}
			written = true
			db.Statement.WriteQuoted(column.Name)
		}
	}

	_, _ = db.Statement.WriteString(") VALUES (")

	written = false
	for _, column := range values.Columns {
		if db.Statement.Schema.PrioritizedPrimaryField == nil || !db.Statement.Schema.PrioritizedPrimaryField.AutoIncrement || db.Statement.Schema.PrioritizedPrimaryField.DBName != column.Name {
			if written {
				_ = db.Statement.WriteByte(',')
			}
			written = true
			db.Statement.WriteQuoted(clause.Column{
				Table: "excluded",
				Name:  column.Name,
			})
		}
	}
	_, _ = db.Statement.WriteString(")")
}

func convertValue(val interface{}) interface{} {
	val = ptrDereference(val)
	switch v := val.(type) {
	case bool:
		if v {
			val = 1
		} else {
			val = 0
		}
	case string:
		if len(v) > 2000 {
			val = go_ora.Clob{String: v, Valid: true}
		}
	default:
		val = convertCustomType(val)
	}
	return val
}

func getDefaultValues(db *gorm.DB, idx int) {
	insertTo := db.Statement.ReflectValue
	switch insertTo.Kind() {
	case reflect.Slice, reflect.Array:
		insertTo = insertTo.Index(idx)
	default:
	}

	for _, val := range db.Statement.Vars {
		switch v := val.(type) {
		case sql.Out:
			switch insertTo.Kind() {
			case reflect.Slice, reflect.Array:
				for i := insertTo.Len() - 1; i >= 0; i-- {
					rv := insertTo.Index(i)
					if reflect.Indirect(rv).Kind() != reflect.Struct {
						break
					}

					_, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, rv)
					if isZero {
						_ = db.AddError(db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, rv, v.Dest))
					}
				}
			case reflect.Struct:
				_, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, insertTo)
				if isZero {
					_ = db.AddError(db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, insertTo, v.Dest))
				}
			default:
			}
		default:
		}
	}
}
