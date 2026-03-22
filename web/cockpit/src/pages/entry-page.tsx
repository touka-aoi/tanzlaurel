import { useRoute } from "preact-iso";
import { useState, useRef, useEffect, useCallback } from "preact/hooks";
import { useDocument } from "../hooks/use-document";
import { useAuth } from "../hooks/use-auth";
import { renderMarkdown } from "../lib/markdown";

interface EntryDetail {
  id: string;
  title: string;
  content: string;
  text: string;
  created_at: string;
  updated_at: string;
}

export function EntryPage(_props: { path?: string }) {
  const { params } = useRoute();
  const id = params.id as string;
  const [isEditing, setIsEditing] = useState(false);
  const [entry, setEntry] = useState<EntryDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const editContainerRef = useRef<HTMLDivElement>(null);
  const { getWsTicket } = useAuth();

  // 編集モード時のみWS接続
  const { text, connected, applyTextChange } = useDocument(
    isEditing ? id : null,
    getWsTicket,
  );

  // 閲覧用: APIからエントリ取得
  useEffect(() => {
    setLoading(true);
    fetch(`/api/entries/${id}`)
      .then((res) => res.json())
      .then((data) => setEntry(data))
      .finally(() => setLoading(false));
  }, [id]);

  // 編集終了時にAPIから再取得
  const handleStopEdit = useCallback(() => {
    setIsEditing(false);
    fetch(`/api/entries/${id}`)
      .then((res) => res.json())
      .then((data) => setEntry(data));
  }, [id]);

  useEffect(() => {
    if (isEditing && textareaRef.current) {
      textareaRef.current.focus();
    }
  }, [isEditing]);

  // リモート変更時のみtextareaを同期（ローカル入力時はDOMが既に正しい）
  useEffect(() => {
    if (textareaRef.current && textareaRef.current.value !== text) {
      const { selectionStart, selectionEnd } = textareaRef.current;
      textareaRef.current.value = text;
      textareaRef.current.selectionStart = selectionStart;
      textareaRef.current.selectionEnd = selectionEnd;
      textareaRef.current.style.height = "auto";
      textareaRef.current.style.height = textareaRef.current.scrollHeight + "px";
    }
  }, [text]);

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

  // textarea以外をクリックしたら編集終了
  useEffect(() => {
    if (!isEditing) return;
    const handler = (e: MouseEvent | TouchEvent) => {
      if (
        editContainerRef.current &&
        !editContainerRef.current.contains(e.target as Node)
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

  // 表示テキスト: 編集中はWS経由、閲覧中はAPI取得の全文(text)
  const displayContent = isEditing ? text : (entry?.text ?? "");

  if (loading) {
    return (
      <div class="flex justify-center py-16">
        <div class="font-mono text-[11px] text-ink-muted">読み込み中...</div>
      </div>
    );
  }

  return (
    <article class="max-w-2xl mx-auto px-4 py-8">
      {/* ナビゲーション */}
      <nav class="flex items-center gap-3 mb-8">
        <a
          href="/"
          class="font-mono text-[10px] text-ink-muted hover:text-ink-sub transition-colors"
        >
          &larr; フィードに戻る
        </a>
        {isEditing && (
          <span
            class={`w-2 h-2 rounded-full ${connected ? "bg-green-400" : "bg-red-400"}`}
          />
        )}
      </nav>

      {/* 記事本文 */}
      <div onClick={handleContentClick}>
        {isEditing ? (
          <div ref={editContainerRef} class="border border-ink-border rounded p-3">
            <textarea
              ref={textareaRef}
              onInput={handleInput}
              onKeyDown={(e: KeyboardEvent) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault();
                  handleStopEdit();
                }
              }}
              class="w-full bg-transparent border-none text-base text-ink-text font-serif leading-relaxed resize-none focus:outline-none min-h-[200px]"
            />
            <div class="flex items-center justify-end border-t border-ink-border px-2 py-1">
              <button
                type="button"
                onClick={(e: Event) => {
                  e.stopPropagation();
                  handleStopEdit();
                }}
                class="flex items-center justify-center w-8 h-8 rounded text-ink-muted hover:text-ink-sub transition-colors cursor-pointer"
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
            class="prose-glass text-base cursor-pointer"
            dangerouslySetInnerHTML={{
              __html: renderMarkdown(displayContent),
            }}
          />
        )}
      </div>
    </article>
  );
}
