import { useState, useEffect, useCallback } from "preact/hooks";

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

  const fetchEntries = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fetch("/api/entries");
      const data = await res.json();
      setEntries(data.entries || []);
    } catch (err) {
      console.error("Failed to fetch entries:", err);
    } finally {
      setLoading(false);
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
