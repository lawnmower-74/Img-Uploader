package main

import (
	"log"
	
	"uploader/db"
	"uploader/uploader"
)

func main() {
	// DBインスタンス生成・インデックス設定
	mongoClient := db.ConnectDB()
	
	// CLI処理が終了したらDB接続をクローズ
	defer db.CloseDB(mongoClient)

	log.Println("DBの初期設定が完了しました")

	// アップロード処理
	imagesCollection := db.GetDBCollection("images")
	uploader.Run(imagesCollection)
}