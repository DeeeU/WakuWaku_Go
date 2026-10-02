// Package repository は、DB（PostgreSQL）や Redis への読み書きを担当する。
//
// Railsでは ActiveRecord のモデルが「DBへの接続」も「データの読み書き」も
// 自動でやってくれるが、Goでは接続の作成から自分で書く。
// このフェーズでは「接続を作る」部分だけを置いている。
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPostgresPool は PostgreSQL への「コネクションプール」を作って返す。
//
// 【用語: コネクションプール】
// DB接続を毎回作ると遅いので、いくつかの接続を作り置きしておき、使い回す仕組み。
// Railsでも database.yml の pool: 5 で同じことをしている（ActiveRecordが裏でやってくれていた）。
//
// 【Go初登場: 戻り値が2つ ＆ エラー処理】
// Goの関数は値を複数返せる。慣習として「最後の戻り値が error」になっている。
// Rubyは失敗すると例外を raise するが、Goは「エラーを戻り値として返す」。
// 呼び出し側は毎回 `if err != nil { ... }` でチェックする（Goで一番よく見る書き方）。
//
// 【Go初登場: context.Context】
// 「この処理はいつまでに終わらせるか」「途中でキャンセルされたか」を伝える入れ物。
// DBアクセスなど時間のかかる処理には、第1引数で渡すのがGoのお作法。
func NewPostgresPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		// fmt.Errorf の %w は「元のエラーを包む」書き方。
		// Rubyで raise MyError, "..." するときに cause を残すイメージ。
		return nil, fmt.Errorf("postgres の接続設定に失敗: %w", err)
	}

	// pgxpool.New は「実際にはまだ接続しない」ことがあるので、Ping で疎通確認する。
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres に接続できません: %w", err)
	}

	return pool, nil
}
