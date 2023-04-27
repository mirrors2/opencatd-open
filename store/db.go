package store

import (
	"log"
	"os"

	// "gorm.io/driver/sqlite"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

var usage *gorm.DB

func init() {
	if _, err := os.Stat("db"); os.IsNotExist(err) {
		errDir := os.MkdirAll("db", 0755)
		if errDir != nil {
			log.Fatalln("Error creating directory:", err)
		}
	}
	var err error
	db, err = gorm.Open(sqlite.Open("./db/cat.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// 自动迁移 User 结构体
	err = db.AutoMigrate(&User{}, &Key{})
	if err != nil {
		panic(err)
	}
	LoadKeysCache()
	LoadAuthCache()

	usage, err = gorm.Open(sqlite.Open("./db/usage.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	err = usage.AutoMigrate(&DailyUsage{}, &Usage{})
	if err != nil {
		panic(err)
	}
}
