import { useRef, useEffect, useCallback } from "preact/hooks";
import { renderMarkdown } from "../lib/markdown";
import { useDocument } from "../hooks/use-document";
import type { EntryListItem } from "../hooks/use-entries";

interface FeedCardProps {
  entry: EntryListItem;
  isEditing: boolean;
  onStartEdit: (id: string) => void;
  onStopEdit: () => void;
}

export function FeedCard({
  entry,
  isEditing,
  onStartEdit,
  onStopEdit,
}: FeedCardProps) {
  const { text, connected, applyTextChange } = useDocument(
    isEditing ? entry.id : null,
  );
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // textarea以外をクリック/タップしたら編集終了
  useEffect(() => {
    if (!isEditing) return;

    const handler = (e: MouseEvent | TouchEvent) => {
      if (
        textareaRef.current &&
        !textareaRef.current.contains(e.target as Node)
      ) {
        onStopEdit();
      }
    };

    document.addEventListener("mousedown", handler);
    document.addEventListener("touchstart", handler);
    return () => {
      document.removeEventListener("mousedown", handler);
      document.removeEventListener("touchstart", handler);
    };
  }, [isEditing, onStopEdit]);

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

  const handleContentClick = useCallback(
    (e: Event) => {
      e.stopPropagation();
      if (!isEditing) {
        onStartEdit(entry.id);
      }
    },
    [isEditing, onStartEdit, entry.id],
  );

  const PREVIEW_LIMIT = 280;
  const date = new Date(entry.created_at).toLocaleDateString("ja-JP");
  const rawContent = isEditing ? text : entry.content;
  const isTruncated = !isEditing && (rawContent?.length ?? 0) > PREVIEW_LIMIT;
  const displayContent = isTruncated
    ? rawContent!.slice(0, PREVIEW_LIMIT)
    : rawContent;

  return (
    <div class="backdrop-blur-xl bg-white/[0.05] border border-white/[0.08] rounded-lg shadow-lg">
      {/* 日付バー */}
      <div class="flex items-center justify-end px-4 sm:px-5 pt-3 pb-1">
        {isEditing && (
          <span
            class={`w-2 h-2 rounded-full mr-2 ${connected ? "bg-green-400" : "bg-red-400"}`}
          />
        )}
        <span class="text-xs text-white/40">{date}</span>
      </div>

      {/* コンテンツ */}
      <div
        onClick={handleContentClick}
        class={`px-4 sm:px-5 pb-4 ${
          !isEditing ? "cursor-pointer active:bg-white/[0.03]" : ""
        }`}
      >
        {isEditing ? (
          <textarea
            ref={textareaRef}
            value={text}
            onInput={handleInput}
            onKeyDown={(e: KeyboardEvent) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                onStopEdit();
              }
            }}
            class="w-full bg-transparent border-none text-base text-white/80 leading-relaxed resize-none focus:outline-none min-h-[80px]"
          />
        ) : (
          <div class="relative">
            <div
              class="prose-glass text-base"
              dangerouslySetInnerHTML={{
                __html: renderMarkdown(displayContent || ""),
              }}
            />
            {isTruncated && (
              <div class="absolute bottom-0 left-0 right-0 h-16 bg-gradient-to-t from-slate-950 to-transparent pointer-events-none" />
            )}
          </div>
        )}
      </div>
    </div>
  );
}
