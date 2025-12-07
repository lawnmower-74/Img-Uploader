package migrate

import (
	"log"

	"gorm.io/gorm"
	"uploader/models"
)

// ================================================
// AllModels に定義された全モデルのマイグレーション
// ================================================
func RunAllMigrations(db *gorm.DB) error {
	log.Println("マイグレーションを開始...")
	
	err := db.AutoMigrate(AllModels...)
	if err != nil {
		log.Printf("ERROR: マイグレーション中にエラーが発生しました: %v", err)
		return err
	}
	
	log.Println("マイグレーション完了")
	return nil
}

// ================================================
// テーブルを追加する場合、ここにモデルを追加する
// ================================================
var AllModels = []interface{}{
	&models.Image{},
}