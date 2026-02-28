import { useState, useCallback } from "preact/hooks";
import { useEntries } from "./hooks/use-entries";
import { useDocument } from "./hooks/use-document";
import { EntryList } from "./components/entry-list";
import { EntryEditor } from "./components/entry-editor";
import { MarkdownPreview } from "./components/markdown-preview";

export function App() {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const { entries, loading, createEntry, deleteEntry } = useEntries();
  const { text, connected } = useDocument(selectedId);

  const handleCreate = useCallback(async () => {
    const id = await createEntry();
    setSelectedId(id);
  }, [createEntry]);

  const handleDelete = useCallback(
    async (id: string) => {
      await deleteEntry(id);
      if (selectedId === id) setSelectedId(null);
    },
    [deleteEntry, selectedId],
  );

  const handleInput = useCallback((_newText: string) => {
    // TODO: diff old text vs new text and generate insert/delete ops
  }, []);

  return (
    <div class="min-h-screen bg-gradient-to-br from-slate-950 via-blue-950/30 to-slate-950 text-white">
      <header class="h-12 flex items-center px-4 border-b border-white/5 bg-white/[0.02] backdrop-blur-xl">
        <h1 class="text-sm font-semibold tracking-wide text-white/70">
          Flourish
        </h1>
        <span class="ml-2 text-[10px] px-1.5 py-0.5 rounded-full bg-blue-500/10 text-blue-300/60 border border-blue-400/10">
          CRDT
        </span>
      </header>

      <div class="flex h-[calc(100vh-3rem)]">
        {/* Sidebar */}
        <div class="w-64 shrink-0 border-r border-white/5 bg-white/[0.01]">
          <EntryList
            entries={entries}
            loading={loading}
            onSelect={setSelectedId}
            onCreate={handleCreate}
            onDelete={handleDelete}
          />
        </div>

        {/* Main content */}
        {selectedId ? (
          <div class="flex-1 flex">
            {/* Editor */}
            <div class="flex-1 border-r border-white/5">
              <EntryEditor
                text={text}
                connected={connected}
                onInput={handleInput}
              />
            </div>
            {/* Preview */}
            <div class="flex-1 bg-white/[0.01]">
              <MarkdownPreview text={text} />
            </div>
          </div>
        ) : (
          <div class="flex-1 flex items-center justify-center">
            <div class="text-center">
              <div class="text-4xl mb-4 opacity-20">📝</div>
              <p class="text-white/30 text-sm">
                Select an entry or create a new one
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
