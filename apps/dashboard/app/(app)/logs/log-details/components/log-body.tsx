import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { toast } from "@/components/ui/toaster";
import { Copy } from "lucide-react";
import { highlighter } from "..";
import type { Log } from "../../data";
import { getObjectsFromLogs } from "../../utils";

type Props = {
  log: Log;
};

export const LogBody = ({ log }: Props) => {
  return (
    <div className="p-2">
      <Card className="rounded-[5px] relative">
        <CardContent
          className="whitespace-pre-wrap text-[12px]"
          dangerouslySetInnerHTML={{
            __html: highlighter.codeToHtml(getObjectsFromLogs(log), {
              lang: "json",
              themes: {
                dark: "github-dark",
                light: "github-light",
              },
              mergeWhitespaces: true,
            }),
          }}
        />
        <div className="absolute bottom-2 right-3">
          <Button
            size="block"
            variant="primary"
            className="bg-background border-border text-current"
            onClick={() => {
              navigator.clipboard.writeText(getObjectsFromLogs(log));
              toast.success("Copied to clipboard");
            }}
          >
            <Copy className="w-4 h-4" />
          </Button>
        </div>
      </Card>
    </div>
  );
};
