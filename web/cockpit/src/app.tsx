import { useState, useCallback } from "preact/hooks";
import { useEntries } from "./hooks/use-entries";
import { SyncManager } from "./sync/sync-manager";
import { ComposeBox } from "./components/compose-box";
import { FeedCard } from "./components/feed-card";

export function App() {
  const [editingId, setEditingId] = useState<string | null>(null);
  const { entries, loading, createEntry, refresh } = useEntries();

  const handleSubmit = useCallback(
    async (text: string) => {
      const id = await createEntry();
      // CRDT接続してテキストを送信
      const siteId = crypto.randomUUID();
      const wsUrl = `${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/ws`;
      const manager = new SyncManager(wsUrl, id, siteId);

      await new Promise<void>((resolve) => {
        manager.onChange((s) => {
          if (s.connected) resolve();
        });
        manager.connect();
      });

      manager.applyTextChange(text);

      // ACKを待ってから切断
      await new Promise((r) => setTimeout(r, 300));
      manager.disconnect();
      await refresh();
    },
    [createEntry, refresh],
  );

  const handleStartEdit = useCallback((id: string) => {
    setEditingId(id);
  }, []);

  const handleStopEdit = useCallback(() => {
    setEditingId(null);
    refresh();
  }, [refresh]);

  return (
    <div class="min-h-screen text-white">
      {/* ヘッダー */}
      <header class="sticky top-0 z-10 h-14 flex items-center px-4 border-b border-white/5 bg-slate-950/80 backdrop-blur-xl">
        <h1 class="text-base font-semibold tracking-wide text-white/70">
          Flourish
        </h1>
        <span class="ml-2 text-[10px] px-1.5 py-0.5 rounded-full bg-blue-500/10 text-blue-300/60 border border-blue-400/10">
          CRDT
        </span>
      </header>

      {/* フィード */}
      <main class="max-w-2xl mx-auto px-4 py-6 space-y-4">
        {/* 新規入力欄 */}
        <ComposeBox onSubmit={handleSubmit} />

        {/* 記事一覧 */}
        {loading ? (
          <div class="flex justify-center py-16">
            <div class="text-white/30 text-sm">読み込み中...</div>
          </div>
        ) : entries.length === 0 ? (
          <div class="text-center py-16">
            <p class="text-white/30 text-sm">
              まだ記事がありません。上の入力欄から投稿してみましょう。
            </p>
          </div>
        ) : (
          entries.map((entry) => (
            <FeedCard
              key={entry.id}
              entry={entry}
              isEditing={editingId === entry.id}
              onStartEdit={handleStartEdit}
              onStopEdit={handleStopEdit}
            />
          ))
        )}
      </main>
    </div>
  );
}
