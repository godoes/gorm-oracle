package oracle

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

// Migrator implement gorm migrator interface
type Migrator struct {
	migrator.Migrator
}

// AutoMigrate 自动迁移模型为表结构
//
//	// 迁移并设置单个表注释
//	db.Set("gorm:table_comments", "用户信息表").AutoMigrate(&User{})
//
//	// 迁移并设置多个表注释
//	db.Set("gorm:table_comments", []string{"用户信息表", "公司信息表"}).AutoMigrate(&User{}, &Company{})
func (m Migrator) AutoMigrate(dst ...interface{}) error {
	if err := m.Migrator.AutoMigrate(dst...); err != nil {
		return err
	}
	// set table comment
	if tableComments, ok := m.DB.Get("gorm:table_comments"); ok {
		var comments []string
		switch c := tableComments.(type) {
		case string:
			comments = append(comments, c)
		case []string:
			comments = c
		default:
			return nil
		}
		for i := 0; i < len(dst) && i < len(comments); i++ {
			value := dst[i]
			tx := m.DB.Session(&gorm.Session{})
			if err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
				return tx.Exec("COMMENT ON TABLE ? IS '?'", m.CurrentTable(stmt), GetStringExpr(comments[i])).Error
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// FullDataTypeOf returns field's db full data type
func (m Migrator) FullDataTypeOf(field *schema.Field) (expr clause.Expr) {
	expr.SQL = m.DataTypeOf(field)

	if field.HasDefaultValue && (field.DefaultValueInterface != nil || field.DefaultValue != "") {
		if field.DefaultValueInterface != nil {
			defaultStmt := &gorm.Statement{Vars: []interface{}{field.DefaultValueInterface}}
			m.Dialector.BindVarTo(defaultStmt, defaultStmt, field.DefaultValueInterface)
			expr.SQL += " DEFAULT " + m.Dialector.Explain(defaultStmt.SQL.String(), field.DefaultValueInterface)
		} else if field.DefaultValue != "(-)" {
			expr.SQL += " DEFAULT " + field.DefaultValue
		}
	}

	if field.NotNull {
		expr.SQL += " NOT NULL"
	}

	// see https://github.com/go-gorm/gorm/pull/6822
	//if field.Unique {
	//	expr.SQL += " UNIQUE"
	//}

	return
}

// CurrentDatabase returns current database name
func (m Migrator) CurrentDatabase() (name string) {
	_ = m.DB.Raw(
		fmt.Sprintf(`SELECT ORA_DATABASE_NAME as "Current Database" FROM %s`, m.Dialector.(Dialector).DummyTableName()),
	).Row().Scan(&name)
	return
}

// GetTypeAliases return database type aliases
func (m Migrator) GetTypeAliases(databaseTypeName string) (types []string) {
	switch databaseTypeName {
	case "blob", "raw", "longraw", "ocibloblocator", "ocifilelocator":
		types = append(types, "blob", "raw", "longraw", "ocibloblocator", "ocifilelocator")
	case "clob", "nclob", "longvarchar", "ocicloblocator":
		types = append(types, "clob", "nclob", "longvarchar", "ocicloblocator")
	case "char", "nchar", "varchar", "varchar2", "nvarchar2":
		types = append(types, "char", "nchar", "varchar", "varchar2", "nvarchar2")
	case "number", "integer", "smallint":
		types = append(types, "number", "integer", "smallint")
	case "decimal", "numeric", "ibfloat", "ibdouble":
		types = append(types, "decimal", "numeric", "ibfloat", "ibdouble")
	case "timestampdty", "timestamp", "date":
		types = append(types, "timestampdty", "timestamp", "date")
	case "timestamptz_dty", "timestamp with time zone":
		types = append(types, "timestamptz_dty", "timestamp with time zone")
	case "timestampltz_dty", "timestampeltz", "timestamp with local time zone":
		types = append(types, "timestampltz_dty", "timestampeltz", "timestamp with local time zone")
	default:
		return
	}
	return
}

// CreateTable create table in database for values
func (m Migrator) CreateTable(values ...interface{}) (err error) {
	ignoreCase := !m.Dialector.(Dialector).NamingCaseSensitive
	for _, value := range values {
		if ignoreCase {
			_ = m.TryQuotifyReservedWords(value)
		}
		_ = m.TryRemoveOnUpdate(value)
	}
	if err = m.Migrator.CreateTable(values...); err != nil {
		return
	}
	// set column comment
	for _, value := range m.ReorderModels(values, false) {
		if err = m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
			if stmt.Schema != nil {
				for _, fieldName := range stmt.Schema.DBNames {
					field := stmt.Schema.FieldsByDBName[fieldName]
					if err = m.setCommentForColumn(field, stmt); err != nil {
						return
					}
				}
			}
			return
		}); err != nil {
			return
		}
	}
	return
}

func (m Migrator) setCommentForColumn(field *schema.Field, stmt *gorm.Statement) (err error) {
	if field == nil || stmt == nil || field.Comment == "" {
		return
	}
	table := m.CurrentTable(stmt)
	column := clause.Column{Name: field.DBName}
	comment := GetStringExpr(field.Comment)
	err = m.DB.Exec("COMMENT ON COLUMN ?.? IS '?'", table, column, comment).Error
	return
}

// DropTable drop table for values
//
//goland:noinspection SqlNoDataSourceInspection
func (m Migrator) DropTable(values ...interface{}) error {
	values = m.ReorderModels(values, false)
	for i := len(values) - 1; i >= 0; i-- {
		value := values[i]
		tx := m.DB.Session(&gorm.Session{})
		if m.HasTable(value) {
			if err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
				return tx.Exec("DROP TABLE ? CASCADE CONSTRAINTS", clause.Table{Name: stmt.Table}).Error
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// HasTable returns table exists or not for value, value could be a struct or string
func (m Migrator) HasTable(value interface{}) bool {
	var count int64

	_ = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if ownerName, tableName := m.getSchemaTable(stmt); ownerName != "" {
			return m.DB.Raw("SELECT COUNT(*) FROM ALL_TABLES WHERE OWNER = ? and TABLE_NAME = ?", ownerName, tableName).Row().Scan(&count)
		} else {
			return m.DB.Raw("SELECT COUNT(*) FROM USER_TABLES WHERE TABLE_NAME = ?", tableName).Row().Scan(&count)
		}
	})

	return count > 0
}

func (m Migrator) getSchemaTable(stmt *gorm.Statement) (ownerName, tableName string) {
	if stmt == nil {
		return
	}
	if stmt.Schema == nil {
		tableName = stmt.Table
	} else {
		tableName = stmt.Schema.Table
		if strings.Contains(tableName, ".") {
			ownerTable := strings.Split(tableName, ".")
			ownerName, tableName = ownerTable[0], ownerTable[1]
		}
	}
	return
}

// ColumnTypes return columnTypes []gorm.ColumnType and execErr error
func (m Migrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	columnTypes := make([]gorm.ColumnType, 0)
	execErr := m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
		_, tableName := m.getSchemaTable(stmt)
		rows, err := m.DB.Session(&gorm.Session{}).Table(tableName).Where("ROWNUM = 1").Rows()
		if err != nil {
			return err
		}

		defer func() {
			err = rows.Close()
		}()

		var rawColumnTypes []*sql.ColumnType
		rawColumnTypes, err = rows.ColumnTypes()
		if err != nil {
			return err
		}

		ignoreCase := !m.Dialector.(Dialector).NamingCaseSensitive
		for _, c := range rawColumnTypes {
			columnType := migrator.ColumnType{SQLColumnType: c}
			if ignoreCase && IsReservedWord(c.Name()) {
				columnType.NameValue = sql.NullString{
					String: strconv.Quote(c.Name()),
					Valid:  true,
				}
			}
			columnTypes = append(columnTypes, columnType)
		}

		return
	})

	return columnTypes, execErr
}

// RenameTable rename table from oldName to newName
func (m Migrator) RenameTable(oldName, newName interface{}) (err error) {
	resolveTable := func(name interface{}) (result string, err error) {
		if v, ok := name.(string); ok {
			result = v
		} else {
			stmt := &gorm.Statement{DB: m.DB}
			if err = stmt.Parse(name); err == nil {
				result = stmt.Table
			}
		}
		return
	}

	var oldTable, newTable string

	if oldTable, err = resolveTable(oldName); err != nil {
		return
	}

	if newTable, err = resolveTable(newName); err != nil {
		return
	}

	if !m.HasTable(oldTable) {
		return
	}

	return m.DB.Exec("RENAME TABLE ? TO ?",
		clause.Table{Name: oldTable},
		clause.Table{Name: newTable},
	).Error
}

// GetTables returns tables under the current user database
func (m Migrator) GetTables() (tableList []string, err error) {
	err = m.DB.Raw(`SELECT TABLE_NAME FROM USER_TABLES
		WHERE TABLESPACE_NAME IS NOT NULL AND TABLESPACE_NAME <> 'SYSAUX'
			AND TABLE_NAME NOT LIKE 'AQ$%' AND TABLE_NAME NOT LIKE 'MVIEW$%' AND TABLE_NAME NOT LIKE 'ROLLING$%'
			AND TABLE_NAME NOT IN ('HELP', 'SQLPLUS_PRODUCT_PROFILE', 'LOGSTDBY$PARAMETERS', 'LOGMNRGGC_GTCS', 'LOGMNRGGC_GTLO', 'LOGMNR_PARAMETER$', 'LOGMNR_SESSION$', 'SCHEDULER_JOB_ARGS_TBL', 'SCHEDULER_PROGRAM_ARGS_TBL')
		`).Scan(&tableList).Error
	return
}

// AddColumn create "name" column for value
func (m Migrator) AddColumn(value interface{}, name string) (err error) {
	if err = m.Migrator.AddColumn(value, name); err != nil {
		return err
	}
	// set column comment
	err = m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
		if field := stmt.Schema.LookUpField(name); field != nil {
			if err = m.setCommentForColumn(field, stmt); err != nil {
				return
			}
		}
		return
	})
	return
}

// DropColumn drop value's "name" column
func (m Migrator) DropColumn(value interface{}, name string) error {
	return m.Migrator.DropColumn(value, name)
}

// AlterColumn alter value's "field" column's type based on schema definition
//
//goland:noinspection SqlNoDataSourceInspection
func (m Migrator) AlterColumn(value interface{}, field string) error {
	if !m.HasColumn(value, field) {
		return nil
	}

	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			_, tableName := m.getSchemaTable(stmt)
			return m.DB.Exec(
				"ALTER TABLE ? MODIFY ? ?",
				clause.Table{Name: tableName},
				clause.Column{Name: field.DBName},
				m.AlterDataTypeOf(stmt, field),
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

// HasColumn check has column "field" for value or not
func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if ownerName, tableName := m.getSchemaTable(stmt); ownerName != "" {
			return m.DB.Raw("SELECT COUNT(*) FROM ALL_TAB_COLUMNS WHERE OWNER = ? and TABLE_NAME = ? AND COLUMN_NAME = ?", ownerName, tableName, field).Row().Scan(&count)
		} else {
			return m.DB.Raw("SELECT COUNT(*) FROM USER_TAB_COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?", tableName, field).Row().Scan(&count)
		}

	}) == nil && count > 0
}

// MigrateColumn migrate column
func (m Migrator) MigrateColumn(value interface{}, field *schema.Field, columnType gorm.ColumnType) (err error) {
	if err = m.Migrator.MigrateColumn(value, field, columnType); err != nil {
		return
	}

	return m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
		var description string
		if ownerName, tableName := m.getSchemaTable(stmt); ownerName != "" {
			_ = m.DB.Raw(
				"SELECT COMMENTS FROM ALL_COL_COMMENTS WHERE OWNER = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?",
				ownerName, tableName, field.DBName,
			).Row().Scan(&description)
		} else {
			_ = m.DB.Raw(
				"SELECT COMMENTS FROM USER_COL_COMMENTS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?",
				tableName, field.DBName,
			).Row().Scan(&description)
		}
		if comment := field.Comment; comment != "" && comment != description {
			if err = m.setCommentForColumn(field, stmt); err != nil {
				return
			}
		}
		return
	})
}

func (m Migrator) AlterDataTypeOf(stmt *gorm.Statement, field *schema.Field) (expr clause.Expr) {
	expr.SQL = m.DataTypeOf(field)

	var nullable = ""
	if ownerName, tableName := m.getSchemaTable(stmt); ownerName != "" {
		_ = m.DB.Raw("SELECT NULLABLE FROM ALL_TAB_COLUMNS WHERE OWNER = ? and TABLE_NAME = ? AND COLUMN_NAME = ?", ownerName, tableName, field.DBName).Row().Scan(&nullable)
	} else {
		_ = m.DB.Raw("SELECT NULLABLE FROM USER_TAB_COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?", tableName, field.DBName).Row().Scan(&nullable)
	}

	if field.HasDefaultValue && (field.DefaultValueInterface != nil || field.DefaultValue != "") {
		if field.DefaultValueInterface != nil {
			defaultStmt := &gorm.Statement{Vars: []interface{}{field.DefaultValueInterface}}
			m.Dialector.BindVarTo(defaultStmt, defaultStmt, field.DefaultValueInterface)
			expr.SQL += " DEFAULT " + m.Dialector.Explain(defaultStmt.SQL.String(), field.DefaultValueInterface)
		} else if field.DefaultValue != "(-)" {
			expr.SQL += " DEFAULT " + field.DefaultValue
		}
	}

	if field.NotNull && nullable == "Y" {
		expr.SQL += " NOT NULL"
	}
	if field.Unique {
		expr.SQL += " UNIQUE"
	}
	return
}

// CreateConstraint create constraint
func (m Migrator) CreateConstraint(value interface{}, name string) error {
	_ = m.TryRemoveOnUpdate(value)
	return m.Migrator.CreateConstraint(value, name)
}

// DropConstraint drop constraint
//
//goland:noinspection SqlNoDataSourceInspection
func (m Migrator) DropConstraint(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		_, tableName := m.getSchemaTable(stmt)
		for _, chk := range stmt.Schema.ParseCheckConstraints() {
			if chk.Name == name {
				return m.DB.Exec(
					"ALTER TABLE ? DROP CHECK ?",
					clause.Table{Name: tableName}, clause.Column{Name: name},
				).Error
			}
		}

		return m.DB.Exec(
			"ALTER TABLE ? DROP CONSTRAINT ?",
			clause.Table{Name: tableName}, clause.Column{Name: name},
		).Error
	})
}

// HasConstraint check has constraint or not
func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			"SELECT COUNT(*) FROM USER_CONSTRAINTS WHERE TABLE_NAME = ? AND CONSTRAINT_NAME = ?", stmt.Table, name,
		).Row().Scan(&count)
	}) == nil && count > 0
}

// DropIndex drop index "name"
func (m Migrator) DropIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}
		_, tableName := m.getSchemaTable(stmt)

		return m.DB.Exec("DROP INDEX ?", clause.Column{Name: name}, clause.Table{Name: tableName}).Error
	})
}

// HasIndex check has index "name" or not
func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	_ = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Raw(
			"SELECT COUNT(*) FROM USER_INDEXES WHERE TABLE_NAME = ? AND INDEX_NAME = ?",
			m.Migrator.DB.NamingStrategy.TableName(stmt.Table),
			name,
		).Row().Scan(&count)
	})

	return count > 0
}

// RenameIndex rename index from oldName to newName
//
// see also:
// https://docs.oracle.com/database/121/SPATL/alter-index-rename.htm
func (m Migrator) RenameIndex(value interface{}, oldName, newName string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Exec(
			"ALTER INDEX ? RENAME TO ?",
			clause.Column{Name: oldName}, clause.Column{Name: newName},
		).Error
	})
}

func (m Migrator) TryRemoveOnUpdate(values ...interface{}) error {
	for _, value := range values {
		if err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
			for _, rel := range stmt.Schema.Relationships.Relations {
				constraint := rel.ParseConstraint()
				if constraint != nil {
					rel.Field.TagSettings["CONSTRAINT"] = strings.ReplaceAll(rel.Field.TagSettings["CONSTRAINT"], fmt.Sprintf("ON UPDATE %s", constraint.OnUpdate), "")
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func (m Migrator) TryQuotifyReservedWords(values ...interface{}) error {
	for _, value := range values {
		if err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
			ignoreCase := !m.Dialector.(Dialector).NamingCaseSensitive
			for idx, v := range stmt.Schema.DBNames {
				if ignoreCase {
					v = strings.ToUpper(v)
				}
				if IsReservedWord(v) {
					v = strconv.Quote(v)
				}
				stmt.Schema.DBNames[idx] = v
			}

			for _, v := range stmt.Schema.Fields {
				fieldDBName := v.DBName
				if ignoreCase {
					v.DBName = strings.ToUpper(v.DBName)
				}
				if IsReservedWord(v.DBName) {
					v.DBName = strconv.Quote(v.DBName)
				}
				delete(stmt.Schema.FieldsByDBName, fieldDBName)
				stmt.Schema.FieldsByDBName[v.DBName] = v
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
