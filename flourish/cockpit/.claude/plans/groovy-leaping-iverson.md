# E2Eスモークテスト (Playwright Test)

## Context
CRDTブログのフロントエンド（Preact + WebSocket同期）が実装済み。UIの基本操作が正しく動作することを冪等なE2Eテストで担保する。

## ユースケース
1. 記事の作成ができる
2. 記事の編集ができる (a. 文字入力, b. 文字削除)
3. 編集後閲覧できる (Markdownプレビュー)
4. 記事の削除ができる

## ステップ

### Step 1: playwright.config.ts 作成
- `web/cockpit/playwright.config.ts` を作成
- baseURL: `http://localhost:5173`
- webServer設定は不要（手動起動前提）
- テストディレクトリ: `e2e/`

### Step 2: E2Eテストファイル作成
- `web/cockpit/e2e/entry-crud.spec.ts` に全UCをまとめて記述
- 各テストは冪等: テスト内で作成→操作→削除（APIでクリーンアップ）
- テスト間の依存なし

```
test('UC1: 記事の作成ができる')
test('UC2a: 文字の入力ができる')
test('UC2b: 文字の削除ができる')
test('UC3: 編集後Markdownプレビューで閲覧できる')
test('UC4: 記事の削除ができる')
```

### Step 3: テスト実行 + 修正
- `npx playwright test` で実行
- 失敗するテストがあれば原因を調査し修正

## 対象ファイル
- 新規: `web/cockpit/playwright.config.ts`
- 新規: `web/cockpit/e2e/entry-crud.spec.ts`

## 検証
```bash
cd web/cockpit && npx playwright test
```
