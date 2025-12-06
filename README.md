# アップロード手順

1. `images/upload` 下にアップしたい画像を配置

2. コマンド実行
    ```bash
    docker-compose up --build
    ```

3. 
    ```bash
    docker-compose run app go run uploader.go
    ```


3. 完了したら
    ```bash
    docker-compose down

    # DB内データも削除する場合はこちら
    docker-compose down -v
    ```


# mod生成
```bash
docker run --rm -v ${PWD}/app:/usr/src/app -w /usr/src/app golang:1.21 sh -c "go mod init uploader"
```
- `go.mod` がなければ要実行
- `golang: n`: `go get ...` などで利用するGoのバージョン


# ライブラリのインストール
```bash
docker run --rm -v ${PWD}/app:/usr/src/app -w /usr/src/app golang:1.21 sh -c "go mod tidy"
```


# 高速化の取り組み

Goroutineによるアップロードの並行処理

カラムへのインデックス付与

# 