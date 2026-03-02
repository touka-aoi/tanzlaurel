import { useRef, useState, useEffect, useCallback } from "preact/hooks";
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
  const [isExpanded, setIsExpanded] = useState(false);
  const needsConnection = isEditing || isExpanded;
  const { text, connected, applyTextChange } = useDocument(
    needsConnection ? entry.id : null,
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

  const PREVIEW_LINES = 6;
  const contentRef = useRef<HTMLDivElement>(null);
  const [isClamped, setIsClamped] = useState(false);
  const date = new Date(entry.created_at).toLocaleDateString("ja-JP");
  const liveText = isEditing || isExpanded ? text : entry.content;

  useEffect(() => {
    const el = contentRef.current;
    if (!el || isEditing || isExpanded) return;
    setIsClamped(el.scrollHeight > el.clientHeight);
  }, [liveText, isEditing, isExpanded]);

  return (
    <div class="backdrop-blur-xl bg-white/[0.05] border border-white/[0.08] rounded-lg shadow-lg overflow-hidden">
      {/* 日付バー */}
      <div class="flex items-center gap-2 px-4 sm:px-5 pt-3 pb-1">
        {isEditing && (
          <span
            class={`w-2 h-2 rounded-full ${connected ? "bg-green-400" : "bg-red-400"}`}
          />
        )}
        <span class="ml-auto text-xs text-white/40">{date}</span>
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
          <div
            ref={contentRef}
            class={`prose-glass text-base ${!isExpanded ? "overflow-hidden" : ""}`}
            style={!isExpanded ? { display: "-webkit-box", WebkitLineClamp: PREVIEW_LINES, WebkitBoxOrient: "vertical" } : undefined}
            dangerouslySetInnerHTML={{
              __html: renderMarkdown(liveText || ""),
            }}
          />
        )}
      </div>

      {/* 展開/折りたたみ & 個別ページリンク */}
      {!isEditing && (isClamped || isExpanded) && (
        <div class="flex items-center gap-3 px-4 sm:px-5 pb-3 pt-1">
          <button
            type="button"
            class="text-xs text-white/30 hover:text-white/60 transition-colors"
            onClick={(e: Event) => {
              e.stopPropagation();
              setIsExpanded(!isExpanded);
            }}
          >
            {isExpanded ? "折りたたむ" : "もっと読む"}
          </button>
          <a
            href={`/entries/${entry.id}`}
            class="text-xs text-white/30 hover:text-white/60 transition-colors"
            onClick={(e: Event) => e.stopPropagation()}
          >
            個別ページ &rarr;
          </a>
        </div>
      )}
    </div>
  );
}
