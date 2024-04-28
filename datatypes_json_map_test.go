package oracle

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// JSONMap defined JSON data type, need to implement driver.Valuer, sql.Scanner interface
type JSONMap map[string]interface{}

// Value return json value, implement driver.Valuer interface
//
//goland:noinspection GoMixedReceiverTypes
func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	ba, err := m.MarshalJSON()
	return string(ba), err
}

// Scan value into Jsonb, implements sql.Scanner interface
//
//goland:noinspection GoMixedReceiverTypes
func (m *JSONMap) Scan(val interface{}) error {
	if val == nil {
		*m = make(JSONMap)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := map[string]interface{}{}
	rd := bytes.NewReader(ba)
	decoder := json.NewDecoder(rd)
	decoder.UseNumber()
	err := decoder.Decode(&t)
	*m = t
	return err
}

// MarshalJSON to output non base64 encoded []byte
//
//goland:noinspection GoMixedReceiverTypes
func (m JSONMap) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	t := (map[string]interface{})(m)
	return json.Marshal(t)
}

// UnmarshalJSON to deserialize []byte
//
//goland:noinspection GoMixedReceiverTypes
func (m *JSONMap) UnmarshalJSON(b []byte) error {
	t := map[string]interface{}{}
	err := json.Unmarshal(b, &t)
	*m = t
	return err
}

// GormDataType gorm common data type
//
//goland:noinspection GoMixedReceiverTypes
func (m JSONMap) GormDataType() string {
	return "jsonmap"
}

// GormDBDataType gorm db data type
//
//goland:noinspection GoMixedReceiverTypes
func (JSONMap) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	case "sqlserver":
		return "NVARCHAR(MAX)"
	case "oracle":
		//return "BLOB"
		// BLOB is only supported in Oracle databases version 12c r12.2.0.1.0 and above.
		// to support lower versions of Oracle databases, it is recommended to use CLOB.
		// see also:
		// https://stackoverflow.com/questions/43603905/oracle-12c-error-getting-while-create-blob-column-table-with-json-type
		return "CLOB"
	default:
		return getGormTypeFromTag(field)
	}
}

func getGormTypeFromTag(field *schema.Field) (dataType string) {
	if field != nil {
		if val, ok := field.TagSettings["TYPE"]; ok {
			dataType = strings.ToLower(val)
		}
	}
	return
}

//goland:noinspection GoMixedReceiverTypes
func (m JSONMap) GormValue(_ context.Context, _ *gorm.DB) clause.Expr {
	data, _ := m.MarshalJSON()
	return gorm.Expr("?", string(data))
}
