import { test, expect } from "@playwright/test";

const API = "http://localhost:5173/api";

/** POST /api/entries して ID を返す */
async function createEntryViaAPI(): Promise<string> {
  const res = await fetch(`${API}/entries`, { method: "POST" });
  const data = await res.json();
  return data.id as string;
}

/** DELETE /api/entries/:id */
async function deleteEntryViaAPI(id: string): Promise<void> {
  await fetch(`${API}/entries/${id}`, { method: "DELETE" });
}

/** 全エントリを削除 */
async function deleteAllEntries(): Promise<void> {
  const res = await (await fetch(`${API}/entries`)).json();
  for (const entry of (res.entries as { id: string }[]) || []) {
    await deleteEntryViaAPI(entry.id);
  }
}

test.beforeEach(async () => {
  await deleteAllEntries();
});

test.afterEach(async () => {
  await deleteAllEntries();
});

// ---------- UC1: 記事の作成ができる ----------

test("UC1: 記事の作成ができる", async ({ page }) => {
  await page.goto("/");

  // "+ New" ボタンをクリック
  await page.getByRole("button", { name: "+ New" }).click();

  // サイドバーに "Untitled" エントリが現れる
  const entry = page.locator("text=Untitled").first();
  await expect(entry).toBeVisible();

  // エディタ（textarea）が表示される
  const textarea = page.locator("textarea");
  await expect(textarea).toBeVisible();
});

// ---------- UC2a: 文字の入力ができる ----------

test("UC2a: 文字の入力ができる", async ({ page }) => {
  await createEntryViaAPI();
  await page.goto("/");

  // エントリを選択
  await page.locator("text=Untitled").first().click();

  // textarea が有効になるまで待つ（WebSocket 接続）
  const textarea = page.locator("textarea");
  await expect(textarea).toBeEnabled({ timeout: 10_000 });

  // 文字を入力
  await textarea.fill("Hello CRDT");
  await expect(textarea).toHaveValue("Hello CRDT");
});

// ---------- UC2b: 文字の削除ができる ----------

test("UC2b: 文字の削除ができる", async ({ page }) => {
  await createEntryViaAPI();
  await page.goto("/");
  await page.locator("text=Untitled").first().click();

  const textarea = page.locator("textarea");
  await expect(textarea).toBeEnabled({ timeout: 10_000 });

  // テキストを入力
  await textarea.fill("abcdef");
  await expect(textarea).toHaveValue("abcdef");

  // 全選択して削除
  await textarea.selectText();
  await textarea.press("Backspace");
  await expect(textarea).toHaveValue("");
});

// ---------- UC3: 編集後 Markdown プレビューで閲覧できる ----------

test("UC3: 編集後Markdownプレビューで閲覧できる", async ({ page }) => {
  await createEntryViaAPI();
  await page.goto("/");
  await page.locator("text=Untitled").first().click();

  const textarea = page.locator("textarea");
  await expect(textarea).toBeEnabled({ timeout: 10_000 });

  // Markdown を入力
  await textarea.fill("# Hello");

  // サーバーへの同期を待つ
  await page.waitForTimeout(500);

  // ページをリロードしてエントリを再選択
  await page.reload();
  await page.locator("text=Untitled").first().click();

  // WebSocket接続 → sync でCRDT内容が復元される
  const reloadedTextarea = page.locator("textarea");
  await expect(reloadedTextarea).toBeEnabled({ timeout: 10_000 });

  // プレビュー領域に h1 が表示される（永続化された内容の閲覧）
  const preview = page.locator(".prose");
  await expect(preview.locator("h1")).toHaveText("Hello", { timeout: 10_000 });
});

// ---------- UC4: 記事の削除ができる ----------

test("UC4: 記事の削除ができる", async ({ page }) => {
  const id = await createEntryViaAPI();
  await page.goto("/");

  // エントリが表示されていることを確認
  const entry = page.locator("text=Untitled").first();
  await expect(entry).toBeVisible();

  // ホバーして Delete ボタンを表示→クリック
  await entry.hover();
  await page.getByRole("button", { name: "Delete" }).first().click();

  // API で存在しないことを確認
  const res = await (await fetch(`${API}/entries`)).json();
  const still = (res.entries as { id: string }[]).find((e) => e.id === id);
  expect(still).toBeUndefined();
});
