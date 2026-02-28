import type { EntryListItem } from "../hooks/use-entries";

interface Props {
  entries: EntryListItem[];
  loading: boolean;
  onSelect: (id: string) => void;
  onCreate: () => void;
  onDelete: (id: string) => void;
}

export function EntryList({
  entries,
  loading,
  onSelect,
  onCreate,
  onDelete,
}: Props) {
  return (
    <div class="h-full flex flex-col">
      <div class="flex items-center justify-between p-4 border-b border-white/10">
        <h2 class="text-lg font-semibold text-white/90">Entries</h2>
        <button
          onClick={onCreate}
          class="px-3 py-1.5 rounded-lg bg-blue-500/20 text-blue-300 hover:bg-blue-500/30 backdrop-blur-sm border border-blue-400/20 transition-all text-sm"
        >
          + New
        </button>
      </div>

      <div class="flex-1 overflow-y-auto p-2 space-y-1">
        {loading && (
          <div class="text-center text-white/40 py-8">Loading...</div>
        )}
        {!loading && entries.length === 0 && (
          <div class="text-center text-white/40 py-8">No entries yet</div>
        )}
        {entries.map((entry) => (
          <div
            key={entry.id}
            class="group flex items-center justify-between p-3 rounded-xl bg-white/5 hover:bg-white/10 backdrop-blur-sm border border-white/5 hover:border-white/10 cursor-pointer transition-all"
            onClick={() => onSelect(entry.id)}
          >
            <div class="min-w-0 flex-1">
              <div class="text-sm font-medium text-white/80 truncate">
                {entry.title || "Untitled"}
              </div>
              <div class="text-xs text-white/40 truncate mt-0.5">
                {entry.content || "Empty"}
              </div>
            </div>
            <button
              onClick={(e) => {
                e.stopPropagation();
                onDelete(entry.id);
              }}
              class="opacity-0 group-hover:opacity-100 ml-2 px-2 py-1 rounded-md text-xs text-red-300/60 hover:text-red-300 hover:bg-red-500/10 transition-all"
            >
              Delete
            </button>
          </div>
        ))}
      </div>
    </div>
  );
}
