import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Copy } from "lucide-react";
import React, { useEffect } from "react";
import { createHighlighter } from "shiki";
import type { Log } from "../../data";
import { getObjectsFromLogs } from "../../utils";

type Props = {
  log: Log;
};

const highlighter = createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["json"],
});

export function LogBody({ log }: Props) {
  const [innerHtml, setHtml] = React.useState("Loading...");

  useEffect(() => {
    highlighter.then((highlight) => {
      const html = highlight.codeToHtml(getObjectsFromLogs(log), {
        lang: "json",
        themes: {
          dark: "github-dark",
          light: "github-light",
        },
        mergeWhitespaces: true,
      });
      setHtml(html);
    });
  }, [log]);

  return (
    <div className="p-2">
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
    </div>
  );
}
