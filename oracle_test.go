package oracle

import (
	"log"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestGetStringExpr(t *testing.T) {
	db, err := openTestConnection(true, true)
	if db == nil && err == nil {
		return
	} else if err != nil {
		t.Fatal(err)
	}

	type args struct {
		prepareSQL string
		value      string
		quote      bool
	}
	tests := []struct {
		name    string
		args    args
		wantSQL string
	}{
		{"1", args{`SELECT ? AS HELLO FROM DUAL`, "Hi!", true}, `SELECT 'Hi!' AS HELLO FROM DUAL`},
		{"2", args{`SELECT '?' AS HELLO FROM DUAL`, "Hi!", false}, `SELECT 'Hi!' AS HELLO FROM DUAL`},
		{"3", args{`SELECT ? AS HELLO FROM DUAL`, "What's your name?", true}, `SELECT q'[What's your name?]' AS HELLO FROM DUAL`},
		{"4", args{`SELECT '?' AS HELLO FROM DUAL`, "What's your name?", false}, `SELECT 'What''s your name?' AS HELLO FROM DUAL`},
		{"5", args{`SELECT ? AS HELLO FROM DUAL`, "What's up]'?", true}, `SELECT q'{What's up]'?}' AS HELLO FROM DUAL`},
		{"6", args{`SELECT ? AS HELLO FROM DUAL`, "What's up]'}'?", true}, `SELECT q'<What's up]'}'?>' AS HELLO FROM DUAL`},
		{"7", args{`SELECT ? AS HELLO FROM DUAL`, "What's up]'}'>'?", true}, `SELECT q'(What's up]'}'>'?)' AS HELLO FROM DUAL`},
		{"8", args{`SELECT ? AS HELLO FROM DUAL`, "What's up)'}'>'?", true}, `SELECT q'[What's up)'}'>'?]' AS HELLO FROM DUAL`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
				return tx.Raw(tt.args.prepareSQL, GetStringExpr(tt.args.value, tt.args.quote))
			})
			if !reflect.DeepEqual(gotSQL, tt.wantSQL) {
				t.Fatalf("ToSQL = %v, want %v", gotSQL, tt.wantSQL)
			}
			var results []map[string]interface{}
			if err = db.Raw(gotSQL).Find(&results).Error; err != nil {
				t.Fatalf("finds all records from raw sql got error: %v", err)
			}
			t.Log("result:", results)
		})
	}
}

func openTestConnection(ignoreCase, namingCase bool) (db *gorm.DB, err error) {
	dsn := os.Getenv("GORM_ORA_DSN")
	if dsn == "" {
		server := os.Getenv("GORM_ORA_SERVER")
		port, _ := strconv.Atoi(os.Getenv("GORM_ORA_PORT"))
		if server == "" || port < 1 {
			return
		}

		language := os.Getenv("GORM_ORA_LANG")
		if language == "" {
			language = "SIMPLIFIED CHINESE"
		}
		territory := os.Getenv("GORM_ORA_TERRITORY")
		if territory == "" {
			territory = "CHINA"
		}

		dsn = BuildUrl(server, port,
			os.Getenv("GORM_ORA_SID"),
			os.Getenv("GORM_ORA_USER"),
			os.Getenv("GORM_ORA_PASS"),
			map[string]string{
				"CONNECTION TIMEOUT": "90",
				"LANGUAGE":           language,
				"TERRITORY":          territory,
				"SSL":                "false",
			})
	}
	if wait := os.Getenv("GORM_ORA_WAIT_MIN"); wait != "" {
		if min, e := strconv.Atoi(wait); e == nil {
			log.Println("wait for oracle database initialization to complete...")
			time.Sleep(time.Duration(min) * time.Minute)
		}
	}

	logWriter := new(log.Logger)
	logWriter.SetOutput(os.Stdout)
	db, err = gorm.Open(New(Config{
		DSN:                 dsn,
		IgnoreCase:          ignoreCase,
		NamingCaseSensitive: namingCase,
	}), &gorm.Config{
		Logger: logger.New(
			logWriter,
			logger.Config{LogLevel: logger.Info},
		),
		DisableForeignKeyConstraintWhenMigrating: false,
		IgnoreRelationshipsWhenMigrating:         false,
	})
	if db != nil && err == nil {
		log.Println("open oracle database connection success!")
	}
	return
}
