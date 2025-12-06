package db

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// DBインスタンスを返す
func ConnectDB() *gorm.DB {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")

	if user == "" || pass == "" || host == "" || port == "" || dbname == "" {
		// 終了
		log.Fatalln("FATAL: DB接続に必要な環境変数が設定されていません")
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, 
		pass, 
		host, 
		port, 
		dbname,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		// 終了
		log.Fatalf("FATAL: DB接続失敗 (DSN: %s の試行中にエラー): %v\n", dsn, err)
	}

	fmt.Println("INFO: DB接続成功")
	return db
}

// DB接続のクローズ
func CloseDB(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("WARN: DBオブジェクト取得失敗: %v\n", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("ERROR: DB接続切断失敗: %v\n", err)
	}
	fmt.Println("INFO: DB接続を切断しました")
}