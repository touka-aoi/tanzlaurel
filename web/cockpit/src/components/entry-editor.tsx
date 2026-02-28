import { useRef, useCallback } from "preact/hooks";

interface Props {
  text: string;
  connected: boolean;
  onInput: (text: string) => void;
}

export function EntryEditor({ text, connected, onInput }: Props) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleInput = useCallback(
    (e: Event) => {
      const target = e.target as HTMLTextAreaElement;
      onInput(target.value);
    },
    [onInput],
  );

  return (
    <div class="h-full flex flex-col">
      <div class="flex items-center gap-2 px-4 py-2 border-b border-white/10">
        <div
          class={`w-2 h-2 rounded-full ${connected ? "bg-emerald-400 shadow-[0_0_6px_rgba(52,211,153,0.5)]" : "bg-red-400 shadow-[0_0_6px_rgba(248,113,113,0.5)]"}`}
        />
        <span class="text-xs text-white/40">
          {connected ? "Connected" : "Disconnected"}
        </span>
      </div>
      <textarea
        ref={textareaRef}
        value={text}
        onInput={handleInput}
        disabled={!connected}
        placeholder={
          connected ? "Start writing..." : "Connecting to server..."
        }
        class="flex-1 w-full p-4 bg-transparent text-white/90 placeholder-white/20 resize-none outline-none font-mono text-sm leading-relaxed disabled:opacity-50"
      />
    </div>
  );
}
