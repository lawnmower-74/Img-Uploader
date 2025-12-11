package uploader

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"uploader/config"
	"uploader/models"
)

func Run(imagesCollection *mongo.Collection) {
	// ----------------------------------
	// アップロード対象の画像ファイルを検索
	// ----------------------------------
	imageDir := config.AppConfig.ImageDir
	if imageDir == "" {
		// 終了
		log.Fatal("FATAL: 環境変数 IMAGE_DIR が設定されていません")
	}

	log.Printf("アップロード画像検索中...: %s\n", imageDir)

	dirEntries, err := os.ReadDir(imageDir)
	if err != nil {
		// 終了
		log.Fatalf("FATAL: 画像ディレクトリの読み込みに失敗しました。パスをご確認ください: %v\n", err)
	}

	var filesToUpload []os.DirEntry
	for _, file := range dirEntries {
		// フィルタリング: フォルダ、隠しファイル、.gitkeepを除外（スキップ）
		if file.IsDir() || strings.HasPrefix(file.Name(), ".") || file.Name() == ".gitkeep" {
			continue
		}
		// それ以外はアップロード対象としてリストに追加
		filesToUpload = append(filesToUpload, file)
	}

	totalFiles := len(filesToUpload)

	if totalFiles == 0 {
		log.Println("アップする画像ファイルが見つかりませんでした")
		return
	}

	// ===========================
	// 並行処理の制御（Goroutine）
	// ===========================
	log.Printf("%d 件の画像を検出。アップロードを開始します\n", totalFiles)

	var wg sync.WaitGroup
	sem := make(chan struct{}, config.AppConfig.MaxWorkers)	// 同時に起動する Goroutine の最大数を制御するセマフォ
	var completedCount uint64								// ※アップロード件数表示のため

	startTime := time.Now() // ※処理時間計測用

	// 各画像に対してGoroutineを起動
	for _, file := range filesToUpload {
		fileName := file.Name()
		filePath := filepath.Join(imageDir, fileName)

		// １画像１タスク
		wg.Add(1)

		// セマフォにトークンを送信
		// 上限超過するとタスクをブロックしメモリに一時待機させる。（バッファが空き次第自動で再開）
		sem <- struct{}{}

		// -----------------
		// Goroutine 起動
		// -----------------
		go func(path string, name string) { 
			defer wg.Done()				// 処理の完了を通知
			defer func() { <-sem }()	// 処理が完了したらセマフォを開放

			// 画像サイズ取得
			fileInfo, err := os.Stat(path)
			if err != nil {
				log.Printf("ERROR: %s の情報取得に失敗: %v\n", name, err)
				return
			}
			fileSize := fileInfo.Size()

			// --------------------------------------------
			// MIMEタイプを識別（例: image/jpeg、image/png）	
			// --------------------------------------------
			contentType, err := getContentType(path)
			if err != nil {
				log.Printf("ERROR: %s のMIMEタイプ判定に失敗: %v\n", name, err)
				return
			}
			// 画像ファイル以外（不明なバイナリ）はスキップ
			if !strings.HasPrefix(contentType, "image/") {
				log.Printf("WARN: %s は画像ファイルではありません (%s)。スキップします。\n", name, contentType)
				return
			}

			// -----------------------
			// DBへのアップロード
			// -----------------------
			err = saveImageData(imagesCollection, name, fileSize, contentType, path)
			if err != nil {
				// DB登録エラー（※ユニーク制約違反含む）
				log.Printf("ERROR: %s のDB登録に失敗: %v\n", name, err)
				return
			}
			
			newCount := atomic.AddUint64(&completedCount, 1)

			log.Printf("(%d/%d) %s の登録に成功 \n", newCount, totalFiles, name)

		}(filePath, fileName)
	}

	// すべてのGoroutineが完了するまで以降の処理を行わない
	wg.Wait()

	duration := time.Since(startTime)

	log.Println("\n----- 全てのアップロード処理が完了しました -----")
	log.Printf("総検出ファイル数: %d\n", totalFiles)
	log.Printf("成功登録数: %d\n", completedCount)
	log.Printf("かかった時間: %s\n", duration)
	log.Printf("平均処理速度: %.2f files/sec\n", float64(completedCount)/duration.Seconds())
}

// ==============================
//　画像をDBに登録する（一件ずつ）
// ==============================
func saveImageData(collection *mongo.Collection, fileName string, size int64, contentType string, filePath string) error {
	image := models.Image{
		FileName:    fileName,
		Size:        size,
		ContentType: contentType,
		FilePath:    filePath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// -------------------------------------
	// ※ユニーク制限のかかったフィールド
	// -------------------------------------
	filter := bson.M{
		"filename": fileName,
	}

	update := bson.M{
		"$set": bson.M{
			"size": 		image.Size,
			"contenttype": 	image.ContentType,
			"filepath": 	image.FilePath,
			"updatedat": 	time.Now(),
		},
		"$setOnInsert": bson.M{
			// 新規挿入時のみCreatedAtを設定
			"createdat": 	time.Now(),
		},
	}

	// ※Upsert (ドキュメントが存在しなければ新規作成) を有効にする
	opts := options.Update().SetUpsert(true)

	// 更新／作成
	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}
	
	return nil
}

// ===========================================================
// 画像のヘッダー情報をもとにデータ形式を識別（拡張子偽造を防ぐ）
// ===========================================================
func getContentType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	// 画像ファイルの先頭512バイトを読み込む
	_, err = io.ReadFull(file, buffer)
	// ファイルが512バイト未満でも処理を続ける
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("ファイル読み込みエラー: %w", err)
	}

	// MIMEタイプを識別
	contentType := http.DetectContentType(buffer)
	
	// DetectContentTypeは text/plain; charset=utf-8 のように返すため、セミコロン以降を除去
	contentType = strings.Split(contentType, ";")[0]

	return contentType, nil
}