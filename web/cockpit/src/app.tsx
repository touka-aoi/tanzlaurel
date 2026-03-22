import { LocationProvider, Router } from "preact-iso";
import { FeedPage } from "./pages/feed-page";
import { EntryPage } from "./pages/entry-page";
import { LoginPage } from "./pages/login-page";
import { LogoutPage } from "./pages/logout-page";
import { useAuth } from "./hooks/use-auth";
import { useTheme } from "./hooks/use-theme";

function Header() {
  const { authenticated } = useAuth();
  const { theme, toggle } = useTheme();

  return (
    <header class="sticky top-0 z-10 h-14 flex items-center justify-between px-4 border-b border-ink-border bg-ink-bg">
      <a href="/" class="flex items-center gap-2">
        <span class="font-mono text-sm font-bold text-accent tracking-wider">
          Flourish
        </span>
        {authenticated && (
          <span class="w-2 h-2 rounded-full bg-green-400" />
        )}
      </a>
      <button
        type="button"
        onClick={toggle}
        class="font-mono text-[10px] text-ink-muted hover:text-ink-sub transition-colors"
        aria-label="テーマ切替"
      >
        {theme === "dark" ? "light" : "dark"}
      </button>
    </header>
  );
}

export function App() {
  return (
    <LocationProvider>
      <div class="min-h-screen text-ink-text">
        <Header />
        <Router>
          <FeedPage path="/" />
          <EntryPage path="/entries/:id" />
          <LoginPage path="/login" />
          <LogoutPage path="/logout" />
        </Router>
      </div>
    </LocationProvider>
  );
}
