package oracle

import (
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
			values                  = callbacks.ConvertToCreateValues(stmt)
			onConflict, hasConflict = stmt.Clauses["ON CONFLICT"].Expression.(clause.OnConflict)
		)

		if hasConflict {
			if len(stmtSchema.PrimaryFields) > 0 {
				// are all columns in value the primary fields in schema only?
				columnsMap := map[string]bool{}
				for _, column := range values.Columns {
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
			MergeCreate(db, onConflict, values)
		} else {
			stmt.AddClauseIfNotExists(clause.Insert{Table: clause.Table{Name: stmt.Schema.Table}})
			stmt.AddClause(clause.Values{Columns: values.Columns, Values: [][]interface{}{values.Values[0]}})
			stmt.Build("INSERT", "VALUES")
		}

		if !db.DryRun && db.Error == nil {
			for _, value := range values.Values {
				// HACK: replace values one by one, assuming its value layout will be the same all the time, i.e. aligned
				for idx, val := range value {
					val = ptrDereference(val)
					switch v := val.(type) {
					case bool:
						if v {
							val = 1
						} else {
							val = 0
						}
					default:
						val = convertCustomType(val)
					}

					stmt.Vars[idx] = val
				}
				// and then we insert each row one by one then put the returning values back (i.e. last return id => smart insert)
				// we keep track of the index so that the sub-reflected value is also correct

				// BIG BUG: what if any of the transactions failed? some result might already be inserted that oracle is so
				// sneaky that some transaction inserts will exceed the buffer and so will be pushed at unknown point,
				// resulting in dangling row entries, so we might need to delete them if an error happens

				result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)
				if db.AddError(err) == nil {
					// success
					db.RowsAffected, _ = result.RowsAffected()
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
