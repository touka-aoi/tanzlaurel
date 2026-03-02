import { useState, useEffect, useRef, useCallback } from "preact/hooks";
import { SyncManager, type SyncState } from "../sync/sync-manager";

export function useDocument(entryId: string | null, getWsTicket?: () => Promise<string | null>) {
  const [state, setState] = useState<SyncState>({
    text: "",
    connected: false,
    lastServerSeq: 0,
    authenticated: false,
  });
  const managerRef = useRef<SyncManager | null>(null);

  useEffect(() => {
    if (!entryId) return;

    let cancelled = false;

    (async () => {
      const ticket = getWsTicket ? await getWsTicket() : null;
      if (cancelled) return;

      const siteId = crypto.randomUUID();
      const baseUrl = `${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/api/ws`;
      const wsUrl = ticket ? `${baseUrl}?ticket=${encodeURIComponent(ticket)}` : baseUrl;
      const manager = new SyncManager(wsUrl, entryId, siteId);
      managerRef.current = manager;

      manager.onChange((s) => setState(s));
      manager.connect();
    })();

    return () => {
      cancelled = true;
      managerRef.current?.disconnect();
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
    authenticated: state.authenticated,
    applyTextChange,
  };
}
