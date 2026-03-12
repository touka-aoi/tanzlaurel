import { useState, useCallback, useRef } from "preact/hooks";

interface ComposeBoxProps {
  onSubmit: (text: string) => Promise<void>;
}

export function ComposeBox({ onSubmit }: ComposeBoxProps) {
  const [text, setText] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSubmit = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || submitting) return;

    setSubmitting(true);
    try {
      await onSubmit(trimmed);
      setText("");
      if (textareaRef.current) {
        textareaRef.current.style.height = "auto";
      }
    } finally {
      setSubmitting(false);
    }
  }, [text, submitting, onSubmit]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        handleSubmit();
      }
    },
    [handleSubmit],
  );

  const handleInput = useCallback((e: Event) => {
    const target = e.target as HTMLTextAreaElement;
    setText(target.value);
    target.style.height = "auto";
    target.style.height = target.scrollHeight + "px";
  }, []);

  return (
    <div class="flex flex-col gap-2">
      <textarea
        ref={textareaRef}
        value={text}
        onInput={handleInput}
        onKeyDown={handleKeyDown}
        placeholder="本文を入力..."
        rows={3}
        disabled={submitting}
        class="w-full bg-white/[0.05] border border-white/[0.15] rounded-lg px-4 py-3 text-base text-white/90 placeholder-white/50 resize-none focus:outline-none focus:border-white/30 transition-colors disabled:opacity-50"
      />
      <div class="flex items-center justify-end gap-3">
        <span class="hidden sm:block text-xs text-white/30">
          ⌘/Ctrl+Enter で投稿
        </span>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={!text.trim() || submitting}
          class="px-4 py-1.5 rounded-md text-sm font-medium bg-blue-600 hover:bg-blue-500 disabled:opacity-40 disabled:cursor-not-allowed text-white transition-colors"
        >
          {submitting ? "投稿中..." : "投稿"}
        </button>
      </div>
    </div>
  );
}
