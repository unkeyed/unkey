import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Copy } from "lucide-react";
import React, { useEffect } from "react";
import { createHighlighter } from "shiki";

type Props = {
  field: string;
  title: string;
};

const highlighter = createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["json"],
});

export function LogBody({ field, title }: Props) {
  const [innerHtml, setHtml] = React.useState("Loading...");

  useEffect(() => {
    highlighter.then((highlight) => {
      const html = highlight.codeToHtml(
        JSON.stringify(JSON.parse(field), null, 2),
        {
          lang: "json",
          themes: {
            dark: "github-dark",
            light: "github-light",
          },
          mergeWhitespaces: true,
        }
      );
      setHtml(html);
    });
  }, [field]);

  return (
    <div className="pl-3 flex flex-col gap-[2px] mt-[10px]">
      <span className="text-sm text-content/65 font-sans mb-[10px]">
        {title}
      </span>
      <Card className="rounded-[5px] relative">
        <CardContent
          className="whitespace-pre-wrap text-[12px]"
          dangerouslySetInnerHTML={{
            __html: innerHtml,
          }}
        />
        <div className="absolute bottom-2 right-3">
          <Button
            size="block"
            variant="primary"
            className="bg-background border-border text-current"
          >
            <Copy className="w-4 h-4" />
          </Button>
        </div>
      </Card>
      <div className="w-full border-border border-solid py-[10px] items-center border-b" />
    </div>
  );
}
