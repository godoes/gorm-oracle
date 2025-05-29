package oracle

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMergeCreate(t *testing.T) {
	db, err := dbNamingCase, dbErrors[0]
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Log("db is nil!")
		return
	}

	model := TestTableUser{}
	migrator := db.Set("gorm:table_comments", "用户信息表").Migrator()
	if migrator.HasTable(model) {
		if err = migrator.DropTable(model); err != nil {
			t.Fatalf("DropTable() error = %v", err)
		}
	}
	if err = migrator.AutoMigrate(model); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	} else {
		t.Log("AutoMigrate() success!")
	}

	data := []TestTableUser{
		{
			UID:         "U1",
			Name:        "Lisa",
			Account:     "lisa",
			Password:    "H6aLDNr",
			PhoneNumber: "+8616666666666",
			Sex:         "0",
			UserType:    1,
			Enabled:     true,
		},
		{
			UID:         "U1",
			Name:        "Lisa",
			Account:     "lisa",
			Password:    "H6aLDNr",
			PhoneNumber: "+8616666666666",
			Sex:         "0",
			UserType:    1,
			Enabled:     true,
		},
		{
			UID:         "U2",
			Name:        "Daniela",
			Account:     "daniela",
			Password:    "Si7l1sRIC79",
			PhoneNumber: "+8619999999999",
			Sex:         "1",
			UserType:    1,
			Enabled:     true,
		},
	}
	t.Run("MergeCreate", func(t *testing.T) {
		tx := db.Create(&data)
		if err = tx.Error; err != nil {
			t.Fatal(err)
		}
		dataJsonBytes, _ := json.MarshalIndent(data, "", "  ")
		t.Logf("result: %s", dataJsonBytes)
	})
}

type TestTableUserUnique struct {
	ID          uint64     `gorm:"column:id;size:64;not null;autoIncrement:true;autoIncrementIncrement:1;primaryKey;comment:自增 ID" json:"id"`
	UID         string     `gorm:"column:uid;type:varchar(50);comment:用户身份标识;unique" json:"uid"`
	Name        string     `gorm:"column:name;size:50;comment:用户姓名" json:"name"`
	Account     string     `gorm:"column:account;type:varchar(50);comment:登录账号" json:"account"`
	Password    string     `gorm:"column:password;type:varchar(512);comment:登录密码（密文）" json:"password"`
	Email       string     `gorm:"column:email;type:varchar(128);comment:邮箱地址" json:"email"`
	PhoneNumber string     `gorm:"column:phone_number;type:varchar(15);comment:E.164" json:"phoneNumber"`
	Sex         string     `gorm:"column:sex;type:char(1);comment:性别" json:"sex"`
	Birthday    *time.Time `gorm:"column:birthday;->:false;<-:create;comment:生日" json:"birthday,omitempty"`
	UserType    int        `gorm:"column:user_type;size:8;comment:用户类型" json:"userType"`
	Enabled     bool       `gorm:"column:enabled;comment:是否可用" json:"enabled"`
	Remark      string     `gorm:"column:remark;size:1024;comment:备注信息" json:"remark"`
}

func (TestTableUserUnique) TableName() string {
	return "test_user_unique"
}

func TestMergeCreateUnique(t *testing.T) {
	db, err := dbNamingCase, dbErrors[0]
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Log("db is nil!")
		return
	}

	model := TestTableUserUnique{}
	migrator := db.Set("gorm:table_comments", "用户信息表").Migrator()
	if migrator.HasTable(model) {
		if err = migrator.DropTable(model); err != nil {
			t.Fatalf("DropTable() error = %v", err)
		}
	}
	if err = migrator.AutoMigrate(model); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	} else {
		t.Log("AutoMigrate() success!")
	}

	data := []TestTableUserUnique{
		{
			UID:         "U1",
			Name:        "Lisa",
			Account:     "lisa",
			Password:    "H6aLDNr",
			PhoneNumber: "+8616666666666",
			Sex:         "0",
			UserType:    1,
			Enabled:     true,
		},
		{
			UID:         "U2",
			Name:        "Daniela",
			Account:     "daniela",
			Password:    "Si7l1sRIC79",
			PhoneNumber: "+8619999999999",
			Sex:         "1",
			UserType:    1,
			Enabled:     true,
		},
		{
			UID:         "U2",
			Name:        "Daniela",
			Account:     "daniela",
			Password:    "Si7l1sRIC79",
			PhoneNumber: "+8619999999999",
			Sex:         "1",
			UserType:    1,
			Enabled:     true,
		},
	}
	t.Run("MergeCreateUnique", func(t *testing.T) {
		tx := db.Create(&data)
		if err = tx.Error; err != nil {
			if strings.Contains(err.Error(), "ORA-00001") {
				t.Log(err) // ORA-00001: 违反唯一约束条件
				var gotData []TestTableUserUnique
				tx = db.Where(map[string]interface{}{"uid": []string{"U1", "U2"}}).Find(&gotData)
				if err = tx.Error; err != nil {
					t.Fatal(err)
				} else {
					if len(gotData) > 0 {
						t.Error("Unique constraint violation, but some data was inserted!")
					} else {
						t.Log("Unique constraint violation, rolled back!")
					}
				}
			} else {
				t.Fatal(err)
			}
			return
		}
		dataJsonBytes, _ := json.MarshalIndent(data, "", "  ")
		t.Logf("result: %s", dataJsonBytes)
	})
}

type testModelOra03146TTC struct {
	Id          int64     `gorm:"primaryKey;autoIncrement:false;column:SL_ID;type:uint;size:20;default:0;comment:id" json:"SL_ID"`
	ApiName     string    `gorm:"column:SL_API_NAME;type:VARCHAR2;size:100;default:null;comment:接口名称" json:"SL_API_NAME"`
	RawReceive  string    `gorm:"column:SL_RAW_RECEIVE_JSON;type:VARCHAR2;size:4000;default:null;comment:原始请求参数" json:"SL_RAW_RECEIVE_JSON"`
	RawSend     string    `gorm:"column:SL_RAW_SEND_JSON;type:VARCHAR2;size:4000;default:null;comment:原始响应参数" json:"SL_RAW_SEND_JSON"`
	DealReceive string    `gorm:"column:SL_DEAL_RECEIVE_JSON;type:VARCHAR2;size:4000;default:null;comment:处理请求参数" json:"SL_DEAL_RECEIVE_JSON"`
	DealSend    string    `gorm:"column:SL_DEAL_SEND_JSON;type:VARCHAR2;size:4000;default:null;comment:处理响应参数" json:"SL_DEAL_SEND_JSON"`
	Code        string    `gorm:"column:SL_CODE;type:VARCHAR2;size:16;default:null;comment:http状态" json:"SL_CODE"`
	CreatedTime time.Time `gorm:"column:SL_CREATED_TIME;type:date;default:null;comment:创建时间" json:"SL_CREATED_TIME"`
}

func TestOra03146TTC(t *testing.T) {
	db, err := dbNamingCase, dbErrors[0]
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Log("db is nil!")
		return
	}

	model := testModelOra03146TTC{}
	migrator := db.Set("gorm:table_comments", "TTC 字段的缓冲区长度无效问题测试表").Migrator()
	if migrator.HasTable(model) {
		if err = migrator.DropTable(model); err != nil {
			t.Fatalf("DropTable() error = %v", err)
		}
	}
	if err = migrator.AutoMigrate(model); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	} else {
		t.Log("AutoMigrate() success!")
	}

	// INSERT INTO "T100_SCPTOAPI_LOG" ("SL_ID","SL_API_NAME","SL_RAW_RECEIVE_JSON","SL_RAW_SEND_JSON","SL_DEAL_RECEIVE_JSON","SL_DEAL_SEND_JSON","SL_CODE","SL_CREATED_TIME")
	// VALUES (9578529926701056,'/v1/t100/packingNum','11111','11111','11111','11111','111','2024-08-27 18:21:39.495')
	data := testModelOra03146TTC{
		Id:          9578529926701056,
		ApiName:     "/v1/t100/packingNum",
		RawReceive:  "11111",
		RawSend:     "11111",
		DealReceive: "11111",
		DealSend:    "11111",
		Code:        "111",
		CreatedTime: time.Now(),
	}
	result := db.Create(&data)
	if err = result.Error; err != nil {
		t.Fatalf("执行失败：%v", err)
	}
	t.Log("执行成功，影响行数：", result.RowsAffected)
}

func TestCreateInBatches(t *testing.T) {
	db, err := dbNamingCase, dbErrors[0]
	if err != nil {
		t.Fatal(err)
	}
	if db == nil {
		t.Log("db is nil!")
		return
	}

	model := TestTableUser{}
	migrator := db.Set("gorm:table_comments", "用户信息表").Migrator()
	if migrator.HasTable(model) {
		if err = migrator.DropTable(model); err != nil {
			t.Fatalf("DropTable() error = %v", err)
		}
	}
	if err = migrator.AutoMigrate(model); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	} else {
		t.Log("AutoMigrate() success!")
	}

	data := []TestTableUser{
		{UID: "U1", Name: "Lisa", Account: "lisa", Password: "H6aLDNr", PhoneNumber: "+8616666666666", Sex: "0", UserType: 1, Enabled: true},
		{UID: "U2", Name: "Daniela", Account: "daniela", Password: "Si7l1sRIC79", PhoneNumber: "+8619999999999", Sex: "1", UserType: 1, Enabled: true},
		{UID: "U3", Name: "Tom", Account: "tom", Password: "********", PhoneNumber: "+8618888888888", Sex: "1", UserType: 1, Enabled: true},
		{UID: "U4", Name: "James", Account: "james", Password: "********", PhoneNumber: "+8617777777777", Sex: "1", UserType: 2, Enabled: true},
		{UID: "U5", Name: "John", Account: "john", Password: "********", PhoneNumber: "+8615555555555", Sex: "1", UserType: 1, Enabled: true},
	}
	t.Run("CreateInBatches", func(t *testing.T) {
		tx := db.CreateInBatches(&data, 2)
		if err = tx.Error; err != nil {
			t.Fatal(err)
		}
		dataJsonBytes, _ := json.MarshalIndent(data, "", "  ")
		t.Logf("result: %s", dataJsonBytes)
	})
}
