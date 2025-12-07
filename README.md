# 概要

数百～数千枚の画像をできるだけ早くDBへアップし保存するCLI


# 取り組み

- 早く実行するために...
    - I/Oの並行化（Goroutine）
        - コンテナが起動する環境のスレッド数を検出し、その値を上限にGoroutineとしてI/Oを実行

    - ソースコードの事前コンパイル
        - 初回ビルド以降はコンパイル済みファイルが実行されるため実行速度が向上

    - ディレクトリ構成
        - 画像データをDockerイメージにコピーするのを避け、ビルド時間を短縮

- 上げたデータは消えてほしくないので...
    - DBボリュームの永続化
        - DBコンテナが削除されてもデータは消えないよう設定


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


# 備忘録

## ライブラリのインストール
```bash
docker run --rm -v ${PWD}/app:/usr/src/app -w /usr/src/app golang:1.21 sh -c "go mod tidy"
```

## テーブルを追加する場合

1. `models` 配下に、テーブルのもととなる構造体を定義したファイルを作成

2. `migrate/migrate.go` にテーブルとして追加したいモデルを追加

3. コマンド実行
    ```bash
    docker-compose up
    ```

## その他ファイルの役割

- `config/config.go`
    - appコンテナで利用する定数を管理
