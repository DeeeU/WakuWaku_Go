// Package handler は HTTPリクエストを受け取ってレスポンスを返す層。
// Railsの controller に相当する。
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// Pinger は「Ping(ctx) error というメソッドを持っているもの」を表すインターフェース。
//
// 【Go初登場: インターフェース】
// 「このメソッドを持っていればOK」という約束（型）。
// Rubyのダックタイピング（respond_to?(:ping) なら何でも受け入れる）を、
// コンパイル時にチェックできるようにしたもの、と考えると分かりやすい。
// Goでは implements と書く必要はなく、メソッドが揃っていれば自動的に満たしたことになる。
//
// PostgreSQL の *pgxpool.Pool はもともと Ping(ctx) error を持っているのでそのまま渡せる。
// Redis は repository.RedisPinger で形を合わせて渡している。
// テストでは「わざと失敗する偽物の Pinger」を渡せるので、DBなしでテストできる。
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler は GET /health を処理する。
// Railsでいうと HealthController のようなもの。依存（DB・Redis）をフィールドに持つ。
type HealthHandler struct {
	Postgres Pinger
	Redis    Pinger
}

// healthResponse は /health が返すJSONの形。
//
// `json:"status"` は「タグ」と呼ばれる注釈で、JSONに変換するときのキー名を指定している。
// Railsの render json: { status: "ok" } で書いていたキー名を、型に直接書いておくイメージ。
type healthResponse struct {
	Status   string `json:"status"`
	Postgres string `json:"postgres"`
	Redis    string `json:"redis"`
}

// Check は PostgreSQL と Redis に Ping して、両方OKなら 200、どちらかNGなら 503 を返す。
//
// echo のハンドラは func(c echo.Context) error という形で書く。
// c はリクエスト・レスポンスをまとめたもので、Railsの params や render をまとめた存在。
func (h *HealthHandler) Check(c echo.Context) error {
	// 2秒以内に終わらなければ諦める context を作る。
	// defer cancel() は「この関数を抜けるときに必ず cancel() を呼ぶ」予約。
	//
	// 【Go初登場: defer】
	// Rubyの begin ... ensure ... end の ensure 部分を、先に書いておけるイメージ。
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
	defer cancel()

	res := healthResponse{
		Status:   "ok",
		Postgres: pingStatus(ctx, h.Postgres),
		Redis:    pingStatus(ctx, h.Redis),
	}

	code := http.StatusOK
	if res.Postgres != "ok" || res.Redis != "ok" {
		res.Status = "ng"
		code = http.StatusServiceUnavailable // 503: サービスが使えない状態
	}

	// Railsの render json: res, status: code と同じ。
	return c.JSON(code, res)
}

// pingStatus は Ping の結果を "ok" か エラーメッセージ の文字列にする。
func pingStatus(ctx context.Context, p Pinger) string {
	if err := p.Ping(ctx); err != nil {
		return err.Error()
	}
	return "ok"
}
