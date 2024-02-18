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
	} else if err == nil {
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
	} else if err == nil {
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
