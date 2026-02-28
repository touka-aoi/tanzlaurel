import { useState, useEffect, useRef, useCallback } from "preact/hooks";
import { SyncManager, type SyncState } from "../sync/sync-manager";

export function useDocument(entryId: string | null) {
  const [state, setState] = useState<SyncState>({
    text: "",
    connected: false,
    lastServerSeq: 0,
  });
  const managerRef = useRef<SyncManager | null>(null);

  useEffect(() => {
    if (!entryId) return;

    const siteId = crypto.randomUUID();
    const wsUrl = `${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/ws`;
    const manager = new SyncManager(wsUrl, entryId, siteId);
    managerRef.current = manager;

    const unsub = manager.onChange((s) => setState(s));
    manager.connect();

    return () => {
      unsub();
      manager.disconnect();
      managerRef.current = null;
    };
  }, [entryId]);

  const applyTextChange = useCallback((newText: string) => {
    managerRef.current?.applyTextChange(newText);
  }, []);

  return {
    text: state.text,
    connected: state.connected,
    lastServerSeq: state.lastServerSeq,
    applyTextChange,
  };
}
