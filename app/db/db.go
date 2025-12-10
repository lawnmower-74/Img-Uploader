package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoClientInstance *mongo.Client

// ============================================
// DBへの接続（成功した場合、DBインスタンスを返す）
// ============================================
func ConnectDB() *mongo.Client {
	// ※シングルトンパターン（生成するのは初回のみ。以降はその値を流用）
	if mongoClientInstance != nil {
		return mongoClientInstance
	}

	// -------------------
	// DB接続情報のセット
	// -------------------
	user := os.Getenv("MONGO_USER")
	pass := os.Getenv("MONGO_PASSWORD")
	host := os.Getenv("MONGO_HOST")
	port := os.Getenv("MONGO_PORT")
	dbName := os.Getenv("MONGO_DB")

	if user == "" || pass == "" || host == "" || port == "" || dbName == "" {
		// 終了
		log.Fatalln("FATAL: DB接続に必要な環境変数が設定されていません")
	}
	
	mongoURI := fmt.Sprintf(
		"mongodb://%s:%s@%s:%s/%s?authSource=admin",
		user, 
		pass, 
		host, 
		port, 
		dbName,
	)
	clientOptions := options.Client().ApplyURI(mongoURI)

	// -------------------------------------------------------
	// DB接続を試行（※ログで状況確認できるのでアプリ側で実行）
	// -------------------------------------------------------
	var client *mongo.Client
	var err error
	maxRetries := 5 				// 最大試行回数
	retryDelay := 5 * time.Second 	// 待機時間

	log.Println("DB接続試行中...")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 接続の試行箇所
	for i := 0; i < maxRetries; i++ {
		// クライアントを作成（接続成功時にはこれを返す）
		client, err = mongo.Connect(ctx, clientOptions)
		if err != nil {
			// 終了
			log.Fatalf("FATAL: DBクライアントの作成に失敗しました: %v", err)
		}

		// 接続の疎通確認 (Ping) を実行
		if err = client.Ping(ctx, nil); err == nil {
			log.Println("DBへの接続を確立しました")

			// ========================================
			// コレクションに付与するインデックスを作成
			// ========================================
			ensureIndexes(client, dbName)

			// 返り値はココ（※グローバル変数に代入）
			mongoClientInstance = client
			return mongoClientInstance
		}

		// Pingが失敗した場合のリトライ処理
		log.Printf("WARN: DB接続失敗 (試行 %d/%d): Ping失敗: %v. %v後にリトライします...", i+1, maxRetries, err, retryDelay)

		if err := client.Disconnect(context.Background()); err != nil {
            log.Printf("ERROR: Ping失敗後のDBクライアント切断に失敗しました: %v", err)
        }

		time.Sleep(retryDelay)

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	}

	// 終了
	log.Fatalf("FATAL: DBへの接続に最大試行回数 (%d回) 失敗しました: %v", maxRetries, err)
}

// ======================
// DB接続のクローズ
// ======================
func CloseDB(client *mongo.Client) {
	if client == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Disconnect(ctx); err != nil {
		log.Printf("ERROR: DB接続のクローズに失敗しました: %v", err)
		return
	}
	
	log.Println("DB接続を正常に切断しました")
	mongoClientInstance = nil
}