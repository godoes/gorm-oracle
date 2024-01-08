package oracle

import (
	"database/sql"
	"log"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	dbNamingCase *gorm.DB
	dbIgnoreCase *gorm.DB

	dbErrors = make([]error, 2)
)

func init() {
	if wait := os.Getenv("GORM_ORA_WAIT_MIN"); wait != "" {
		if min, e := strconv.Atoi(wait); e == nil {
			log.Println("wait for oracle database initialization to complete...")
			time.Sleep(time.Duration(min) * time.Minute)
		}
	}
	var err error
	if dbNamingCase, err = openTestConnection(true, true); err != nil {
		dbErrors[0] = err
	}
	if dbIgnoreCase, err = openTestConnection(true, false); err != nil {
		dbErrors[1] = err
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

func TestAddSessionParams(t *testing.T) {
	db, err := dbIgnoreCase, dbErrors[1]
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Log("db is nil!")
		return
	}
	var sqlDB *sql.DB
	if sqlDB, err = db.DB(); err != nil {
		t.Fatal(err)
	}
	type args struct {
		params map[string]string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "TimeParams", args: args{params: map[string]string{
			"TIME_ZONE":               "+08:00",                       // alter session set TIME_ZONE = '+08:00';
			"NLS_DATE_FORMAT":         "YYYY-MM-DD",                   // alter session set NLS_DATE_FORMAT = 'YYYY-MM-DD';
			"NLS_TIME_FORMAT":         "HH24:MI:SSXFF",                // alter session set NLS_TIME_FORMAT = 'HH24:MI:SS.FF3';
			"NLS_TIMESTAMP_FORMAT":    "YYYY-MM-DD HH24:MI:SSXFF",     // alter session set NLS_TIMESTAMP_FORMAT = 'YYYY-MM-DD HH24:MI:SS.FF3';
			"NLS_TIME_TZ_FORMAT":      "HH24:MI:SS.FF TZR",            // alter session set NLS_TIME_TZ_FORMAT = 'HH24:MI:SS.FF3 TZR';
			"NLS_TIMESTAMP_TZ_FORMAT": "YYYY-MM-DD HH24:MI:SSXFF TZR", // alter session set NLS_TIMESTAMP_TZ_FORMAT = 'YYYY-MM-DD HH24:MI:SS.FF3 TZR';
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//queryTime := `SELECT SYSDATE FROM DUAL`
			queryTime := `SELECT CAST(SYSDATE AS VARCHAR(30)) AS D FROM DUAL`
			var timeStr string
			if err = db.Raw(queryTime).Row().Scan(&timeStr); err != nil {
				t.Fatal(err)
			}
			t.Logf("SYSDATE 1: %s", timeStr)

			var keys []string
			if keys, err = AddSessionParams(sqlDB, tt.args.params); err != nil {
				t.Fatalf("AddSessionParams() error = %v", err)
			}
			if err = db.Raw(queryTime).Row().Scan(&timeStr); err != nil {
				t.Fatal(err)
			}
			defer DelSessionParams(sqlDB, keys)
			t.Logf("SYSDATE 2: %s", timeStr)
			t.Logf("keys: %#v", keys)
		})
	}
}

func TestGetStringExpr(t *testing.T) {
	db, err := dbNamingCase, dbErrors[0]
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Log("db is nil!")
		return
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
