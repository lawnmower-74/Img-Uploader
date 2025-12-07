# 概要

数百～数千枚の画像をできるだけ早くアップし保存するCLI

# 取り組み

- コードの最適化
    - Goroutineによるアップロードの並行処理
        - コンテナが起動する環境の論理コア数を検出し、その値を上限にGoroutineを実行可能
        - 



# アップロード手順

1. `images/upload` 下にアップしたい画像を配置
    - ※名前など変更したい場合
        - `docker-compose.yml` のマウント箇所を編集
        - `config/config.go` を編集

2. DB起動・マイグレートの実行
    ```bash
    docker-compose up --build
    ```

3. アップロード実行
    ```bash
    docker-compose run app go run uploader.go
    ```

3. 完了したら
    ```bash
    docker-compose down

    # DB内データも削除する場合はこちら
    docker-compose down -v
    ```


# ライブラリ関連

## mod生成
```bash
docker run --rm -v ${PWD}/app:/usr/src/app -w /usr/src/app golang:1.21 sh -c "go mod init uploader"
```
- `go.mod` がなければ要実行
- `golang: n`: `go get ...` などで利用するGoのバージョン

## ライブラリのインストール
```bash
docker run --rm -v ${PWD}/app:/usr/src/app -w /usr/src/app golang:1.21 sh -c "go mod tidy"
```


# テーブル追加手順

1. `models` 配下に、もととなる構造体を定義したファイルを作成

2. `migrate/migrate.go` に追加したいモデルを追加

3. コマンド実行
    ```bash
    docker-compose up
    ```


# 備忘録

## ファイルの役割

- `config/config.go`
    - appコンテナで利用する定数を管理

- 