package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings" 
	"sync"
	"time"
	"sync/atomic"

	_ "github.com/go-sql-driver/mysql"
)

// 画像データの構造体
type ImageRecord struct {
	FileName string
	Size     int64
	Data     []byte
}

// データベース接続情報
var (
	DB_HOST = os.Getenv("DB_HOST")
	DB_USER = os.Getenv("DB_USER")
	DB_PASS = os.Getenv("DB_PASS")
	DB_NAME = os.Getenv("DB_NAME")
	DB_PORT = os.Getenv("DB_PORT")
)

// 最大並行処理数を設定（ここでは固定値として4）
const MAX_CONCURRENCY = 4

func main() {
	// -----------
	// DB接続設定
	// -----------
	if DB_HOST == "" {
		fmt.Println("FATAL: DB_HOSTがセットされていません")
		return
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", DB_USER, DB_PASS, DB_HOST, DB_PORT, DB_NAME)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("FATAL: データベース接続の初期設定に失敗しました: %v\n", err)
		return
	}
	defer db.Close()

	// 対策: DBへの接続が確立できるまでリトライ
	for i := 0; i < 30; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		fmt.Printf("データベース接続を待機中... 1秒後にリトライします。（試行 %d/30, エラー: %v）\n", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		fmt.Printf("FATAL: データベースへの接続試行が規定回数失敗しました。DBの状態と接続設定を確認してください: %v\n", err)
		return
	}
	fmt.Println("SUCCESS: データベースとの接続が確立されました。")

	// -------------
	// テーブル作成 
	// -------------
	createTable(db)

	// -----------------------------------
	// 画像ファイルの検索と並行アップロード
	// -----------------------------------
	imageDir := os.Getenv("IMAGE_DIR")
	if imageDir == "" {
		imageDir = "./images/uploads" // <- 必要あればここを編集
	}
	fmt.Printf("アップロード画像検索中: %s\n", imageDir)

	dirEntries, err := os.ReadDir(imageDir)
	if err != nil {
		fmt.Printf("FATAL: 画像ディレクトリの読み込みに失敗しました。パスなどをご確認ください: %v\n", err)
		return
	}

	var filesToUpload []os.DirEntry
	for _, file := range dirEntries {
		// フィルタリング: フォルダ、隠しファイル、.gitkeepを除外（スキップ）
		if file.IsDir() || strings.HasPrefix(file.Name(), ".") || file.Name() == ".gitkeep" {
			continue
		}
		// それ以外は対象としてリストに追加
		filesToUpload = append(filesToUpload, file)
	}

	totalFiles := len(filesToUpload)
	if totalFiles == 0 {
		fmt.Println("INFO: アップする画像がないですヨ？")
		return
	}
	fmt.Printf("%d 件の画像を検出。並行処理を開始します。 \n", totalFiles)

	// ---------------------------
	// 並行処理の制御（Goroutine）
	// ---------------------------
	var wg sync.WaitGroup                        // 全ての Goroutine の完了を待つ監督
	sem := make(chan struct{}, MAX_CONCURRENCY)  // 同時に起動する Goroutine の最大数を制御
	var completedCount uint64

	startTime := time.Now()

	// 各ファイルに対してGoroutineを起動
	for _, file := range filesToUpload {
		
		filePath := filepath.Join(imageDir, file.Name())

		// １画像１タスク
		wg.Add(1)

		// 設定のマルチコアが処理できるタスク上限を超えた場合に、タスクはブロックされメモリ上で待機。バッファが空き次第、自動で再開される（担当するのがセマフォ）
		sem <- struct{}{}

		// -----------------
		// Goroutine 起動
		// -----------------
		go func(path string, name string) { 
			defer wg.Done()          // 処理の完了を通知
			defer func() { <-sem }() // 処理が完了したらセマフォを開放

			//----------------------
			// 画像ファイルの読み込み
			//----------------------
			record, err := loadImage(path)
			if err != nil {
				fmt.Printf("ERROR: %s の読み込みに失敗: %v\n", name, err)
				return // このGoroutineのみを終了させ、他のGoroutineの実行は継続
			}
			
			// ----------------
			// DBへのインサート
			// ----------------
			err = uploadToDB(db, record)
			if err != nil {
				fmt.Printf("ERROR: %s のアップに失敗: %v\n", name, err)
				return
			}
			
			newCount := atomic.AddUint64(&completedCount, 1)

			fmt.Printf("SUCCESS: (%d/%d) %s のアップに成功 \n", newCount, totalFiles, name)

		}(filePath, file.Name())

	}

	// すべてのGoroutineが完了するのを待つ
	wg.Wait()

	duration := time.Since(startTime)

	fmt.Println("INFO: 全てのアップロード処理が完了しました。")
	fmt.Printf("総数: %d\n", totalFiles)
	fmt.Printf("かかった時間: %s\n", duration)
	fmt.Printf("平均処理速度: %.2f files/sec\n", float64(totalFiles)/duration.Seconds())
}

// =============================
// 画像を保存するテーブルを作成
// =============================
func createTable(db *sql.DB) {
	query := `
	CREATE TABLE IF NOT EXISTS images (
		id INT AUTO_INCREMENT PRIMARY KEY,
		file_name VARCHAR(255) NOT NULL,
		file_size BIGINT NOT NULL,
		image_data LONGBLOB NOT NULL,
		uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(query)
	if err != nil {
		fmt.Printf("FATAL: テーブルの作成に失敗しました: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("SUCCESS: imagesテーブルの作成が完了")
}

// ==================================
//　画像を読み込み、バイトデータにする
// ==================================
func loadImage(filePath string) (ImageRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return ImageRecord{}, err
	}
	defer file.Close()

	// ファイルのメタデータ（サイズ、更新日時など）を取得
	fileInfo, err := file.Stat()
	if err != nil {
		return ImageRecord{}, err
	}

	// ファイル全体をメモリに読み込む
	data, err := io.ReadAll(file)
	if err != nil {
		return ImageRecord{}, err
	}

	// DBにインサートする画像一件のレコード
	return ImageRecord{
		FileName: filepath.Base(filePath),
		Size:     fileInfo.Size(),
		Data:     data,
	}, nil
}
// ===============================
// 1つの画像をDBにインサートする
// ===============================
func uploadToDB(db *sql.DB, record ImageRecord) error {
	stmt, err := db.Prepare("INSERT INTO images (file_name, file_size, image_data) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(record.FileName, record.Size, record.Data)
	return err
}