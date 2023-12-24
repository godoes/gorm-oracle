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
			comment := strings.ReplaceAll(comments[i], "'", "''")
			if err := m.RunWithValue(value, func(stmt *gorm.Statement) error {
				return tx.Exec(fmt.Sprintf("COMMENT ON TABLE ? IS '%s'", comment), m.CurrentTable(stmt)).Error
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

	if field.Unique {
		expr.SQL += " UNIQUE"
	}

	return
}

func (m Migrator) CurrentDatabase() (name string) {
	_ = m.DB.Raw(
		fmt.Sprintf(`SELECT ORA_DATABASE_NAME as "Current Database" FROM %s`, m.Dialector.(Dialector).DummyTableName()),
	).Row().Scan(&name)
	return
}

// GetTypeAliases return database type aliases
func (m Migrator) GetTypeAliases(databaseTypeName string) (types []string) {
	switch databaseTypeName {
	case "clob", "ocicloblocator":
		types = append(types, "clob", "ocicloblocator")
	case "nchar", "varchar", "varchar2":
		types = append(types, "nchar", "varchar", "varchar2")
	case "number", "integer", "smallint":
		types = append(types, "number", "integer", "smallint")
	case "timestampdty", "timestamp":
		types = append(types, "timestampdty", "timestamp")
	case "timestamptz_dty", "timestamp with time zone":
		types = append(types, "timestamptz_dty", "timestamp with time zone")
	default:
		return
	}
	return
}

func (m Migrator) CreateTable(values ...interface{}) error {
	for _, value := range values {
		_ = m.TryQuotifyReservedWords(value)
		_ = m.TryRemoveOnUpdate(value)
	}
	return m.Migrator.CreateTable(values...)
}

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

func (m Migrator) HasTable(value interface{}) bool {
	var count int64

	_ = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil && strings.Contains(stmt.Schema.Table, ".") {
			ownertable := strings.Split(stmt.Schema.Table, ".")
			return m.DB.Raw("SELECT COUNT(*) FROM ALL_TABLES WHERE OWNER = ?  and  TABLE_NAME = ?", ownertable[0], ownertable[1]).Row().Scan(&count)
		} else {
			return m.DB.Raw("SELECT COUNT(*) FROM USER_TABLES WHERE TABLE_NAME = ?", stmt.Table).Row().Scan(&count)
		}
	})

	return count > 0
}

// ColumnTypes return columnTypes []gorm.ColumnType and execErr error
func (m Migrator) ColumnTypes(value interface{}) ([]gorm.ColumnType, error) {
	columnTypes := make([]gorm.ColumnType, 0)
	execErr := m.RunWithValue(value, func(stmt *gorm.Statement) (err error) {
		rows, err := m.DB.Session(&gorm.Session{}).Table(stmt.Schema.Table).Where("ROWNUM = 1").Rows()
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

		for _, c := range rawColumnTypes {
			columnTypes = append(columnTypes, migrator.ColumnType{SQLColumnType: c})
		}

		return
	})

	return columnTypes, execErr
}

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

func (m Migrator) AddColumn(value interface{}, field string) error {
	return m.Migrator.AddColumn(value, field)
}

func (m Migrator) DropColumn(value interface{}, name string) error {
	return m.Migrator.DropColumn(value, name)
}

//goland:noinspection SqlNoDataSourceInspection
func (m Migrator) AlterColumn(value interface{}, field string) error {
	if !m.HasColumn(value, field) {
		return nil
	}

	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if field := stmt.Schema.LookUpField(field); field != nil {
			return m.DB.Exec(
				"ALTER TABLE ? MODIFY ? ?",
				clause.Table{Name: stmt.Schema.Table},
				clause.Column{Name: field.DBName},
				m.AlterDataTypeOf(stmt, field),
			).Error
		}
		return fmt.Errorf("failed to look up field with name: %s", field)
	})
}

func (m Migrator) HasColumn(value interface{}, field string) bool {
	var count int64
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if stmt.Schema != nil && strings.Contains(stmt.Schema.Table, ".") {
			ownertable := strings.Split(stmt.Schema.Table, ".")
			return m.DB.Raw("SELECT COUNT(*) FROM ALL_TAB_COLUMNS WHERE OWNER = ?  and TABLE_NAME = ? AND COLUMN_NAME = ?", ownertable[0], ownertable[1], field).Row().Scan(&count)
		} else {
			return m.DB.Raw("SELECT COUNT(*) FROM USER_TAB_COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?", stmt.Table, field).Row().Scan(&count)
		}

	}) == nil && count > 0
}

func (m Migrator) AlterDataTypeOf(stmt *gorm.Statement, field *schema.Field) (expr clause.Expr) {
	expr.SQL = m.DataTypeOf(field)

	var nullable = ""
	if stmt.Schema != nil && strings.Contains(stmt.Schema.Table, ".") {
		ownertable := strings.Split(stmt.Schema.Table, ".")
		_ = m.DB.Raw("SELECT NULLABLE FROM ALL_TAB_COLUMNS WHERE OWNER = ?  and TABLE_NAME = ? AND COLUMN_NAME = ?", ownertable[0], ownertable[1], field.DBName).Row().Scan(&nullable)
	} else {
		_ = m.DB.Raw("SELECT NULLABLE FROM USER_TAB_COLUMNS WHERE TABLE_NAME = ? AND COLUMN_NAME = ?", stmt.Table, field.DBName).Row().Scan(&nullable)
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

func (m Migrator) CreateConstraint(value interface{}, name string) error {
	_ = m.TryRemoveOnUpdate(value)
	return m.Migrator.CreateConstraint(value, name)
}

//goland:noinspection SqlNoDataSourceInspection
func (m Migrator) DropConstraint(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		for _, chk := range stmt.Schema.ParseCheckConstraints() {
			if chk.Name == name {
				return m.DB.Exec(
					"ALTER TABLE ? DROP CHECK ?",
					clause.Table{Name: stmt.Schema.Table}, clause.Column{Name: name},
				).Error
			}
		}

		return m.DB.Exec(
			"ALTER TABLE ? DROP CONSTRAINT ?",
			clause.Table{Name: stmt.Schema.Table}, clause.Column{Name: name},
		).Error
	})
}

func (m Migrator) HasConstraint(value interface{}, name string) bool {
	var count int64
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		return m.DB.Raw(
			"SELECT COUNT(*) FROM USER_CONSTRAINTS WHERE TABLE_NAME = ? AND CONSTRAINT_NAME = ?", stmt.Table, name,
		).Row().Scan(&count)
	}) == nil && count > 0
}

func (m Migrator) DropIndex(value interface{}, name string) error {
	return m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Exec("DROP INDEX ?", clause.Column{Name: name}, clause.Table{Name: stmt.Schema.Table}).Error
	})
}

func (m Migrator) HasIndex(value interface{}, name string) bool {
	var count int64
	_ = m.RunWithValue(value, func(stmt *gorm.Statement) error {
		if idx := stmt.Schema.LookIndex(name); idx != nil {
			name = idx.Name
		}

		return m.DB.Raw(
			"SELECT COUNT(*) FROM USER_INDEXES WHERE TABLE_NAME = ? AND INDEX_NAME = ?",
			m.Migrator.DB.NamingStrategy.TableName(stmt.Table),
			m.Migrator.DB.NamingStrategy.IndexName(stmt.Table, name),
		).Row().Scan(&count)
	})

	return count > 0
}

// RenameIndex https://docs.oracle.com/database/121/SPATL/alter-index-rename.htm
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
			for idx, v := range stmt.Schema.DBNames {
				if IsReservedWord(v) {
					stmt.Schema.DBNames[idx] = strconv.Quote(v)
				}
			}

			for _, v := range stmt.Schema.Fields {
				if IsReservedWord(v.DBName) {
					v.DBName = strconv.Quote(v.DBName)
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}
