import { test, expect, type Page } from "@playwright/test";

const API = "http://localhost:5174/api";

// ---------- ヘルパー ----------

async function createEntryViaAPI(): Promise<string> {
  const res = await fetch(`${API}/entries`, { method: "POST" });
  const data = await res.json();
  return data.id as string;
}

async function deleteEntryViaAPI(id: string): Promise<void> {
  await fetch(`${API}/entries/${id}`, { method: "DELETE" });
}

async function deleteAllEntries(): Promise<void> {
  const res = await (await fetch(`${API}/entries`)).json();
  for (const entry of (res.entries as { id: string }[]) || []) {
    await deleteEntryViaAPI(entry.id);
  }
}

/** エントリページに遷移してWSが接続されるまで待つ */
async function navigateToEntry(page: Page, entryId: string): Promise<void> {
  await page.goto(`/entries/${entryId}`);
  // WebSocket 接続を示す緑のドットが出るまで待つ
  await expect(page.locator(".bg-green-400")).toBeVisible({ timeout: 10_000 });
}

/** プレビュー領域をクリックしてtextareaを開き、テキストを入力する */
async function editEntry(page: Page, text: string): Promise<void> {
  // prose-glass が空（高さ0）の場合は dispatchEvent でクリックを発火
  await page.locator(".prose-glass").dispatchEvent("click");
  const textarea = page.locator("textarea");
  await expect(textarea).toBeVisible({ timeout: 5_000 });
  await textarea.fill(text);
  await expect(textarea).toHaveValue(text);
}

/** サーバー側の投影が完了するのを待つ */
async function waitForSync(page: Page): Promise<void> {
  await page.waitForTimeout(1000);
}

// ---------- セットアップ ----------

test.beforeEach(async () => {
  await deleteAllEntries();
});

test.afterEach(async () => {
  await deleteAllEntries();
});

// ---------- TC3: 編集ができる ----------

test("TC3: 編集ができる", async ({ page }) => {
  const entryId = await createEntryViaAPI();
  await navigateToEntry(page, entryId);

  await editEntry(page, "テストテキスト");
  await waitForSync(page);

  // APIで公開版テキストを確認
  const res = await fetch(`${API}/entries/${entryId}`);
  const data = await res.json();
  expect(data.text).toContain("テストテキスト");
});

// ---------- TC4: 編集ができる（別テスト） ----------

test("TC4: 編集ができる（別テスト）", async ({ page }) => {
  const entryId = await createEntryViaAPI();
  await navigateToEntry(page, entryId);

  await editEntry(page, "匿名テキスト");
  await waitForSync(page);

  // APIで公開版テキストを確認
  const res = await fetch(`${API}/entries/${entryId}`);
  const data = await res.json();
  expect(data.text).toContain("匿名テキスト");
});
