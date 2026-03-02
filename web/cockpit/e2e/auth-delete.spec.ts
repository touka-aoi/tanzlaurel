import { test, expect, type Page } from "@playwright/test";

const API = "http://localhost:5174/api";
const ADMIN_USER = "admin";
const ADMIN_PASSWORD = "pass";

// ---------- ヘルパー ----------

async function createEntryViaAPI(): Promise<string> {
  const res = await fetch(`${API}/entries`, { method: "POST" });
  const data = await res.json();
  return data.id as string;
}

async function loginViaAPI(): Promise<string> {
  const res = await fetch(`${API}/login`, {
    method: "POST",
    headers: {
      Authorization: "Basic " + btoa(`${ADMIN_USER}:${ADMIN_PASSWORD}`),
    },
  });
  const cookie = res.headers.get("set-cookie") ?? "";
  const match = cookie.match(/token=([^;]+)/);
  return match?.[1] ?? "";
}

async function deleteAllEntriesAsAdmin(): Promise<void> {
  const token = await loginViaAPI();
  const res = await (
    await fetch(`${API}/entries`, {
      headers: { Cookie: `token=${token}` },
    })
  ).json();
  for (const entry of (res.entries as { id: string }[]) || []) {
    await fetch(`${API}/entries/${entry.id}`, {
      method: "DELETE",
      headers: { Cookie: `token=${token}` },
    });
  }
}

async function getEntryText(entryId: string): Promise<string> {
  const res = await fetch(`${API}/entries/${entryId}`);
  const data = await res.json();
  return data.text ?? "";
}

/** ログインページからログインする */
async function loginViaUI(page: Page): Promise<void> {
  await page.goto("/login");
  const inputs = page.locator("input");
  await inputs.nth(0).fill(ADMIN_USER);
  await inputs.nth(1).fill(ADMIN_PASSWORD);
  await page.getByRole("button", { name: "ログイン" }).click();
  // フィードページにリダイレクトされるのを待つ
  await page.waitForURL("/", { timeout: 5_000 });
}

/**
 * EntryPageでテキストを入力する。
 * クリックで編集モードに入り、WS接続後にテキストを入力する。
 */
async function typeOnEntryPage(
  page: Page,
  entryId: string,
  text: string,
): Promise<void> {
  await page.goto(`/entries/${entryId}`);
  // コンテンツ領域をクリックして編集モードに入る
  await page.locator(".prose-glass").dispatchEvent("click");
  const textarea = page.locator("textarea");
  await expect(textarea).toBeVisible({ timeout: 5_000 });
  // WS接続インジケーター（緑ドット）を待つ
  await expect(page.locator("article .bg-green-400")).toBeVisible({ timeout: 10_000 });
  // type で1文字ずつ入力（onInput が発火して applyTextChange が呼ばれる）
  await textarea.type(text, { delay: 50 });
}

/**
 * EntryPageで全テキストを削除する。
 * Ctrl+A → Backspace で全選択して削除する。
 */
async function clearOnEntryPage(
  page: Page,
  entryId: string,
): Promise<void> {
  await page.goto(`/entries/${entryId}`);
  // コンテンツ領域をクリックして編集モードに入る
  await page.locator(".prose-glass").dispatchEvent("click");
  const textarea = page.locator("textarea");
  await expect(textarea).toBeVisible({ timeout: 5_000 });
  // WS接続インジケーター（緑ドット）を待つ
  await expect(page.locator("article .bg-green-400")).toBeVisible({ timeout: 10_000 });
  // 全選択して削除
  const mod = process.platform === "darwin" ? "Meta" : "Control";
  await textarea.press(`${mod}+a`);
  await textarea.press("Backspace");
}

/** サーバー同期を待つ */
async function waitForSync(page: Page): Promise<void> {
  await page.waitForTimeout(1500);
}

// ---------- セットアップ ----------

test.beforeEach(async () => {
  await deleteAllEntriesAsAdmin();
});

test.afterEach(async () => {
  await deleteAllEntriesAsAdmin();
});

// ---------- テストケース ----------

test("認証ユーザーの文字は非認証ユーザーで削除できない", async ({
  browser,
}) => {
  const entryId = await createEntryViaAPI();

  // 認証ユーザーがテキストを入力
  const authContext = await browser.newContext();
  const authPage = await authContext.newPage();
  await loginViaUI(authPage);
  await typeOnEntryPage(authPage, entryId, "認証テキスト");
  await waitForSync(authPage);
  await authContext.close();

  // テキストが保存されたことを確認
  const textAfterAuth = await getEntryText(entryId);
  expect(textAfterAuth).toContain("認証テキスト");

  // 非認証ユーザーが全文削除を試みる
  const anonContext = await browser.newContext();
  const anonPage = await anonContext.newPage();
  await clearOnEntryPage(anonPage, entryId);
  await waitForSync(anonPage);
  await anonContext.close();

  // テキストが残っていることを確認
  const textAfterAnon = await getEntryText(entryId);
  expect(textAfterAnon).toContain("認証テキスト");
});

test("非認証ユーザーの文字は非認証ユーザーで削除できる", async ({
  browser,
}) => {
  const entryId = await createEntryViaAPI();

  // 非認証ユーザーがテキストを入力
  const ctx1 = await browser.newContext();
  const page1 = await ctx1.newPage();
  await typeOnEntryPage(page1, entryId, "匿名テキスト");
  await waitForSync(page1);
  await ctx1.close();

  // テキストが保存されたことを確認
  const textBefore = await getEntryText(entryId);
  expect(textBefore).toContain("匿名テキスト");

  // 別の非認証ユーザーが全文削除
  const ctx2 = await browser.newContext();
  const page2 = await ctx2.newPage();
  await clearOnEntryPage(page2, entryId);
  await waitForSync(page2);
  await ctx2.close();

  // テキストが空になっていることを確認
  const textAfter = await getEntryText(entryId);
  expect(textAfter).toBe("");
});

test("認証ユーザーは非認証ユーザーの文字を削除できる", async ({
  browser,
}) => {
  const entryId = await createEntryViaAPI();

  // 非認証ユーザーがテキストを入力
  const anonContext = await browser.newContext();
  const anonPage = await anonContext.newPage();
  await typeOnEntryPage(anonPage, entryId, "匿名テキスト");
  await waitForSync(anonPage);
  await anonContext.close();

  // テキストが保存されたことを確認
  const textBefore = await getEntryText(entryId);
  expect(textBefore).toContain("匿名テキスト");

  // 認証ユーザーが全文削除
  const authContext = await browser.newContext();
  const authPage = await authContext.newPage();
  await loginViaUI(authPage);
  await authPage.goto(`/entries/${entryId}`);
  await authPage.locator(".prose-glass").dispatchEvent("click");
  const authTextarea = authPage.locator("textarea");
  await expect(authTextarea).toBeVisible({ timeout: 5_000 });
  await expect(authPage.locator("article .bg-green-400")).toBeVisible({ timeout: 10_000 });
  // sync完了を待ってテキストが読み込まれることを確認
  await expect(authTextarea).not.toHaveValue("", { timeout: 10_000 });
  // 全選択して削除
  const mod = process.platform === "darwin" ? "Meta" : "Control";
  await authTextarea.press(`${mod}+a`);
  await authTextarea.press("Backspace");
  // テキストが空になったことをUIで確認
  await expect(authTextarea).toHaveValue("", { timeout: 5_000 });
  await waitForSync(authPage);
  await authContext.close();

  // テキストが空になっていることを確認
  const textAfter = await getEntryText(entryId);
  expect(textAfter).toBe("");
});

test("記事削除時に確認モーダルが表示される", async ({ browser }) => {
  const entryId = await createEntryViaAPI();

  // 認証ユーザーでログイン
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await loginViaUI(page);
  await page.goto("/");

  // 削除ボタンが表示されるまで待つ
  await expect(page.locator("button:text('削除')").first()).toBeVisible({
    timeout: 5_000,
  });

  // 削除ボタンをクリック → モーダルが表示される
  await page.locator("button:text('削除')").first().click();
  await expect(page.locator("text=この記事を削除しますか？")).toBeVisible();

  // キャンセル → モーダルが閉じてエントリは残る
  await page.locator("button:text('キャンセル')").click();
  await expect(page.locator("text=この記事を削除しますか？")).not.toBeVisible();

  const res1 = await fetch(`${API}/entries`);
  const data1 = await res1.json();
  expect(
    (data1.entries as { id: string }[]).some((e) => e.id === entryId),
  ).toBe(true);

  // 再度削除ボタン → モーダル → 「削除する」をクリック
  await page.locator("button:text('削除')").first().click();
  await expect(page.locator("text=この記事を削除しますか？")).toBeVisible();
  await page.locator("button:text('削除する')").click();
  await page.waitForTimeout(1000);

  // エントリが削除されている
  const res2 = await fetch(`${API}/entries`);
  const data2 = await res2.json();
  expect(
    (data2.entries as { id: string }[]).some((e) => e.id === entryId),
  ).toBe(false);

  await ctx.close();
});
