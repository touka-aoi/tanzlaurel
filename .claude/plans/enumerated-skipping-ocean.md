# デプロイ環境構築プラン

## Context

CRDTブログ（web/）を GCE 単一インスタンスに Docker Compose でデプロイする。
DB（ScyllaDB）と管理画面は並行ブランチで開発中のため、まずは現行の JSONファイル永続化構成でデプロイし、後から追加できる構成にする。
インフラは Terraform + Atlantis（Cloud Run）で IaC 管理する。

## 全体構成

```
GitHub PR
  │  terraform/ 配下の変更
  ▼
Atlantis (Cloud Run)
  │  terraform plan/apply
  ▼
GCP
┌─────────────────────────────────────────────┐
│ VPC + Firewall (80/443)                     │
│                                             │
│ GCE Instance (e2-small)                     │
│ ┌─────────────────────────────────────────┐ │
│ │ Docker Compose                          │ │
│ │ ┌─────────┐    ┌──────────────┐        │ │
│ │ │ Caddy   │───▶│ Go Server    │        │ │
│ │ │ :80/443 │    │ :8080        │        │ │
│ │ │(TLS終端)│    │(API+WS+静的) │        │ │
│ │ └─────────┘    └──────┬───────┘        │ │
│ │                  volume:/data           │ │
│ │                 (JSON永続化)            │ │
│ │ ┌───────────────────────────────┐      │ │
│ │ │ (将来) ScyllaDB :9042        │      │ │
│ │ └───────────────────────────────┘      │ │
│ └─────────────────────────────────────────┘ │
│                                             │
│ Cloud DNS (ドメイン管理)                     │
└─────────────────────────────────────────────┘
```

## Part 1: アプリケーションのコンテナ化 (web/ 配下)

### 1-1. Go サーバーに静的ファイル配信を追加

`web/server/router.go` を修正:
- `STATIC_DIR` 環境変数でディスクからフロントエンドを配信
- SPA 対応: 存在しないパスは `index.html` にフォールバック
- `/api/*` は既存ハンドラーが優先

### 1-2. `web/Dockerfile` — マルチステージビルド

```dockerfile
# Stage 1: フロントエンドビルド
FROM node:22-alpine AS frontend
WORKDIR /app/cockpit
COPY cockpit/package*.json ./
RUN npm ci
COPY cockpit/ ./
RUN npm run build

# Stage 2: Go バイナリビルド
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY server/ ./server/
RUN CGO_ENABLED=0 go build -o /bin/server ./server/cmd/

# Stage 3: 実行イメージ
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=backend /bin/server /usr/local/bin/server
COPY --from=frontend /app/cockpit/dist /srv/static
EXPOSE 8080
ENV DATA_DIR=/data
CMD ["server"]
```

### 1-3. `web/Caddyfile`

```
{$DOMAIN:localhost} {
    reverse_proxy server:8080
}
```

Caddy は TLS 終端のみ。全リクエストを Go サーバーへプロキシ。

### 1-4. `web/docker-compose.yml`

```yaml
services:
  server:
    build: .
    volumes:
      - blog-data:/data
    environment:
      - ADDRESS=:8080
      - DATA_DIR=/data
      - ENV=production
      - STATIC_DIR=/srv/static
    restart: unless-stopped

  caddy:
    image: caddy:2-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - caddy-data:/data
      - caddy-config:/config
    depends_on:
      - server
    restart: unless-stopped

volumes:
  blog-data:
  caddy-data:
  caddy-config:
```

### 1-5. `web/.dockerignore`

```
cockpit/node_modules/
server/bin/
data/
*.pid
.claude/
```

## Part 2: Terraform (terraform/ 配下、リポジトリルート)

### ディレクトリ構成

```
terraform/
├── main.tf           # provider, backend
├── variables.tf      # 変数定義
├── terraform.tfvars  # 値（.gitignore対象、例は .tfvars.example に）
├── network.tf        # VPC, サブネット, ファイアウォール
├── compute.tf        # GCE インスタンス + startup script
├── dns.tf            # Cloud DNS ゾーン + レコード
├── atlantis.tf       # Atlantis 用 Cloud Run サービス
├── outputs.tf        # IP アドレス等の出力
└── atlantis.yaml     # Atlantis リポジトリ設定
```

### 2-1. `terraform/main.tf`

- Provider: `google` (GCP)
- Backend: GCS バケット（state 保存）
- Required providers バージョン固定

### 2-2. `terraform/network.tf`

- VPC ネットワーク
- サブネット
- ファイアウォールルール: HTTP(80), HTTPS(443), SSH(22) を許可

### 2-3. `terraform/compute.tf`

- `google_compute_instance` (e2-small)
- startup script で Docker + Docker Compose をインストール
- リポジトリを clone して `docker compose up -d`
- サービスアカウント付与
- 外部 IP（静的）

### 2-4. `terraform/dns.tf`

- `google_dns_managed_zone`
- A レコード → GCE 外部 IP

### 2-5. `terraform/atlantis.tf`

- `google_cloud_run_v2_service` で Atlantis をデプロイ
- GitHub Webhook 用の URL を出力
- Atlantis 用サービスアカウント（Terraform 実行権限）

### 2-6. `atlantis.yaml` (リポジトリルート)

```yaml
version: 3
projects:
  - name: crdt-blog-infra
    dir: terraform
    autoplan:
      when_modified: ["*.tf", "*.tfvars"]
      enabled: true
```

## Part 3: CI/CD (GitHub Actions)

### 3-1. `.github/workflows/deploy.yml`

```yaml
name: Deploy CRDT Blog

on:
  push:
    branches: [main]
    paths: [web/**]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.26' }
      - run: cd web && go test ./...
      - uses: actions/setup-node@v4
        with: { node-version: '22' }
      - run: cd web/cockpit && npm ci && npx vitest run

  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: SSH deploy
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.GCE_HOST }}
          username: ${{ secrets.GCE_USER }}
          key: ${{ secrets.GCE_SSH_KEY }}
          script: |
            cd /opt/crdt-blog
            git pull origin main
            docker compose -f web/docker-compose.yml build
            docker compose -f web/docker-compose.yml up -d
```

## 実装ステップ

### Phase A: コンテナ化
1. Go サーバーに静的ファイル配信を追加 (`web/server/router.go`)
2. Dockerfile 作成 (`web/Dockerfile`)
3. .dockerignore 作成 (`web/.dockerignore`)
4. Caddyfile 作成 (`web/Caddyfile`)
5. docker-compose.yml 作成 (`web/docker-compose.yml`)
6. ローカルで `docker compose up` して動作確認

### Phase B: Terraform
7. terraform/ ディレクトリ作成（main, network, compute, dns, atlantis）
8. atlantis.yaml 作成

### Phase C: CI/CD
9. GitHub Actions ワークフロー作成

## 将来の拡張ポイント

- ScyllaDB: `docker-compose.yml` にサービス追加、Terraform で永続ディスク追加
- 管理画面: 同一 Go サーバーに認証ミドルウェア追加
- ドメイン: `DOMAIN` 環境変数で Caddy が自動 HTTPS

## 検証方法

```bash
# コンテナ化の検証
cd web && docker compose build && docker compose up -d
# http://localhost でフロント表示、/api/health が 200
docker compose down

# Terraform の検証
cd terraform && terraform init && terraform plan
```
