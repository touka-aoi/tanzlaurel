# フロントエンドデザイン刷新: GitHub Feed風 + glass-ui

## Context

現在のフロントは3ペイン（サイドバー + エディタ + プレビュー）レイアウト。
GitHub.comのフィードページ風の縦並びレイアウトに変更し、glass-uiデザインを適用する。

## UX設計

### フィードページ
```
┌─────────────────────────────────────────┐
│  Flourish                               │  ← ヘッダー
├─────────────────────────────────────────┤
│  ┌───────────────────────────────────┐  │
│  │ 本文を入力...                     │  │  ← 新規入力欄（glass card）
│  │                                   │  │    本文のみ。タイトルは1行目から自動導出
│  │                          [投稿]   │  │
│  └───────────────────────────────────┘  │
│                                         │
│  ┌───────────────────────────────────┐  │
│  │ タイトル              2026-03-01  │  │  ← 記事カード（glass card）
│  │ Markdownプレビュー表示            │  │    通常: プレビュー表示
│  │ ...                               │  │    クリック: その場でtextareaに切替
│  └───────────────────────────────────┘  │    外クリックでプレビューに戻る
│                                         │
│  ┌───────────────────────────────────┐  │
│  │ タイトル              2026-02-28  │  │
│  │ ...                               │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

### 記事カードの状態
- **閲覧モード**: Markdownレンダリング表示
- **編集モード**: カードクリックでtextareaに切替、CRDTリアルタイム同期。外クリックでプレビューに戻る

### 削除
- 認証実装前のため、UIから削除ボタンを非表示にする

## glass-uiデザイントークン

既存ダークテーマベース:
- 背景: ダークグラデーション
- カード: `backdrop-blur-xl bg-white/[0.05] border border-white/[0.08] rounded-2xl shadow-lg`
- 入力欄: `bg-white/[0.03] border border-white/[0.1]`
- ボタン: `bg-blue-500/20 backdrop-blur border border-blue-400/30`
- テキスト: `text-white/90`（本文）、`text-white/50`（サブ）

## 変更対象ファイル

| ファイル | 変更内容 |
|---|---|
| `src/app.tsx` | 3ペイン → フィードレイアウト |
| `src/components/compose-box.tsx` | **新規** 上部の新規入力欄 |
| `src/components/feed-card.tsx` | **新規** 記事カード（プレビュー/編集切替） |
| `src/components/entry-list.tsx` | **削除** |
| `src/components/entry-editor.tsx` | **削除** |
| `src/components/markdown-preview.tsx` | **削除** |
| `src/index.css` | glass-uiカスタムスタイル追加 |
| `src/hooks/use-document.ts` | 編集中カードのみ接続するよう調整 |

## 実装ステップ

### Step 1: index.css にglass-uiベーススタイル

### Step 2: compose-box.tsx
- textarea + 投稿ボタン
- 投稿時: createEntry → CRDT接続 → テキスト送信 → 入力欄クリア
- glass-uiカードスタイル

### Step 3: feed-card.tsx
- props: entry, isEditing, onStartEdit, onStopEdit
- 閲覧モード: markdown-itでレンダリング（既存のlib/markdown.ts再利用）
- 編集モード: textarea + useDocumentでCRDT同期
- 日付表示、接続状態インジケーター

### Step 4: app.tsx リライト
- サイドバー削除、中央カラムのフィードレイアウト
- 上部にcompose-box、下に記事カード縦並び
- editingId stateで編集中カードを管理

### Step 5: 旧コンポーネント削除

## 検証

```bash
cd web/cockpit && npx vite build
cd web && make restart
# ブラウザで http://localhost:5173 を確認
```

1. フィードが縦並びで表示される
2. 上部の入力欄から新規記事を投稿できる
3. 記事カードクリックで編集、外クリックでプレビューに戻る
4. CRDTリアルタイム同期が動作する
5. glass-uiスタイルが適用されている
