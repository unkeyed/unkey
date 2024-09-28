import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { Copy, X } from "lucide-react";
import { useState } from "react";
import { createHighlighter } from "shiki";
import { toast } from "sonner";
import { useDebounceCallback } from "usehooks-ts";
import {
  DEFAULT_DRAGGABLE_WIDTH,
  RED_STATES,
  YELLOW_STATES,
} from "../constants";
import type { Log } from "../data";
import {
  getObjectsFromLogs,
  getOutcomeIfValid,
  getRequestHeader,
  getResponseBodyFieldOutcome,
} from "../utils";
import ResizablePanel from "./resizable-panel";

const highlighter = await createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["json"],
});

type Props = {
  log: Log | null;
  onClose: () => void;
};

export const LogDetails = ({ log, onClose }: Props) => {
  const [panelWidth, setPanelWidth] = useState(DEFAULT_DRAGGABLE_WIDTH);
  const debouncedSetPanelWidth = useDebounceCallback((newWidth) => {
    setPanelWidth(newWidth);
  }, 150);

  if (!log) {
    return null;
  }

  const outcome = getOutcomeIfValid(log);

  return (
    <ResizablePanel
      onResize={debouncedSetPanelWidth}
      className="fixed top-[245px] right-0 bg-background border-l border-t border-solid font-mono border-border shadow-md"
      style={{
        width: `${panelWidth}px`,
        height: "calc(100vh - 245px)",
      }}
    >
      <div className="border-b-[1px] px-3 py-4 flex justify-between border-border items-center">
        <div className="flex gap-1">
          <Badge variant="secondary" className="bg-transparent">
            POST
          </Badge>
          <p className="text-[13px] text-content/65">/api/simple-test</p>
        </div>

        <div className="flex gap-1 items-center">
          <Badge className="bg-background border border-solid border-red-6 text-red-11 hover:bg-transparent">
            400
          </Badge>
          <span className="text-content/65">|</span>
          <X
            onClick={onClose}
            size="22"
            strokeWidth="1.5"
            className="text-content/65 cursor-pointer"
          />
        </div>
      </div>
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
      <div className="font-sans pl-3">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(String(log.time));
                toast.success("Time copied to clipboard");
              }}
            >
              <span className="text-sm text-content/65">Time</span>
              <span className="text-[13px] font-mono ">
                {format(log.time, "MMM dd HH:mm:ss.SS")}
              </span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Time</TooltipContent>
          </Tooltip>
        </TooltipProvider>

        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(String(log.host));
                toast.success("Host copied to clipboard");
              }}
            >
              <span className="text-sm  text-content/65">Host</span>
              <span className="text-[13px] font-mono ">{log.host}</span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Host</TooltipContent>
          </Tooltip>
        </TooltipProvider>

        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(log.path);
                toast.success("Request path copied to clipboard");
              }}
            >
              <span className="text-sm  text-content/65">Request Path</span>
              <span className="text-[13px] font-mono ">{log.path}</span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Request Path</TooltipContent>
          </Tooltip>
        </TooltipProvider>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(log.request_id);
                toast.success("Request ID copied to clipboard");
              }}
            >
              <span className="text-sm  text-content/65">Request ID</span>
              <span className="text-[13px] font-mono">{log.request_id}</span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Request ID</TooltipContent>
          </Tooltip>
        </TooltipProvider>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(
                  getRequestHeader(log, "user-agent") ?? ""
                );
                toast.success("Request user agent copied to clipboard");
              }}
            >
              <span className="text-sm text-content/65">
                Request User Agent
              </span>
              <span className="text-[13px] font-mono">
                {log.request_headers.at(-1)}
              </span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Request User Agent</TooltipContent>
          </Tooltip>
        </TooltipProvider>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(
                  JSON.stringify(getResponseBodyFieldOutcome(log, "code"))
                );
                toast.success("Request user agent copied to clipboard");
              }}
            >
              <span className="text-sm text-content/65">Outcome</span>
              <span className="text-[13px] font-mono">
                <Badge
                  className={cn({
                    "bg-amber-2 text-amber-11  hover:bg-amber-3":
                      YELLOW_STATES.includes(outcome),
                    "bg-red-2 text-red-11  hover:bg-red-3":
                      RED_STATES.includes(outcome),
                  })}
                >
                  {getResponseBodyFieldOutcome(log, "code")}
                </Badge>
              </span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Outcome</TooltipContent>
          </Tooltip>
        </TooltipProvider>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(
                  JSON.stringify(
                    getResponseBodyFieldOutcome(log, "permissions")
                  )
                );
                toast.success("Request user agent copied to clipboard");
              }}
            >
              <span className="text-sm text-content/65">Permissions</span>
              <span className="text-[13px] font-mono flex gap-1">
                {(
                  getResponseBodyFieldOutcome(log, "permissions") as string[]
                ).map((permission) => (
                  <Badge variant="secondary">{permission}</Badge>
                ))}
              </span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Permissions</TooltipContent>
          </Tooltip>
        </TooltipProvider>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger
              className="flex w-full justify-between pr-3 py-[10px] items-center"
              onClick={() => {
                navigator.clipboard.writeText(
                  JSON.stringify(getResponseBodyFieldOutcome(log, "meta"))
                );
                toast.success("Request user agent copied to clipboard");
              }}
            >
              <span className="text-sm text-content/65">Meta</span>
              <span className="text-[13px] font-mono">
                <Card className="rounded-[5px]">
                  <CardContent
                    className="whitespace-pre-wrap text-[12px] w-[300px]"
                    dangerouslySetInnerHTML={{
                      __html: highlighter.codeToHtml(
                        JSON.stringify(
                          getResponseBodyFieldOutcome(log, "meta")
                        ),
                        {
                          lang: "json",
                          themes: {
                            dark: "github-dark",
                            light: "github-light",
                          },
                          mergeWhitespaces: true,
                        }
                      ),
                    }}
                  />
                </Card>
              </span>
            </TooltipTrigger>
            <TooltipContent side="left">Copy Meta</TooltipContent>
          </Tooltip>
        </TooltipProvider>
      </div>
    </ResizablePanel>
  );
};
