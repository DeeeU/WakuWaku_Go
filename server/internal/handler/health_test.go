package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

// fakePinger はテスト用の偽物 Pinger。err に値を入れておくと Ping が失敗する。
// RSpec の double(:pinger, ping: nil) に相当する。
type fakePinger struct {
	err error
}

func (f fakePinger) Ping(ctx context.Context) error {
	return f.err
}

// 【Go初登場: テスト】
// Goのテストは「ファイル名が _test.go」「関数名が Test で始まる」「引数が *testing.T」というルール。
// `go test ./...` で実行できる。RSpecのような DSL は無く、普通の関数で書く。
//
// ここでは「テーブル駆動テスト」というGoでよく使う書き方をしている。
// テストケースを配列で並べて、for ループで順番に検証する（RSpecの複数の it をまとめたイメージ）。
func TestHealthHandler_Check(t *testing.T) {
	tests := []struct {
		name       string
		postgres   Pinger
		redis      Pinger
		wantCode   int
		wantStatus string
	}{
		{"両方OK", fakePinger{}, fakePinger{}, http.StatusOK, "ok"},
		{"PostgreSQLが落ちている", fakePinger{err: errors.New("pg down")}, fakePinger{}, http.StatusServiceUnavailable, "ng"},
		{"Redisが落ちている", fakePinger{}, fakePinger{err: errors.New("redis down")}, http.StatusServiceUnavailable, "ng"},
	}

	for _, tt := range tests {
		// t.Run でサブテストにすると、失敗したときにどのケースか名前で分かる（RSpecの context / it）。
		t.Run(tt.name, func(t *testing.T) {
			// httptest は「本物のサーバーを立てずに」リクエスト・レスポンスを作れる標準ライブラリ。
			// Railsの request spec で get "/health" するのに近い。
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			h := &HealthHandler{Postgres: tt.postgres, Redis: tt.redis}
			if err := h.Check(c); err != nil {
				t.Fatalf("予期しないエラー: %v", err)
			}

			if rec.Code != tt.wantCode {
				t.Errorf("ステータスコード: got %d, want %d", rec.Code, tt.wantCode)
			}

			var body healthResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("JSONの解析に失敗: %v", err)
			}
			if body.Status != tt.wantStatus {
				t.Errorf("status: got %q, want %q", body.Status, tt.wantStatus)
			}
		})
	}
}
