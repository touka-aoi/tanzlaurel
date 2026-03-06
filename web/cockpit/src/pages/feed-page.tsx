import { useState, useCallback } from "preact/hooks";
import { useEntries } from "../hooks/use-entries";
import { useAuth } from "../hooks/use-auth";
import { SyncManager } from "../sync/sync-manager";
import { ComposeBox } from "../components/compose-box";
import { FeedCard } from "../components/feed-card";

export function FeedPage(_props: { path?: string }) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const { entries, loading, createEntry, deleteEntry, refresh } = useEntries();
  const { authenticated, getWsTicket } = useAuth();

  const handleSubmit = useCallback(
    async (text: string) => {
      const id = await createEntry();
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
    <main class="max-w-2xl mx-auto px-4 py-6 space-y-4">
      <ComposeBox onSubmit={handleSubmit} />

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
            onDelete={authenticated ? deleteEntry : undefined}
            getWsTicket={authenticated ? getWsTicket : undefined}
          />
        ))
      )}
    </main>
  );
}
