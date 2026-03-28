.PHONY: server client bot dev install generate-mock test e2e

# サーバー起動
server:
	go run server/cmd/main.go

# クライアント起動
client:
	cd client && npm run dev

# ボット起動 (BOT_COUNT=5 make bot)
BOT_COUNT ?= 3
bot:
	BOT_COUNT=$(BOT_COUNT) go run server/cmd/bot/main.go

# 全起動 (サーバー + クライアント + ボット)
dev:
	@echo "Starting server, client, and bots..."
	@trap 'kill 0' EXIT; \
	go run server/cmd/main.go & \
	sleep 1 && cd client && npm run dev & \
	sleep 2 && BOT_COUNT=$(BOT_COUNT) go run server/cmd/bot/main.go & \
	wait

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