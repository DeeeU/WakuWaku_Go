package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient は Redis クライアントを作り、疎通確認してから返す。
//
// Railsでいうと Redis.new(url: ENV["REDIS_URL"]) に相当する。
// go-redis もクライアントの内部でコネクションプールを持っている。
func NewRedisClient(ctx context.Context, addr string) (*redis.Client, error) {
	// &redis.Options{...} は「構造体を作って、そのポインタ（参照）を渡す」書き方。
	//
	// 【Go初登場: ポインタ（& と *）】
	// &x は「x の置き場所（アドレス）」、*T は「T の置き場所を指す型」。
	// Rubyではオブジェクトは常に参照で渡されるので意識しないが、
	// Goでは「値をコピーして渡す」か「参照（ポインタ）を渡す」かを自分で選ぶ。
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	// Ping(ctx) は結果オブジェクトを返し、.Err() でエラーだけ取り出せる。
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis に接続できません: %w", err)
	}

	return client, nil
}

// RedisPinger は redis.Client を handler.Pinger インターフェースに合わせるための小さな包み（アダプタ）。
//
// redis.Client の Ping は error ではなく *StatusCmd を返すため、
// 「Ping(ctx) error」という形に変換している。
type RedisPinger struct {
	Client *redis.Client
}

// Ping は Redis に PING コマンドを送り、失敗したら error を返す。
//
// 【Go初登場: メソッド】
// func (p RedisPinger) Ping(...) の (p RedisPinger) は「レシーバ」と呼ばれ、
// 「RedisPinger 型にメソッドを生やす」という意味。Rubyの def ping をクラス内に書くのと同じ。
// p は Rubyの self に相当する。
func (p RedisPinger) Ping(ctx context.Context) error {
	return p.Client.Ping(ctx).Err()
}
