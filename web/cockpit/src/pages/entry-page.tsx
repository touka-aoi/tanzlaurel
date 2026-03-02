import { useRoute } from "preact-iso";
import { useState, useRef, useEffect, useCallback } from "preact/hooks";
import { useDocument } from "../hooks/use-document";
import { renderMarkdown } from "../lib/markdown";

export function EntryPage(_props: { path?: string }) {
  const { params } = useRoute();
  const id = params.id as string;
  const { text, connected, applyTextChange } = useDocument(id);
  const [isEditing, setIsEditing] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (isEditing && textareaRef.current) {
      textareaRef.current.focus();
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height =
        textareaRef.current.scrollHeight + "px";
    }
  }, [isEditing, text]);

  const handleInput = useCallback(
    (e: Event) => {
      const target = e.target as HTMLTextAreaElement;
      applyTextChange(target.value);
      target.style.height = "auto";
      target.style.height = target.scrollHeight + "px";
    },
    [applyTextChange],
  );

  const handleContentClick = useCallback(() => {
    if (!isEditing) setIsEditing(true);
  }, [isEditing]);

  const handleStopEdit = useCallback(() => {
    setIsEditing(false);
  }, []);

  // textarea以外をクリックしたら編集終了
  useEffect(() => {
    if (!isEditing) return;
    const handler = (e: MouseEvent | TouchEvent) => {
      if (
        textareaRef.current &&
        !textareaRef.current.contains(e.target as Node)
      ) {
        handleStopEdit();
      }
    };
    document.addEventListener("mousedown", handler);
    document.addEventListener("touchstart", handler);
    return () => {
      document.removeEventListener("mousedown", handler);
      document.removeEventListener("touchstart", handler);
    };
  }, [isEditing, handleStopEdit]);

  return (
    <article class="max-w-2xl mx-auto px-4 py-8">
      {/* ナビゲーション */}
      <nav class="flex items-center gap-3 mb-8">
        <a
          href="/"
          class="text-sm text-white/40 hover:text-white/70 transition-colors"
        >
          &larr; フィードに戻る
        </a>
        <span
          class={`w-2 h-2 rounded-full ${connected ? "bg-green-400" : "bg-red-400"}`}
        />
      </nav>

      {/* 記事本文 */}
      <div onClick={handleContentClick}>
        {isEditing ? (
          <textarea
            ref={textareaRef}
            value={text}
            onInput={handleInput}
            onKeyDown={(e: KeyboardEvent) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                handleStopEdit();
              }
            }}
            class="w-full bg-transparent border-none text-base text-white/80 leading-relaxed resize-none focus:outline-none min-h-[200px]"
          />
        ) : (
          <div
            class="prose-glass text-base cursor-pointer"
            dangerouslySetInnerHTML={{
              __html: renderMarkdown(text || ""),
            }}
          />
        )}
      </div>
    </article>
  );
}
