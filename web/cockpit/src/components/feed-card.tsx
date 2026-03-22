import { useRef, useState, useEffect, useCallback } from "preact/hooks";
import { renderMarkdown } from "../lib/markdown";
import { useDocument } from "../hooks/use-document";
import type { EntryListItem } from "../hooks/use-entries";

interface FeedCardProps {
  entry: EntryListItem;
  isEditing: boolean;
  onStartEdit: (id: string) => void;
  onStopEdit: () => void;
  onDelete?: (id: string) => void;
  getWsTicket?: () => Promise<string | null>;
}

export function FeedCard({
  entry,
  isEditing,
  onStartEdit,
  onStopEdit,
  onDelete,
  getWsTicket,
}: FeedCardProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const deleteConfirmRef = useRef<HTMLDivElement>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const needsConnection = isEditing || isExpanded;
  const { text, connected, applyTextChange } = useDocument(
    needsConnection ? entry.id : null,
    getWsTicket,
  );
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const editContainerRef = useRef<HTMLDivElement>(null);

  // 削除確認の外をクリックしたら閉じる
  useEffect(() => {
    if (!showDeleteConfirm) return;
    const handler = (e: MouseEvent) => {
      if (
        deleteConfirmRef.current &&
        !deleteConfirmRef.current.contains(e.target as Node)
      ) {
        setShowDeleteConfirm(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showDeleteConfirm]);

  // textarea以外をクリック/タップしたら編集終了
  useEffect(() => {
    if (!isEditing) return;

    const handler = (e: MouseEvent | TouchEvent) => {
      if (
        editContainerRef.current &&
        !editContainerRef.current.contains(e.target as Node)
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
    }
  }, [isEditing]);

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
        {onDelete && (
          <button
            type="button"
            class="text-[10px] leading-none text-red-400/50 hover:text-red-400 transition-colors border border-red-400/20 hover:border-red-400/50 rounded px-1 py-0.5"
            onClick={(e: Event) => {
              e.stopPropagation();
              setShowDeleteConfirm(true);
            }}
          >
            削除
          </button>
        )}
      </div>

      {/* 削除確認 */}
      {showDeleteConfirm && onDelete && (
        <div ref={deleteConfirmRef} class="px-4 sm:px-5 py-2 border-y border-red-400/10 bg-red-500/[0.03]">
          <p class="text-xs text-white/60 mb-2">この記事を削除しますか？</p>
          <div class="flex justify-end gap-3">
            <button
              type="button"
              class="text-xs px-3 py-1 text-white/50 hover:text-white/80 transition-colors"
              onClick={() => setShowDeleteConfirm(false)}
            >
              キャンセル
            </button>
            <button
              type="button"
              class="text-xs px-3 py-1 bg-red-500/20 hover:bg-red-500/30 border border-red-400/20 rounded text-red-300 transition-colors"
              onClick={() => {
                setShowDeleteConfirm(false);
                onDelete(entry.id);
              }}
            >
              削除する
            </button>
          </div>
        </div>
      )}

      {/* コンテンツ */}
      <div
        onClick={handleContentClick}
        class={`px-4 sm:px-5 pb-4 ${
          !isEditing ? "cursor-pointer active:bg-white/[0.03]" : ""
        }`}
      >
        {isEditing ? (
          <div ref={editContainerRef}>
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
            <div class="flex items-center justify-end border-t border-white/[0.15] px-2 py-1">
              <button
                type="button"
                onClick={(e: Event) => {
                  e.stopPropagation();
                  onStopEdit();
                }}
                class="flex items-center justify-center w-8 h-8 rounded text-white/50 hover:text-white/80 transition-colors cursor-pointer"
                aria-label="編集を完了する"
              >
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 20 20" fill="currentColor" class="w-4 h-4">
                  <path fill-rule="evenodd" d="M16.704 4.153a.75.75 0 0 1 .143 1.052l-8 10.5a.75.75 0 0 1-1.127.075l-4.5-4.5a.75.75 0 0 1 1.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 0 1 1.05-.143Z" clip-rule="evenodd" />
                </svg>
              </button>
            </div>
          </div>
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
