package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// コレクションに必要なインデックスが設定されていることを確認（※アプリケーション起動時に一度だけ実行）
func ensureIndexes(client *mongo.Client, dbName string) {

	// 対象コレクション
	collection := client.Database(dbName).Collection("images")
	
	// ===========================================
	// インデックスを追加したい場合は、以下に追加
	// ===========================================
	indexModels := []mongo.IndexModel{
		// filename: ユニークチェック用インデックス
		{
			Keys:    bson.D{{Key: "filename", Value: 1}}, 							// filenameフィールドを昇順(1)でインデックス
			Options: options.Index().SetUnique(true).SetName("filename_unique"),	// ユニーク制約
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// インデックスを一括作成（既に存在する場合はスキップ） 
	names, err := collection.Indexes().CreateMany(ctx, indexModels)
	if err != nil {
		// 終了
		log.Fatalf("FATAL: DBのインデックス作成に失敗しました: %v", err)
	}
	log.Printf("DBのインデックス (%d件) の作成または確認が完了しました: %v", len(names), names)
}