package config

import (
	"log"
	"runtime"
)

type Config struct {
	ImageDir	string
	MaxWorkers	int
}

var AppConfig Config

func init() {
	loadConfig()
}

// ===================================
// 必要あれば以下の設定を書き換えてネ
// ===================================
func loadConfig() {
	log.Println("設定をロード中...")

	// アップロード元ディレクトリ
	AppConfig.ImageDir = "/images/upload"

	// 並列・並行含め同時に処理できるタスクの最大数
	AppConfig.MaxWorkers = runtime.NumCPU()

	log.Println("------- 設定値 -------")
	log.Printf("AppConfig: %#v", AppConfig)
	log.Println("----------------------\n")
}