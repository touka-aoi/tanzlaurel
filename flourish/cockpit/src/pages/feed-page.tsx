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
    <main class="texture-overlay min-h-screen">
      <div class="relative z-2 max-w-2xl mx-auto px-4 py-6">
        <ComposeBox onSubmit={handleSubmit} />

        {loading ? (
          <div class="flex justify-center py-16">
            <div class="font-mono text-[11px] text-ink-muted">読み込み中...</div>
          </div>
        ) : entries.length === 0 ? (
          <div class="text-center py-16">
            <p class="font-mono text-[11px] text-ink-muted">
              まだ記事がありません。上の入力欄から投稿してみましょう。
            </p>
          </div>
        ) : (
          <div class="mt-6">
            {entries.map((entry) => (
              <FeedCard
                key={entry.id}
                entry={entry}
                isEditing={editingId === entry.id}
                onStartEdit={handleStartEdit}
                onStopEdit={handleStopEdit}
                onDelete={authenticated ? deleteEntry : undefined}
                getWsTicket={authenticated ? getWsTicket : undefined}
              />
            ))}
          </div>
        )}
      </div>
    </main>
  );
}
