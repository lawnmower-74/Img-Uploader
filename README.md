# アップロード手順

1. `images/uploads` 下にアップしたい画像を配置

2. コマンド実行
    ```bash
    docker-compose up --build
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
docker run --rm -v ${PWD}/app:/usr/src/app -w /usr/src/app golang:1.21 sh -c "go get github.com/go-sql-driver/mysql"
```

# 備忘録

- 並行処理
    - 1つのコアがタスクを高速で切り替えて実行することであたかも同時に進行しているように見せてる

- 並列処理
    - 実際に複数のコアが別々のタスクを同時に実行するからこっちは本当の意味で同時進行

## Goroutine
