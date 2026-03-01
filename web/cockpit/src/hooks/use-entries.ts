import { useState, useEffect, useCallback, useRef } from "preact/hooks";

export interface EntryListItem {
  id: string;
  title: string;
  content: string;
  thumbnail: string | null;
  created_at: string;
  updated_at: string;
}

export function useEntries() {
  const [entries, setEntries] = useState<EntryListItem[]>([]);
  const [loading, setLoading] = useState(true);

  const initializedRef = useRef(false);

  const fetchEntries = useCallback(async () => {
    if (!initializedRef.current) setLoading(true);
    try {
      const res = await fetch("/api/entries");
      const data = await res.json();
      const list: EntryListItem[] = data.entries || [];
      list.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
      setEntries(list);
    } catch (err) {
      console.error("Failed to fetch entries:", err);
    } finally {
      setLoading(false);
      initializedRef.current = true;
    }
  }, []);

  useEffect(() => {
    fetchEntries();
  }, [fetchEntries]);

  const createEntry = useCallback(async () => {
    const res = await fetch("/api/entries", { method: "POST" });
    const data = await res.json();
    await fetchEntries();
    return data.id as string;
  }, [fetchEntries]);

  const deleteEntry = useCallback(
    async (id: string) => {
      await fetch(`/api/entries/${id}`, { method: "DELETE" });
      await fetchEntries();
    },
    [fetchEntries],
  );

  return { entries, loading, createEntry, deleteEntry, refresh: fetchEntries };
}
