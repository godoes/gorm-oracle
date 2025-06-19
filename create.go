package oracle

import (
	"reflect"

	"github.com/sijms/go-ora/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
)

func Create(db *gorm.DB) {
	if db.Error != nil || db.Statement == nil {
		return
	}

	stmt := db.Statement
	stmtSchema := stmt.Schema
	if stmtSchema != nil && !stmt.Unscoped {
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
			if stmtSchema != nil && len(stmtSchema.PrimaryFields) > 0 {
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

		if hasConflict {
			MergeCreate(db, onConflict, createValues)
		} else {
			stmt.AddClauseIfNotExists(clause.Insert{})
			stmt.AddClause(clause.Values{Columns: createValues.Columns, Values: [][]interface{}{createValues.Values[0]}})

			stmt.Build("INSERT", "VALUES")
			_ = outputInserted(db)
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
						rowsAffected, _ := result.RowsAffected()
						db.RowsAffected += rowsAffected

						if stmtSchema != nil && len(stmtSchema.FieldsWithDefaultDBValue) > 0 {
							getDefaultValues(db, idx)
						}
					}
				}
			}
		}
	}
}

func outputInserted(db *gorm.DB) (lenDefaultValue int) {
	stmtSchema := db.Statement.Schema
	if stmtSchema == nil {
		return
	}
	lenDefaultValue = len(stmtSchema.FieldsWithDefaultDBValue)
	if lenDefaultValue == 0 {
		return
	}
	columns := make([]clause.Column, lenDefaultValue)
	for idx, field := range stmtSchema.FieldsWithDefaultDBValue {
		columns[idx] = clause.Column{Name: field.DBName}
	}
	db.Statement.AddClauseIfNotExists(clause.Returning{Columns: columns})
	db.Statement.Build("RETURNING")

	_, _ = db.Statement.WriteString(" INTO ")
	for idx, field := range stmtSchema.FieldsWithDefaultDBValue {
		if idx > 0 {
			_ = db.Statement.WriteByte(',')
		}

		outVar := go_ora.Out{Dest: reflect.New(field.FieldType).Interface()}
		if field.Size > 0 {
			outVar.Size = field.Size
		}
		db.Statement.AddVar(db.Statement, outVar)
	}
	_, _ = db.Statement.WriteString(" /*-go_ora.Out{}-*/")
	return
}

func MergeCreate(db *gorm.DB, onConflict clause.OnConflict, values clause.Values) {
	dummyTable := getDummyTable(db)

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

func getDummyTable(db *gorm.DB) (dummyTable string) {
	switch d := ptrDereference(db.Dialector).(type) {
	case Dialector:
		dummyTable = d.DummyTableName()
	default:
		dummyTable = "DUAL"
	}
	return
}

func getDefaultValues(db *gorm.DB, idx int) {
	if db.Statement.Schema == nil || len(db.Statement.Schema.FieldsWithDefaultDBValue) == 0 {
		return
	}
	insertTo := db.Statement.ReflectValue
	switch insertTo.Kind() {
	case reflect.Slice, reflect.Array:
		insertTo = insertTo.Index(idx)
	default:
	}
	if insertTo.Kind() == reflect.Pointer {
		insertTo = insertTo.Elem()
	}

	for _, val := range db.Statement.Vars {
		switch v := val.(type) {
		case go_ora.Out:
			switch insertTo.Kind() {
			case reflect.Slice, reflect.Array:
				for i := insertTo.Len() - 1; i >= 0; i-- {
					rv := insertTo.Index(i)
					switch reflect.Indirect(rv).Kind() {
					case reflect.Struct:
						setStructFieldValue(db, rv, v)
					default:
					}
				}
			case reflect.Struct:
				setStructFieldValue(db, insertTo, v)
			default:
			}
		default:
		}
	}
}

func setStructFieldValue(db *gorm.DB, insertTo reflect.Value, out go_ora.Out) {
	if _, isZero := db.Statement.Schema.PrioritizedPrimaryField.ValueOf(db.Statement.Context, insertTo); !isZero {
		return
	}
	_ = db.AddError(db.Statement.Schema.PrioritizedPrimaryField.Set(db.Statement.Context, insertTo, out.Dest))
}
