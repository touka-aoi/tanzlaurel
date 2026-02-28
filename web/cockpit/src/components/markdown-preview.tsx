import { useMemo } from "preact/hooks";
import { renderMarkdown } from "../lib/markdown";

interface Props {
  text: string;
}

export function MarkdownPreview({ text }: Props) {
  const html = useMemo(() => renderMarkdown(text), [text]);

  return (
    <div class="h-full overflow-y-auto p-4">
      <div
        class="prose prose-invert prose-sm max-w-none
          prose-headings:text-white/90 prose-p:text-white/70
          prose-a:text-blue-400 prose-code:text-blue-300
          prose-pre:bg-white/5 prose-pre:border prose-pre:border-white/10
          prose-blockquote:border-blue-400/30"
        dangerouslySetInnerHTML={{ __html: html }}
      />
    </div>
  );
}
