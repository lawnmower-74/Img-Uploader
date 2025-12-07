package models

import (
	"time"

	"gorm.io/gorm"
)

type Image struct {
	ID				uint			`gorm:"primaryKey"`
	FileName		string			`gorm:"uniqueIndex;type:varchar(255);not null"`	// 重複禁止
	Size			int64			`gorm:"not null"`
	ContentType		string			`gorm:"type:varchar(50);not null"`  			// 画像のMIMEタイプ (データ形式)
	FilePath		string			`gorm:"type:varchar(512)"`          			// 画像データ本体（バイナリデータ）が存在する場所を示すリンク
	CreatedAt		time.Time
	UpdatedAt		time.Time
	DeletedAt		gorm.DeletedAt	`gorm:"index"`
}