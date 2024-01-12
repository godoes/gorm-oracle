package oracle

import (
	"reflect"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

func Update(config *callbacks.Config) func(db *gorm.DB) {
	supportReturning := utils.Contains(config.UpdateClauses, "RETURNING")

	return func(db *gorm.DB) {
		if db.Error != nil {
			return
		}

		stmt := db.Statement
		if stmt == nil {
			return
		}

		if stmtSchema := stmt.Schema; stmtSchema != nil {
			for _, c := range stmtSchema.UpdateClauses {
				stmt.AddClause(c)
			}
		}

		if stmt.SQL.Len() == 0 {
			stmt.SQL.Grow(180)
			stmt.AddClauseIfNotExists(clause.Update{})
			if _, ok := stmt.Clauses["SET"]; !ok {
				if set := ConvertToAssignments(stmt); len(set) != 0 {
					defer delete(stmt.Clauses, "SET")
					stmt.AddClause(set)
				} else {
					return
				}
			}

			stmt.Build(stmt.BuildClauses...)
		}

		checkMissingWhereConditions(db)

		if !db.DryRun && db.Error == nil {
			for i, val := range stmt.Vars {
				// HACK: replace values one by one, assuming its value layout will be the same all the time, i.e. aligned
				stmt.Vars[i] = convertValue(val)
			}
			if ok, mode := hasReturning(db, supportReturning); ok {
				if rows, err := stmt.ConnPool.QueryContext(stmt.Context, stmt.SQL.String(), stmt.Vars...); db.AddError(err) == nil {
					dest := stmt.Dest
					stmt.Dest = stmt.ReflectValue.Addr().Interface()
					gorm.Scan(rows, db, mode)
					stmt.Dest = dest
					_ = db.AddError(rows.Close())
				}
			} else {
				result, err := stmt.ConnPool.ExecContext(stmt.Context, stmt.SQL.String(), stmt.Vars...)

				if db.AddError(err) == nil {
					db.RowsAffected, _ = result.RowsAffected()
				}
			}
		}
	}
}

func checkMissingWhereConditions(db *gorm.DB) {
	if !db.AllowGlobalUpdate && db.Error == nil {
		where, withCondition := db.Statement.Clauses["WHERE"]
		if withCondition {
			if _, withSoftDelete := db.Statement.Clauses["soft_delete_enabled"]; withSoftDelete {
				whereClause, _ := where.Expression.(clause.Where)
				withCondition = len(whereClause.Exprs) > 1
			}
		}
		if !withCondition {
			_ = db.AddError(gorm.ErrMissingWhereClause)
		}
		return
	}
}

func hasReturning(tx *gorm.DB, supportReturning bool) (bool, gorm.ScanMode) {
	if supportReturning {
		if c, ok := tx.Statement.Clauses["RETURNING"]; ok {
			returning, _ := c.Expression.(clause.Returning)
			if len(returning.Columns) == 0 || (len(returning.Columns) == 1 && returning.Columns[0].Name == "*") {
				return true, 0
			}
			return true, gorm.ScanUpdate
		}
	}
	return false, 0
}

// ConvertToAssignments convert to update assignments
func ConvertToAssignments(stmt *gorm.Statement) (set clause.Set) {
	var (
		selectColumns, restricted = stmt.SelectAndOmitColumns(false, true)
		assignValue               func(field *schema.Field, value interface{})
	)

	switch stmt.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		assignValue = func(field *schema.Field, value interface{}) {
			for i := 0; i < stmt.ReflectValue.Len(); i++ {
				if stmt.ReflectValue.CanAddr() {
					_ = field.Set(stmt.Context, stmt.ReflectValue.Index(i), value)
				}
			}
		}
	case reflect.Struct:
		assignValue = func(field *schema.Field, value interface{}) {
			if stmt.ReflectValue.CanAddr() {
				_ = field.Set(stmt.Context, stmt.ReflectValue, value)
			}
		}
	default:
		assignValue = func(field *schema.Field, value interface{}) {
		}
	}

	updatingValue := reflect.ValueOf(stmt.Dest)
	for updatingValue.Kind() == reflect.Ptr {
		updatingValue = updatingValue.Elem()
	}

	if !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
		switch stmt.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			if size := stmt.ReflectValue.Len(); size > 0 {
				var isZero bool
				for i := 0; i < size; i++ {
					for _, field := range stmt.Schema.PrimaryFields {
						_, isZero = field.ValueOf(stmt.Context, stmt.ReflectValue.Index(i))
						if !isZero {
							break
						}
					}
				}

				if !isZero {
					_, primaryValues := schema.GetIdentityFieldValuesMap(stmt.Context, stmt.ReflectValue, stmt.Schema.PrimaryFields)
					column, values := schema.ToQueryValues("", stmt.Schema.PrimaryFieldDBNames, primaryValues)
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.IN{Column: column, Values: values}}})
				}
			}
		case reflect.Struct:
			for _, field := range stmt.Schema.PrimaryFields {
				if value, isZero := field.ValueOf(stmt.Context, stmt.ReflectValue); !isZero {
					stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
				}
			}
		default:
		}
	}

	switch value := updatingValue.Interface().(type) {
	case map[string]interface{}:
		set = make([]clause.Assignment, 0, len(value))

		keys := make([]string, 0, len(value))
		for k := range value {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			kv := value[k]
			if _, ok := kv.(*gorm.DB); ok {
				kv = []interface{}{kv}
			}

			if stmt.Schema != nil {
				if field := stmt.Schema.LookUpField(k); field != nil {
					if field.DBName != "" {
						if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && !restricted) {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: kv})
							assignValue(field, value[k])
						}
					} else if v, ok := selectColumns[field.Name]; (ok && v) || (!ok && !restricted) {
						assignValue(field, value[k])
					}
					continue
				}
			}

			if v, ok := selectColumns[k]; (ok && v) || (!ok && !restricted) {
				set = append(set, clause.Assignment{Column: clause.Column{Name: k}, Value: kv})
			}
		}

		if !stmt.SkipHooks && stmt.Schema != nil {
			for _, dbName := range stmt.Schema.DBNames {
				field := stmt.Schema.LookUpField(dbName)
				if field.AutoUpdateTime > 0 && value[field.Name] == nil && value[field.DBName] == nil {
					if v, ok := selectColumns[field.DBName]; (ok && v) || !ok {
						now := stmt.DB.NowFunc()
						assignValue(field, now)

						if field.AutoUpdateTime == schema.UnixNanosecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.UnixNano()})
						} else if field.AutoUpdateTime == schema.UnixMillisecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.UnixNano() / 1e6})
						} else if field.AutoUpdateTime == schema.UnixSecond {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now.Unix()})
						} else {
							set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: now})
						}
					}
				}
			}
		}
	default:
		updatingSchema := stmt.Schema
		var isDiffSchema bool
		if !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
			// different schema
			updatingStmt := &gorm.Statement{DB: stmt.DB}
			if err := updatingStmt.Parse(stmt.Dest); err == nil {
				updatingSchema = updatingStmt.Schema
				isDiffSchema = true
			}
		}

		switch updatingValue.Kind() {
		case reflect.Struct:
			set = make([]clause.Assignment, 0, len(stmt.Schema.FieldsByDBName))
			for _, dbName := range stmt.Schema.DBNames {
				if field := updatingSchema.LookUpField(dbName); field != nil {
					if !field.PrimaryKey || !updatingValue.CanAddr() || stmt.Dest != stmt.Model {
						if v, ok := selectColumns[field.DBName]; (ok && v) || (!ok && (!restricted || (!stmt.SkipHooks && field.AutoUpdateTime > 0))) {
							value, isZero := field.ValueOf(stmt.Context, updatingValue)
							if !stmt.SkipHooks && field.AutoUpdateTime > 0 {
								if field.AutoUpdateTime == schema.UnixNanosecond {
									value = stmt.DB.NowFunc().UnixNano()
								} else if field.AutoUpdateTime == schema.UnixMillisecond {
									value = stmt.DB.NowFunc().UnixNano() / 1e6
								} else if field.AutoUpdateTime == schema.UnixSecond {
									value = stmt.DB.NowFunc().Unix()
								} else {
									value = stmt.DB.NowFunc()
								}
								isZero = false
							}

							if (ok || !isZero) && field.Updatable {
								value = convertCustomType(value)
								set = append(set, clause.Assignment{Column: clause.Column{Name: field.DBName}, Value: value})
								assignField := field
								if isDiffSchema {
									if originField := stmt.Schema.LookUpField(dbName); originField != nil {
										assignField = originField
									}
								}
								assignValue(assignField, value)
							}
						}
					} else {
						if value, isZero := field.ValueOf(stmt.Context, updatingValue); !isZero {
							stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Eq{Column: field.DBName, Value: value}}})
						}
					}
				}
			}
		default:
			_ = stmt.AddError(gorm.ErrInvalidData)
		}
	}

	return
}
