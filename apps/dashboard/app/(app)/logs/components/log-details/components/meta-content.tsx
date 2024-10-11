import { Card, CardContent } from "@/components/ui/card";
import { useEffect, useState } from "react";
import { createHighlighter } from "shiki";

const highlighter = createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["json"],
});

export function MetaContent({ content }: { content: any }) {
  const [innerHtml, setHtml] = useState("Loading...");

  useEffect(() => {
    highlighter.then((highlight) => {
      const html = highlight.codeToHtml(JSON.stringify(content), {
        lang: "json",
        themes: {
          dark: "github-dark",
          light: "github-light",
        },
        mergeWhitespaces: true,
      });
      setHtml(html);
    });
  }, [content]);

  return (
    <Card className="rounded-[5px]">
      <CardContent
        className="whitespace-pre-wrap text-[12px] w-[300px]"
        dangerouslySetInnerHTML={{
          __html: innerHtml,
        }}
      />
    </Card>
  );
}
