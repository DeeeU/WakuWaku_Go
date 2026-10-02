// main パッケージの main 関数が、Goプログラムの入り口（起動処理）。
// Railsでいうと `rails server` を叩いたときに config/application.rb や
// config/routes.rb が読まれて起動する、その一連の流れを自分で書いている場所。
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/deeeu/wakuwaku_go/server/internal/config"
	"github.com/deeeu/wakuwaku_go/server/internal/handler"
	"github.com/deeeu/wakuwaku_go/server/internal/repository"
)

func main() {
	// 1. 設定を読み込む
	cfg := config.Load()

	// 起動時の接続処理は10秒以内に終わらせる
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. PostgreSQL に接続
	db, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		// log.Fatalf はログを出してプログラムを終了する（Rubyの abort に近い）
		log.Fatalf("起動失敗: %v", err)
	}
	defer db.Close()

	// 3. Redis に接続
	rdb, err := repository.NewRedisClient(ctx, cfg.RedisAddr)
	if err != nil {
		log.Fatalf("起動失敗: %v", err)
	}
	defer rdb.Close()

	// 4. echo（Webフレームワーク）を準備
	e := echo.New()
	e.HideBanner = true

	// ミドルウェアの登録。
	// Railsの before_action / Rack ミドルウェアに近い。全リクエストの前後に共通処理を挟む。
	e.Use(middleware.Logger())  // アクセスログを出す（Railsの log/development.log の Started GET ... のような出力）
	e.Use(middleware.Recover()) // ハンドラ内で panic（想定外のクラッシュ）が起きてもサーバーを落とさない
	e.Use(middleware.CORS())    // 別オリジン（例: 地図画面）からAPIを呼べるようにする

	// 5. ルーティング（Railsの config/routes.rb に相当）
	//    get "/health", to: "health#check" と同じ意味。
	health := &handler.HealthHandler{
		Postgres: db,
		Redis:    repository.RedisPinger{Client: rdb},
	}
	e.GET("/health", health.Check)

	// 6. サーバーを起動する
	//
	// 【Go初登場: goroutine（ゴルーチン）】
	// `go 関数()` と書くと、その関数が「並行して」動き出す。軽量なスレッドのようなもの。
	// Rubyの Thread.new { ... } に近いが、ずっと軽く、何千個でも作れる。
	// ここでは「サーバー起動」を別の goroutine で走らせ、
	// メインの処理は下で「終了シグナル（Ctrl+C など）」を待つ。
	go func() {
		log.Printf("サーバー起動: http://localhost:%s", cfg.Port)
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("サーバーエラー: %v", err)
		}
	}()

	// 7. 終了シグナルを待って、きれいに終了する（グレースフルシャットダウン）
	//
	// 【用語: グレースフルシャットダウン】
	// いきなり落とさず「処理中のリクエストを返し終えてから」止めること。
	// `docker compose down` で送られる SIGTERM を受け取ったときに使われる。
	//
	// 【Go初登場: チャネル（chan）】
	// goroutine 同士で値をやりとりする「パイプ」。
	// <-quit は「quit に値が届くまでここで待つ」という意味。
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("シャットダウン中...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("シャットダウンエラー: %v", err)
	}
}
