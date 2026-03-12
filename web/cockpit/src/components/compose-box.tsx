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

  const canSubmit = text.trim().length > 0 && !submitting;

  return (
    <div class="w-full bg-white/[0.05] border border-white/[0.15] rounded-lg overflow-hidden focus-within:border-white/30 transition-colors">
      <textarea
        ref={textareaRef}
        value={text}
        onInput={handleInput}
        onKeyDown={handleKeyDown}
        placeholder="本文を入力..."
        rows={3}
        disabled={submitting}
        class="w-full bg-transparent px-4 py-3 text-base text-white/90 placeholder-white/50 resize-none focus:outline-none disabled:opacity-50"
      />
      <div class="flex items-center justify-end border-t border-white/[0.15] px-2 py-1">
        <button
          type="button"
          onClick={handleSubmit}
          disabled={!canSubmit}
          class="flex items-center justify-center w-8 h-8 rounded text-white/50 hover:text-white/80 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
          aria-label="投稿する"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="w-4 h-4"
          >
            <path d="M22 2L11 13" />
            <path d="M22 2L15 22L11 13L2 9L22 2Z" />
          </svg>
        </button>
      </div>
    </div>
  );
}
