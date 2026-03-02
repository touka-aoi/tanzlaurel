import { LocationProvider, Router } from "preact-iso";
import { FeedPage } from "./pages/feed-page";
import { EntryPage } from "./pages/entry-page";

export function App() {
  return (
    <LocationProvider>
      <div class="min-h-screen text-white">
        {/* ヘッダー */}
        <header class="sticky top-0 z-10 h-14 flex items-center px-4 border-b border-white/5 bg-slate-950/80 backdrop-blur-xl">
          <a href="/" class="flex items-center gap-2">
            <h1 class="text-base font-semibold tracking-wide text-white/70">
              Flourish
            </h1>
            <span class="text-[10px] px-1.5 py-0.5 rounded-full bg-blue-500/10 text-blue-300/60 border border-blue-400/10">
              CRDT
            </span>
          </a>
        </header>

        {/* ルーティング */}
        <Router>
          <FeedPage path="/" />
          <EntryPage path="/entries/:id" />
        </Router>
      </div>
    </LocationProvider>
  );
}
