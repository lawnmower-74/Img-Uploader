package main

import (
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

	"gorm.io/gorm"
	"uploader/db"
	"uploader/models"
)

// 最大並行処理数を設定（処理を実行可能なコアの上限数）
const MAX_CONCURRENCY = 4

func main() {
	// -----------
	// DB接続設定
	// -----------
	gormDB := db.ConnectDB()
	defer db.CloseDB(gormDB) // ※CLI処理が終了したらDB接続をクローズ

	log.Println("SUCCESS: DBとの接続が確立されました")

	// ------------------
	// 画像ファイルの検索
	// ------------------
	imageDir := os.Getenv("IMAGE_DIR")
	if imageDir == "" {
		// 終了
		log.Fatal("FATAL: 環境変数 IMAGE_DIR が設定されていません")
	}

	fmt.Printf("INFO: アップロード画像検索中: %s\n", imageDir)

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
		log.Println("INFO: アップする画像ファイルが見つかりませんでした")
		return
	}

	log.Printf("%d 件の画像を検出。アップロードを開始します \n", totalFiles)

	// ===========================
	// 並行処理の制御（Goroutine）
	// ===========================
	var wg sync.WaitGroup							// 全ての Goroutine の完了を待つ監督
	sem := make(chan struct{}, MAX_CONCURRENCY)		// 同時に起動する Goroutine の最大数を制御するセマフォ
	var completedCount uint64

	startTime := time.Now()

	// 各画像に対してGoroutineを起動
	for _, file := range filesToUpload {
		fileName := file.Name()
		filePath := filepath.Join(imageDir, fileName)

		// １画像１タスク
		wg.Add(1)

		// セマフォにトークンを送信（最大数を超えるとここでブロック）
		// (※ブロックされたタスクは一時的にメモリで待機。バッファが空き次第自動で再開される)
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

			// ------------------------------------------------------------
			// MIMEタイプからタイプ／拡張子を識別（例: image/png, text/html）	
			// ------------------------------------------------------------
			contentType, err := getContentType(path)
			if err != nil {
				fmt.Printf("ERROR: %s のMIMEタイプ判定に失敗: %v\n", name, err)
				return
			}
			// 画像ファイル以外（不明なバイナリ）はスキップ
			if !strings.HasPrefix(contentType, "image/") {
				fmt.Printf("WARN: %s は画像ファイルではありません (%s)。スキップします。\n", name, contentType)
				return
			}

			// -----------------------
			// DBへのメタデータ登録
			// -----------------------
			err = recordImageData(gormDB, name, fileSize, contentType, path)
			if err != nil {
				// DB登録エラー（※ユニーク制約違反含む）
				log.Printf("ERROR: %s のDB登録に失敗: %v\n", name, err)
				return
			}
			
			newCount := atomic.AddUint64(&completedCount, 1)

			log.Printf("SUCCESS: (%d/%d) %s の登録に成功 \n", newCount, totalFiles, name)

		}(filePath, fileName)
	}

	// すべてのGoroutineが完了するのを待つ
	wg.Wait()

	duration := time.Since(startTime)

	log.Println("INFO: 全てのアップロード処理が完了しました。")
	log.Printf("総検出ファイル数: %d\n", totalFiles)
	log.Printf("成功登録数: %d\n", completedCount)
	log.Printf("かかった時間: %s\n", duration)
	log.Printf("平均処理速度: %.2f files/sec\n", float64(completedCount)/duration.Seconds())
}

// =========================
// 1つの画像をDBに登録する
// =========================
func recordImageData(db *gorm.DB, fileName string, size int64, contentType string, filePath string) error {
	image := models.Image{
		FileName:    fileName,
		Size:        size,
		ContentType: contentType,
		FilePath:    filePath,
	}

	result := db.Create(&image)
	if result.Error != nil {
		// ユニーク制約違反のエラー処理
		if strings.Contains(result.Error.Error(), "Duplicate entry") {
			return fmt.Errorf("指定されたファイル名 '%s' は既に存在します", fileName)
		}
		return result.Error
	}
	return nil
}

// =====================================
// 簡易的な拡張子からMIMEタイプを返す関数
// =====================================
func getContentType(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// ファイルヘッダーの最初の512バイトを読み込む
	// net/http.DetectContentType はこの512バイトのマジックナンバーに基づいて判定する
	buffer := make([]byte, 512)
	// io.ReadFullを使うことで、ファイルが512バイト未満でもエラーなく読み込める（短い場合はファイル終端まで読み込む）
	_, err = io.ReadFull(file, buffer)
	// EOFエラーは気にせず続行（ファイルが512バイト未満でも処理を続けたい）
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("ファイル読み込みエラー: %w", err)
	}

	// MIMEタイプを判定
	contentType := http.DetectContentType(buffer)
	
	// 例外: DetectContentTypeはテキストファイルを text/plain; charset=utf-8 のように返すため、
	// DBに記録するためにセミコロン以降を除去する
	contentType = strings.Split(contentType, ";")[0]

	return contentType, nil
}