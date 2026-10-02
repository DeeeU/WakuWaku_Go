# LEARNING.md — 学習ノート

フェーズごとに「何を作ったか」「新しく出てきた概念」「改造課題」をまとめます。
Ruby / Rails と比べながら読めるように書いています。

---

## フェーズ1：環境構築

### 何を作ったか

- `docker compose up` で **Goサーバー / PostgreSQL / Redis** の3つが起動する環境
- `GET /health`：PostgreSQL と Redis に接続できているかを返すAPI
  - 両方OKなら `200 {"status":"ok",...}`、どちらかNGなら `503 {"status":"ng",...}`
- air によるホットリロード（保存すると自動で再ビルド・再起動）
- `/health` のテスト（`go test ./...`）

### リクエストが処理される流れ

```
curl /health
  → echo（ルーティング: e.GET("/health", ...)）   … Rails の routes.rb
  → ミドルウェア（Logger, Recover, CORS）           … Rack ミドルウェア / before_action
  → handler.HealthHandler.Check                   … HealthController#check
      → PostgreSQL に Ping / Redis に Ping
  → c.JSON(200, ...)                              … render json:
```

### 新しく出てきた概念

#### 1. Goのプロジェクト構成（`go.mod`, `package`, `internal`）

| Go | Ruby/Rails | 説明 |
|---|---|---|
| `go.mod` | `Gemfile` | 使うライブラリとバージョンを書く |
| `go.sum` | `Gemfile.lock` | ライブラリの中身のチェックサム（改ざん防止） |
| `go mod tidy` | `bundle install` | 必要なライブラリを揃え、不要なものを消す |
| `package handler` | `module Handler` | ファイルの所属グループ。**1フォルダ = 1パッケージ** |
| `internal/` | — | このフォルダの中身は、このモジュールの外から import できない（Goの特別なルール） |
| `cmd/server/main.go` | `bin/rails server` | プログラムの入り口。`package main` の `func main()` から実行が始まる |

Railsは「規約（Convention over Configuration）」でファイルの置き場所が決まっていましたが、
Goは**フレームワークが構成を決めない**ので、自分で構成を決めます。今回の `handler / repository / service` 分けは、Goでよく使われる形です。

#### 2. エラーは「戻り値」で返す（`if err != nil`）

```go
db, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL)
if err != nil {
    log.Fatalf("起動失敗: %v", err)
}
```

```ruby
# Rubyなら例外で書く
begin
  db = connect_postgres(url)
rescue => e
  abort "起動失敗: #{e.message}"
end
```

- Goには例外（raise / rescue）が基本的にありません。失敗する可能性のある関数は**最後の戻り値で `error` を返します**
- 呼び出し側は毎回 `if err != nil` でチェックします。冗長に見えますが「どこで失敗しうるか」がコードを読むだけで分かる、というのがGoの考え方です
- `fmt.Errorf("...: %w", err)` で、元のエラーに説明を足して包めます

#### 3. 構造体（struct）とメソッド

```go
type HealthHandler struct {
    Postgres Pinger
    Redis    Pinger
}

func (h *HealthHandler) Check(c echo.Context) error { ... }
```

```ruby
class HealthHandler
  attr_reader :postgres, :redis
  def check(c) ... end
end
```

- Goには `class` がありません。**データは struct、振る舞いはメソッド**として別々に書きます
- `func (h *HealthHandler) Check(...)` の `(h *HealthHandler)` が「どの型のメソッドか」を表す部分（レシーバ）。`h` は Ruby の `self`
- 名前が**大文字始まりなら public、小文字始まりなら private**（パッケージ外から見えない）

#### 4. インターフェース

```go
type Pinger interface {
    Ping(ctx context.Context) error
}
```

- 「`Ping(ctx) error` というメソッドを持っていれば何でもOK」という型
- Rubyのダックタイピングに近いですが、**コンパイル時にチェックされる**のが違い
- `implements Pinger` のような宣言は不要。メソッドが揃っていれば自動で満たします
- 使いどころ：テストで「偽物」を差し込める。`health_test.go` では `fakePinger` を渡して、本物のDBなしでテストしています（RSpecの `double` に相当）

#### 5. context（コンテキスト）

```go
ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
defer cancel()
```

- 「この処理は2秒でタイムアウト」「クライアントが切断したら中断」といった情報を運ぶ入れ物
- DB・Redis・外部APIなど**時間がかかる処理の第1引数に渡すのがGoのお作法**
- Railsではタイムアウトは database.yml などに設定して自動で効いていましたが、Goでは処理ごとに明示的に渡します

#### 6. defer

- `defer cancel()` は「この関数を抜けるときに `cancel()` を必ず実行する」という予約
- Rubyの `ensure` に近い。`defer db.Close()` のように「開いたらすぐ閉じる予約をする」のが定番パターン

#### 7. goroutine とチャネル（main.go のグレースフルシャットダウン）

```go
go func() { e.Start(":8080") }()   // 別の goroutine でサーバー起動

quit := make(chan os.Signal, 1)
signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
<-quit                              // シグナルが来るまでここで待つ
```

- `go 関数()` で関数を並行実行できる（**goroutine**）。Rubyの `Thread.new { }` に近いが、とても軽量
- `chan`（**チャネル**）は goroutine 間で値を受け渡すパイプ。`<-quit` は値が届くまで待つ
- フェーズ4の WebSocket 配信でたくさん使うので、ここで雰囲気をつかんでおきましょう

#### 8. Docker まわり

| 用語 | 説明 |
|---|---|
| イメージ | アプリを動かすための「型」。`Dockerfile` から作る |
| コンテナ | イメージから起動した実体 |
| `docker-compose.yml` | 複数コンテナ（server / db / redis）をまとめて定義・起動する設定 |
| `depends_on` + `healthcheck` | 「DBの準備ができてからサーバーを起動する」ための設定 |
| ボリューム | コンテナを消しても残る保存領域。DBのデータなどに使う |

- compose の中では、**サービス名がそのままホスト名**になります（`db:5432`, `redis:6379`）
- Goはコンパイル言語なので、本来はコードを変えるたびにビルドし直しが必要です。**air** がファイル変更を検知して自動でビルド＆再起動してくれるので、Railsに近い開発体験になります

#### 9. コネクションプール

- DB接続を毎回作ると遅いので、何本か作り置きして使い回す仕組み
- Railsでは `database.yml` の `pool: 5` で ActiveRecord が自動管理していました。Goでは `pgxpool.New(...)` で自分で作ります
- go-redis も内部でプールを持っています

### 改造課題

1. **`/health` にサーバー時刻を追加してみよう**
   - `healthResponse` に `Time string \`json:"time"\`` を追加し、`time.Now().Format(time.RFC3339)` を入れて返す
   - 保存するだけで air が再起動し、`curl` の結果が変わることを確認する
2. **`GET /ping` を追加してみよう**
   - DBなどにはアクセスせず `{"message":"pong"}` を返すだけのハンドラを作り、`main.go` にルーティングを追加する
   - できたら `health_test.go` を参考にテストも書いてみる
