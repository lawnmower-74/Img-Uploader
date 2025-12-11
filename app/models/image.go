package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive" 
)

type Image struct {
	ID            	primitive.ObjectID 	`bson:"_id,omitempty" json:"id,omitempty"` 
	FileName      	string             	`bson:"filename" json:"fileName"`
	Size          	int64              	`bson:"size" json:"size"`
	ContentType   	string             	`bson:"content_type" json:"contentType"`	// 画像のMIMEタイプ (データ形式)
	FilePath      	string             	`bson:"file_path" json:"filePath"`      	// 画像データ本体（バイナリデータ）が存在する場所を示すリンクを格納
	CreatedAt 		time.Time 			`bson:"created_at" json:"createdAt"`
	UpdatedAt 		time.Time 			`bson:"updated_at" json:"updatedAt"`
}