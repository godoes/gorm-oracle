package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sijms/go-ora/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

type Config struct {
	DriverName        string
	DSN               string
	Conn              gorm.ConnPool //*sql.DB
	DefaultStringSize uint
	DBVer             string

	IgnoreCase          bool // warning: may cause performance issues
	NamingCaseSensitive bool // whether naming is case-sensitive
}

type Dialector struct {
	*Config
}

//goland:noinspection GoUnusedExportedFunction
func Open(dsn string) gorm.Dialector {
	return &Dialector{Config: &Config{DSN: dsn}}
}

//goland:noinspection GoUnusedExportedFunction
func New(config Config) gorm.Dialector {
	return &Dialector{Config: &config}
}

// BuildUrl create databaseURL from server, port, service, user, password, urlOptions
// this function help build a will formed databaseURL and accept any character as it
// convert special charters to corresponding values in URL
//
//goland:noinspection GoUnusedExportedFunction
func BuildUrl(server string, port int, service, user, password string, options map[string]string) string {
	return go_ora.BuildUrl(server, port, service, user, password, options)
}

func convertCustomType(val interface{}) interface{} {
	rv := reflect.ValueOf(val)
	ri := rv.Interface()
	typeName := reflect.TypeOf(ri).Name()
	if reflect.TypeOf(val).Kind() == reflect.Ptr {
		if rv.IsNil() {
			typeName = rv.Type().Elem().Name()
		} else {
			for rv.Kind() == reflect.Ptr {
				rv = rv.Elem()
			}
			ri = rv.Interface()
			typeName = reflect.TypeOf(ri).Name()
		}
	}
	if typeName == "DeletedAt" {
		// gorm.DeletedAt
		if rv.IsZero() {
			val = sql.NullTime{}
		} else {
			val = getTimeValue(ri.(gorm.DeletedAt).Time)
		}
	} else if m := rv.MethodByName("Time"); m.IsValid() && m.Type().NumIn() == 0 {
		// custom time type
		for _, result := range m.Call([]reflect.Value{}) {
			if reflect.TypeOf(result.Interface()).Name() == "Time" {
				val = getTimeValue(result.Interface().(time.Time))
			}
		}
	}
	return val
}

func getTimeValue(t time.Time) interface{} {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return t
}

func (d Dialector) DummyTableName() string {
	return "DUAL"
}

func (d Dialector) Name() string {
	return "oracle"
}

func (d Dialector) Initialize(db *gorm.DB) (err error) {
	db.NamingStrategy = Namer{
		NamingStrategy: db.NamingStrategy.(schema.NamingStrategy),
		CaseSensitive:  d.NamingCaseSensitive,
	}
	d.DefaultStringSize = 1024

	// register callbacks
	config := &callbacks.Config{
		CreateClauses: []string{"INSERT", "VALUES", "ON CONFLICT", "RETURNING"},
		UpdateClauses: []string{"UPDATE", "SET", "WHERE", "RETURNING"},
		DeleteClauses: []string{"DELETE", "FROM", "WHERE", "RETURNING"},
	}
	callbacks.RegisterDefaultCallbacks(db, config)

	d.DriverName = "oracle"

	if d.Conn != nil {
		db.ConnPool = d.Conn
	} else {
		db.ConnPool, err = sql.Open(d.DriverName, d.DSN)
		if err != nil {
			return
		}
	}
	if d.IgnoreCase {
		if sqlDB, ok := db.ConnPool.(*sql.DB); ok {
			// warning: may cause performance issues
			_ = go_ora.AddSessionParam(sqlDB, "NLS_COMP", "LINGUISTIC")
			_ = go_ora.AddSessionParam(sqlDB, "NLS_SORT", "BINARY_CI")
		}
	}
	err = db.ConnPool.QueryRowContext(context.Background(), "select version from product_component_version where rownum = 1").Scan(&d.DBVer)
	if err != nil {
		return err
	}
	//log.Println("DBVer:" + d.DBVer)
	if err = db.Callback().Create().Replace("gorm:create", Create); err != nil {
		return
	}
	if err = db.Callback().Update().Replace("gorm:update", Update(config)); err != nil {
		return
	}

	for k, v := range d.ClauseBuilders() {
		db.ClauseBuilders[k] = v
	}
	return
}

func (d Dialector) ClauseBuilders() map[string]clause.ClauseBuilder {
	dbVer, _ := strconv.Atoi(strings.Split(d.DBVer, ".")[0])
	if dbVer > 0 && dbVer < 12 {
		return map[string]clause.ClauseBuilder{
			"LIMIT": d.RewriteLimit11,
		}

	} else {
		return map[string]clause.ClauseBuilder{
			"LIMIT": d.RewriteLimit,
		}
	}

}

func (d Dialector) RewriteLimit(c clause.Clause, builder clause.Builder) {
	if limit, ok := c.Expression.(clause.Limit); ok {
		if stmt, ok := builder.(*gorm.Statement); ok {
			if _, ok := stmt.Clauses["ORDER BY"]; !ok {
				s := stmt.Schema
				_, _ = builder.WriteString("ORDER BY ")
				if s != nil && s.PrioritizedPrimaryField != nil {
					builder.WriteQuoted(s.PrioritizedPrimaryField.DBName)
					_ = builder.WriteByte(' ')
				} else {
					_, _ = builder.WriteString("(SELECT NULL FROM ")
					_, _ = builder.WriteString(d.DummyTableName())
					_, _ = builder.WriteString(")")
				}
			}
		}

		if offset := limit.Offset; offset > 0 {
			_, _ = builder.WriteString(" OFFSET ")
			_, _ = builder.WriteString(strconv.Itoa(offset))
			_, _ = builder.WriteString(" ROWS")
		}
		if limit := limit.Limit; limit != nil && *limit >= 0 {
			_, _ = builder.WriteString(" FETCH NEXT ")
			_, _ = builder.WriteString(strconv.Itoa(*limit))
			_, _ = builder.WriteString(" ROWS ONLY")
		}
	}
}

// RewriteLimit11 Oracle11 Limit
func (d Dialector) RewriteLimit11(c clause.Clause, builder clause.Builder) {
	if limit, ok := c.Expression.(clause.Limit); ok {
		if stmt, ok := builder.(*gorm.Statement); ok {
			limitSql := strings.Builder{}
			if limit := limit.Limit; *limit > 0 {
				if _, ok := stmt.Clauses["WHERE"]; !ok {
					limitSql.WriteString(" WHERE ")
				} else {
					limitSql.WriteString(" AND ")
				}
				limitSql.WriteString("ROWNUM <= ")
				limitSql.WriteString(strconv.Itoa(*limit))
			}
			if _, ok := stmt.Clauses["ORDER BY"]; !ok {
				_, _ = builder.WriteString(limitSql.String())
			} else {
				//  "ORDER BY" before  insert
				sqlTmp := strings.Builder{}
				sqlOld := stmt.SQL.String()
				orderIndex := strings.Index(sqlOld, "ORDER BY") - 1
				sqlTmp.WriteString(sqlOld[:orderIndex])
				sqlTmp.WriteString(limitSql.String())
				sqlTmp.WriteString(sqlOld[orderIndex:])
				log.Println(sqlTmp.String())
				stmt.SQL = sqlTmp
			}
		}
	}
}

func (d Dialector) DefaultValueOf(*schema.Field) clause.Expression {
	return clause.Expr{SQL: "VALUES (DEFAULT)"}
}

func (d Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return Migrator{
		Migrator: migrator.Migrator{
			Config: migrator.Config{
				DB:                          db,
				Dialector:                   d,
				CreateIndexAfterCreateTable: true,
			},
		},
	}
}

func (d Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, _ interface{}) {
	_, _ = writer.WriteString(":")
	_, _ = writer.WriteString(strconv.Itoa(len(stmt.Vars)))
}

func (d Dialector) QuoteTo(writer clause.Writer, str string) {
	if d.NamingCaseSensitive && str != "" {
		var (
			underQuoted, selfQuoted bool
			continuousBacktick      int8
			shiftDelimiter          int8
		)

		for _, v := range []byte(str) {
			switch v {
			case '"':
				continuousBacktick++
				if continuousBacktick == 2 {
					_, _ = writer.WriteString(`""`)
					continuousBacktick = 0
				}
			case '.':
				if continuousBacktick > 0 || !selfQuoted {
					shiftDelimiter = 0
					underQuoted = false
					continuousBacktick = 0
					_ = writer.WriteByte('"')
				}
				_ = writer.WriteByte(v)
				continue
			default:
				if shiftDelimiter-continuousBacktick <= 0 && !underQuoted {
					_ = writer.WriteByte('"')
					underQuoted = true
					if selfQuoted = continuousBacktick > 0; selfQuoted {
						continuousBacktick -= 1
					}
				}

				for ; continuousBacktick > 0; continuousBacktick -= 1 {
					_, _ = writer.WriteString(`""`)
				}

				_ = writer.WriteByte(v)
			}
			shiftDelimiter++
		}

		if continuousBacktick > 0 && !selfQuoted {
			_, _ = writer.WriteString(`""`)
		}
		_ = writer.WriteByte('"')
	} else {
		_, _ = writer.WriteString(str)
	}
}

var numericPlaceholder = regexp.MustCompile(`:(\d+)`)

func (d Dialector) Explain(sql string, vars ...interface{}) string {
	for idx, v := range vars {
		if b, ok := v.(bool); ok {
			if b {
				vars[idx] = 1
			} else {
				vars[idx] = 0
			}
		}
	}
	return logger.ExplainSQL(sql, numericPlaceholder, `'`, vars...)
}

func (d Dialector) DataTypeOf(field *schema.Field) string {
	delete(field.TagSettings, "RESTRICT")

	var sqlType string

	switch field.DataType {
	case schema.Bool:
		sqlType = "NUMBER(1)"
	case schema.Int, schema.Uint, schema.Float:
		sqlType = "INTEGER"

		switch {
		case field.DataType == schema.Float:
			sqlType = "FLOAT"
		case field.Size > 0 && field.Size <= 8:
			sqlType = "SMALLINT"
		}

		if val, ok := field.TagSettings["AUTOINCREMENT"]; ok && utils.CheckTruth(val) {
			sqlType += " GENERATED BY DEFAULT AS IDENTITY"
		}
	case schema.String, "VARCHAR2":
		size := field.Size
		defaultSize := d.DefaultStringSize

		if size == 0 {
			if defaultSize > 0 {
				size = int(defaultSize)
			} else {
				hasIndex := field.TagSettings["INDEX"] != "" || field.TagSettings["UNIQUE"] != ""
				// TEXT, GEOMETRY or JSON column can't have a default value
				if field.PrimaryKey || field.HasDefaultValue || hasIndex {
					size = 191 // utf8mb4
				}
			}
		}

		if size > 0 && size <= 2000 {
			sqlType = fmt.Sprintf("VARCHAR2(%d)", size)
		} else {
			sqlType = "CLOB"
		}

	case schema.Time:
		sqlType = "TIMESTAMP WITH TIME ZONE"

	case schema.Bytes:
		sqlType = "BLOB"
	default:
		sqlType = string(field.DataType)

		if strings.EqualFold(sqlType, "text") {
			sqlType = "CLOB"
		}

		if sqlType == "" {
			panic(fmt.Sprintf("invalid sql type %s (%s) for oracle", field.FieldType.Name(), field.FieldType.String()))
		}

	}

	return sqlType
}

func (d Dialector) SavePoint(tx *gorm.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return tx.Error
}

func (d Dialector) RollbackTo(tx *gorm.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return tx.Error
}
