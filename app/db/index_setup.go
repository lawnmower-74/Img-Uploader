package db

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// =====================================================================
// 設定するインデックスを定義（インデックスを追加したい場合は、以下に追加）
// =====================================================================
var CollectionIndexModels = map[string][]mongo.IndexModel{
	"images": {
		// filename: ユニークチェック用インデックス
		{
			Keys: 		bson.D{{Key: "filename", Value: 1}}, 							// filenameフィールドを昇順(1)でインデックス
			Options: 	options.Index().SetName("filename_unique").SetUnique(true),		// ユニーク制約
		},
	},
}