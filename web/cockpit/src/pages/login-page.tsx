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
      <h2 class="text-xl font-semibold text-white/80 mb-6">ログイン</h2>
      <form onSubmit={handleSubmit} class="space-y-4">
        <div>
          <label class="block text-sm text-white/50 mb-1">ユーザー名</label>
          <input
            type="text"
            value={username}
            onInput={(e) => setUsername((e.target as HTMLInputElement).value)}
            class="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-white/80 focus:outline-none focus:border-blue-400/50"
            autoFocus
          />
        </div>
        <div>
          <label class="block text-sm text-white/50 mb-1">パスワード</label>
          <input
            type="password"
            value={password}
            onInput={(e) => setPassword((e.target as HTMLInputElement).value)}
            class="w-full px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-white/80 focus:outline-none focus:border-blue-400/50"
          />
        </div>
        {error && <p class="text-red-400 text-sm">{error}</p>}
        <button
          type="submit"
          class="w-full py-2 bg-blue-500/20 hover:bg-blue-500/30 border border-blue-400/20 rounded-lg text-blue-300 text-sm transition-colors"
        >
          ログイン
        </button>
      </form>
    </div>
  );
}
