import { useState, useCallback } from "preact/hooks";
import { useLocation } from "preact-iso";
import { useAuth } from "../hooks/use-auth";

export function LoginPage(_props: { path?: string }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const { login } = useAuth();
  const { route } = useLocation();

  const handleSubmit = useCallback(
    async (e: Event) => {
      e.preventDefault();
      setError("");
      const ok = await login(username, password);
      if (ok) {
        route("/");
      } else {
        setError("認証に失敗しました");
      }
    },
    [username, password, login, route],
  );

  return (
    <div class="max-w-sm mx-auto px-4 py-16">
      <h2 class="font-serif text-xl font-semibold text-ink-text mb-6">ログイン</h2>
      <form onSubmit={handleSubmit} class="space-y-4">
        <div>
          <label class="block font-mono text-[11px] text-ink-muted mb-1">ユーザー名</label>
          <input
            type="text"
            value={username}
            onInput={(e) => setUsername((e.target as HTMLInputElement).value)}
            class="w-full px-3 py-2 bg-ink-surface border border-ink-border rounded text-ink-text font-serif focus:outline-none focus:border-ink-muted"
            autoFocus
          />
        </div>
        <div>
          <label class="block font-mono text-[11px] text-ink-muted mb-1">パスワード</label>
          <input
            type="password"
            value={password}
            onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
            class="w-full px-3 py-2 bg-ink-surface border border-ink-border rounded text-ink-text font-serif focus:outline-none focus:border-ink-muted"
          />
        </div>
        {error && <p class="font-mono text-[11px] text-accent">{error}</p>}
        <button
          type="submit"
          class="w-full py-2 bg-accent-pale hover:bg-accent/20 border border-accent/30 rounded text-accent font-mono text-[11px] tracking-wider transition-colors"
        >
          ログイン
        </button>
      </form>
    </div>
  );
}
