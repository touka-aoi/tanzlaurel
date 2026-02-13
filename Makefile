.PHONY: server client install generate-mock test e2e

# サーバー起動
server:
	go run server/cmd/main.go

# クライアント起動
client:
	cd client && npm run dev

# クライアント依存関係インストール
install:
	cd client && npm install

# モック生成
generate-mock:
	go generate ./...

# テスト実行
test:
	go test ./... -v

# E2Eテスト実行
e2e:
	cd e2e && npm install && npm test