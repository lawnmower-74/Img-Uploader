package main

import (
	"log"
	
	"uploader/db"
	"uploader/migrate"
)

func main() {
	// DB接続を確立
	gormDB := db.ConnectDB()
	// 実行完了時に必ずDB接続を切断
	defer db.CloseDB(gormDB)
	// テーブル作成
	err := migrate.RunAllMigrations(gormDB)
	if err != nil {
		// 終了
		log.Fatalln("FATAL: マイグレーションに失敗したので処理を終了します")
	}

	log.Println("SUCCESS: アプリケーションの初期設定（DB接続・マイグレーション）が正常に完了しました")
}