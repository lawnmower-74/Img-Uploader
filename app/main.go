package main

import (
	"log"
	
	"uploader/db"
	"uploader/migrate"
)

// DB起動・マイグレートの実行
func main() {
	gormDB := db.ConnectDB()
	defer db.CloseDB(gormDB)
	
	err := migrate.RunAllMigrations(gormDB)
	if err != nil {
		// 終了
		log.Fatalln("FATAL: マイグレーションに失敗したので処理を終了します")
	}

	log.Println("アプリケーションの初期設定（DB接続・マイグレーション）が正常に完了しました")
}