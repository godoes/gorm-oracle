package oracle

import (
	"encoding/json"
	"testing"
)

func TestMergeCreate(t *testing.T) {
	db, err := openTestConnection(true, true)
	if err != nil {
		t.Fatal(err)
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
