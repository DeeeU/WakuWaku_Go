// Package config は、アプリの設定値（DBの接続先など）を環境変数から読み込む。
//
// Railsでいうと config/database.yml や ENV["DATABASE_URL"] を読む部分に相当する。
// Goには「規約で決まった設定ファイル」が無いので、自分で読み込み処理を書く。
package config

import "os"

// Config はアプリ全体の設定をまとめた「構造体（struct）」。
//
// 【Go初登場: 構造体】
// 複数の値をひとまとめにした型。Rubyの Struct.new(:port, :database_url) や
// 「メソッドを持たない属性だけのクラス」をイメージすると近い。
// フィールド名が大文字で始まると、パッケージの外から参照できる（public）。
// 小文字だとパッケージ内だけで使える（private）。Rubyの public/private を名前の大小で表す仕組み。
type Config struct {
	Port        string // サーバーが待ち受けるポート番号（例: "8080"）
	DatabaseURL string // PostgreSQL の接続文字列
	RedisAddr   string // Redis の接続先（例: "redis:6379"）
}

// Load は環境変数から設定を読み込んで返す。
// 環境変数が無いときは、ローカル開発用のデフォルト値を使う。
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://wakuwaku:wakuwaku@localhost:5432/wakuwaku?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
	}
}

// getEnv は環境変数 key を読み、空なら fallback を返す。
// Rubyの ENV.fetch("PORT", "8080") と同じ役割。
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
