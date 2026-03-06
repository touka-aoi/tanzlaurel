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

  const login = useCallback(async (username: string, password: string): Promise<boolean> => {
    const res = await fetch("/api/login", {
      method: "POST",
      credentials: "same-origin",
      headers: {
        Authorization: "Basic " + btoa(`${username}:${password}`),
      },
    });
    if (res.ok) {
      setState({ authenticated: true, loading: false });
      return true;
    }
    return false;
  }, []);

  const logout = useCallback(async () => {
    await fetch("/api/logout", {
      method: "POST",
      credentials: "same-origin",
    });
    setState({ authenticated: false, loading: false });
  }, []);

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

  return { ...state, login, logout, getWsTicket, checkAuth };
}
