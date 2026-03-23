import { useState, useEffect, useCallback } from "preact/hooks";

export interface AuthState {
  authenticated: boolean;
  loading: boolean;
}

export function useAuth() {
  const [state, setState] = useState<AuthState>({
    authenticated: false,
    loading: true,
  });

  const checkAuth = useCallback(async () => {
    try {
      const res = await fetch("/api/auth/status", { credentials: "same-origin" });
      if (res.ok) {
        const data = await res.json();
        setState({ authenticated: !!data.authenticated, loading: false });
      } else {
        setState({ authenticated: false, loading: false });
      }
    } catch {
      setState({ authenticated: false, loading: false });
    }
  }, []);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  const getWsTicket = useCallback(async (): Promise<string | null> => {
    if (!state.authenticated) return null;
    try {
      const res = await fetch("/api/ws-ticket", {
        method: "POST",
        credentials: "same-origin",
      });
      if (res.ok) {
        const data = await res.json();
        return data.ticket ?? null;
      }
    } catch {
      // ignore
    }
    return null;
  }, [state.authenticated]);

  return { ...state, getWsTicket, checkAuth };
}
