# 開発用の Go サーバーのイメージ。
#
# air というツールで「ファイルを保存したら自動で再ビルド＆再起動」させる。
# Railsの開発環境でコードを書き換えるとすぐ反映される、あの体験をGoでも再現するため。
# （Goはコンパイル言語なので、本来は書き換えるたびにビルドし直しが必要）
FROM golang:1.26-alpine

WORKDIR /app

# air をインストール（ホットリロード用ツール）
RUN go install github.com/air-verse/air@v1.67.4

# 先に go.mod / go.sum だけコピーして依存ライブラリをダウンロードしておく。
# Railsで Gemfile だけ先にコピーして bundle install するのと同じテクニック
# （ソースを変えても、依存が変わらなければこの層はキャッシュが効く）。
COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]
