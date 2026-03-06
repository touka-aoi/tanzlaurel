.PHONY: server client dev stop restart

SERVER_PID := server/.pid
CLIENT_PID := cockpit/.pid

# サーバー起動（バックグラウンド、PIDをdump）
server:
	@go build -o server/bin/crdt-blog ./server/cmd/ && \
	server/bin/crdt-blog & \
	echo $$! > $(SERVER_PID) && \
	echo "Started server (PID: $$(cat $(SERVER_PID)))"

# フロント起動（バックグラウンド、PIDをdump）
client:
	@cd cockpit && npx vite & \
	echo $$! > $(CLIENT_PID) && \
	echo "Started client (PID: $$(cat $(CLIENT_PID)))"

# 全起動
dev: server client

# 停止（PIDファイル + ポートの両方で確実にkill）
stop:
	@if [ -f $(SERVER_PID) ]; then \
		kill $$(cat $(SERVER_PID)) 2>/dev/null && echo "Stopped server (PID: $$(cat $(SERVER_PID)))"; \
		rm -f $(SERVER_PID); \
	else \
		echo "No server PID file found"; \
	fi
	@lsof -ti:8080 2>/dev/null | xargs kill 2>/dev/null || true
	@if [ -f $(CLIENT_PID) ]; then \
		kill $$(cat $(CLIENT_PID)) 2>/dev/null && echo "Stopped client (PID: $$(cat $(CLIENT_PID)))"; \
		rm -f $(CLIENT_PID); \
	else \
		echo "No client PID file found"; \
	fi
	@lsof -ti:5173 2>/dev/null | xargs kill 2>/dev/null || true

# 再起動
restart: stop dev
