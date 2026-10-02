# WakuWaku_Go — 友達の現在地共有アプリ

同じグループの友達の現在地を、地図上にリアルタイム表示するアプリです。
タクシーアプリ「GO」と同じ技術スタック（Go / PostgreSQL / Redis / Docker / WebSocket）を体験する**学習用プロジェクト**です。

学習メモは [LEARNING.md](./LEARNING.md) にまとめています。

## 技術スタック

| 領域 | 採用技術 | Railsでいうと |
|---|---|---|
| サーバー言語 | Go | Ruby |
| Webフレームワーク | echo | routes + controller |
| DB | PostgreSQL（pgx + 生SQL） | ActiveRecord の接続先DB |
| 最新位置の保存・検索 | Redis（GEO機能） | キャッシュ用Redis |
| リアルタイム通信 | WebSocket | Action Cable |
| 開発環境 | Docker / docker compose | — |

## 必要なもの

- Docker（Docker Desktop など）。`docker compose` コマンドが使えればOK
- （任意）手元で `go test` を実行したい場合は Go 1.25 以上

## 起動方法

```bash
docker compose up
```

初回はイメージのビルドとライブラリのダウンロードで数分かかります。
ログに `サーバー起動: http://localhost:8080` と出れば起動完了です。

止めるときは `Ctrl+C`、またはバックグラウンド起動（`docker compose up -d`）なら `docker compose down`。
DBのデータも消したいときは `docker compose down -v`。

### 起動するサービス

| サービス | ポート | 説明 |
|---|---|---|
| server | 8080 | Go のAPIサーバー（ファイル保存で自動再起動） |
| db | 5432 | PostgreSQL（ユーザー/パスワード/DB名: すべて `wakuwaku`） |
| redis | 6379 | Redis |

## 動作確認手順

### フェーズ1：環境構築

1. `docker compose up` で起動する
2. 別のターミナルでヘルスチェックを叩く

   ```bash
   curl -i http://localhost:8080/health
   ```

   次のように返れば成功です（PostgreSQL と Redis の両方につながっている）。

   ```
   HTTP/1.1 200 OK
   {"status":"ok","postgres":"ok","redis":"ok"}
   ```

3. （確認）Redis を止めると 503 が返ることを確認する

   ```bash
   docker compose stop redis
   curl -i http://localhost:8080/health   # → 503 と "status":"ng"
   docker compose start redis
   ```

4. （確認）ホットリロード：`server/internal/handler/health.go` の `"ok"` を書き換えて保存すると、数秒後にサーバーが自動で再起動し、レスポンスが変わる

### テストの実行

```bash
# コンテナの中で実行（Goを手元に入れていなくてOK）
docker compose exec server go test ./...

# 手元にGoがある場合
cd server && go test ./...
```

## ディレクトリ構成

```
/
├── docker-compose.yml   # 全サービスの起動設定
├── server/              # Go のAPIサーバー
│   ├── Dockerfile       # 開発用イメージ（air でホットリロード）
│   ├── .air.toml        # air の設定
│   ├── cmd/server/      # main.go（起動処理）
│   ├── internal/
│   │   ├── config/      # 環境変数から設定を読む
│   │   ├── handler/     # Railsの controller 相当
│   │   ├── repository/  # DB・Redis への接続・読み書き
│   │   ├── service/     # ビジネスロジック（フェーズ2以降）
│   │   └── realtime/    # WebSocket 関連（フェーズ4）
│   └── migrations/      # DBマイグレーション（フェーズ2）
├── web/                 # 動作確認用の地図画面（フェーズ5）
├── simulator/           # ダミーユーザーを動かすスクリプト（フェーズ6）
├── README.md
└── LEARNING.md
```

## トラブルシューティング

- **ポートが使われていると言われる**：手元で PostgreSQL / Redis / 別アプリが 5432 / 6379 / 8080 を使っていないか確認してください。止めるか、`docker-compose.yml` の `ports` の左側の数字を変えます（例 `"15432:5432"`）
- **ファイルを保存しても再起動しない（Mac/Windows）**：`server/.air.toml` の `poll = false` を `true` にしてください
