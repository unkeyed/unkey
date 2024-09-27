"use client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { ButtonGroup } from "@/components/ui/group-button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { format } from "date-fns";
import { Calendar, Clock, Copy, RefreshCcw, Search, X } from "lucide-react";
import { useState } from "react";
import { createHighlighter } from "shiki";
import { ChartsComp } from "./chart";
import { type Log, type ResponseBody, sampleLogs } from "./data";
import ResizablePanel from "./resizable-panel";
import { useDebounceCallback, useDebounceValue } from "usehooks-ts";
import { DEFAULT_DRAGGABLE_WIDTH } from "./constants";
import { toast } from "@/components/ui/toaster";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const YELLOW_STATES = ["RATE_LIMITED", "EXPIRED", "USAGE_EXCEEDED"];
const RED_STATES = ["DISABLED", "FORBIDDEN", "INSUFFICIENT_PERMISSIONS"];

const highlighter = await createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["json"],
});

export default function Page() {
  const [panelWidth, setPanelWidth] = useState(DEFAULT_DRAGGABLE_WIDTH);
  const debouncedSetPanelWidth = useDebounceCallback((newWidth) => {
    setPanelWidth(newWidth);
  }, 150);
  const [showDetails, setShowDetails] = useState(false);

  return (
    <div className="flex flex-col gap-4 items-start w-full">
      {/* Filter Section */}
      <div className="flex items-center gap-2 w-full">
        <div className="w-[330px]">
          <Input type="text" placeholder="Search logs" startIcon={Search} />
        </div>
        <Button variant="outline" size="icon" className="w-10">
          <RefreshCcw className="h-4 w-4" />
        </Button>
        <ButtonGroup>
          <Button variant="outline">
            <Clock className="h-4 w-4" />
            Last hour
          </Button>
          <Button variant="outline">
            {" "}
            <Calendar className="h-4 w-4" />
            Custom
          </Button>
        </ButtonGroup>

        <ButtonGroup>
          <Button variant="outline">Response Status</Button>
          <Button variant="outline">Request ID</Button>
          <Button variant="outline">Api ID</Button>
          <Button variant="outline">Key ID</Button>
        </ButtonGroup>
      </div>
      <ChartsComp />
      {/* Logs section */}
      <div className="w-full">
        <div className="grid grid-cols-[166px_72px_12%_calc(20%+32px)_1fr] text-sm font-medium text-[#666666]">
          <div className="p-2 flex items-center">Time</div>
          <div className="p-2 flex items-center">Status</div>
          <div className="p-2 flex items-center">Host</div>
          <div className="p-2 flex items-center">Request</div>
          <div className="p-2 flex items-center">Message</div>
        </div>
        <div className="w-full border-t border-border" />
        <div className="relative">
          {sampleLogs.map((log, index) => (
            // biome-ignore lint/a11y/useKeyWithClickEvents: don't know what to do atm, may the gods help us
            <div
              key={`${log.request_id}#${index}`}
              onClick={() => setShowDetails(true)}
              className={cn(
                "font-mono grid grid-cols-[166px_72px_12%_calc(20%+32px)_1fr] text-[13px] leading-[14px] mb-[1px] rounded-[5px] h-[26px] cursor-pointer ",
                "hover:bg-background-subtle/50 data-[state=selected]:bg-background-subtle pl-1",
                {
                  "bg-amber-2 text-amber-11  hover:bg-amber-3":
                    YELLOW_STATES.includes(
                      getResponseBodyFieldOutcome(log, "code")
                    ),
                  "bg-red-2 text-red-11  hover:bg-red-3": RED_STATES.includes(
                    getResponseBodyFieldOutcome(log, "code")
                  ),
                }
              )}
            >
              <div className="px-[2px] flex items-center">
                {format(log.time, "MMM dd HH:mm:ss.SS")}
              </div>
              <div className="px-[2px] flex items-center">
                {log.response_status}
              </div>
              <div className="px-[2px] flex items-center">{log.host}</div>
              <div className="px-[2px] flex items-center">{log.path}</div>
              <div className="px-[2px] flex items-center  w-[700px]">
                <span className="truncate">{log.response_body}</span>
              </div>
            </div>
          ))}
          {showDetails && (
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
                  <p className="text-[13px] text-content/65">
                    /api/simple-test
                  </p>
                </div>

                <div className="flex gap-1 items-center">
                  <Badge className="bg-background border border-solid border-red-6 text-red-11 hover:bg-transparent">
                    400
                  </Badge>
                  <span className="text-content/65">|</span>
                  <X
                    onClick={() => setShowDetails(false)}
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
                      __html: highlighter.codeToHtml(
                        getObjectsFromLogs(sampleLogs[0]),
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
                  <div className="absolute bottom-2 right-3">
                    <Button
                      size="block"
                      variant="primary"
                      className="bg-black border-border text-current"
                      onClick={() => {
                        navigator.clipboard.writeText(
                          getObjectsFromLogs(sampleLogs[0])
                        );
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
                        navigator.clipboard.writeText(
                          String(sampleLogs[0].time)
                        );
                        toast.success("Time copied to clipboard");
                      }}
                    >
                      <span className="text-sm text-content/65">Time</span>
                      <span className="text-[13px] font-mono ">
                        {format(sampleLogs[0].time, "MMM dd HH:mm:ss.SS")}
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
                        navigator.clipboard.writeText(
                          String(sampleLogs[0].host)
                        );
                        toast.success("Host copied to clipboard");
                      }}
                    >
                      <span className="text-sm  text-content/65">Host</span>
                      <span className="text-[13px] font-mono ">
                        {sampleLogs[0].host}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="left">Copy Host</TooltipContent>
                  </Tooltip>
                </TooltipProvider>

                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger
                      className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
                      onClick={() => {
                        navigator.clipboard.writeText(sampleLogs[0].path);
                        toast.success("Request path copied to clipboard");
                      }}
                    >
                      <span className="text-sm  text-content/65">
                        Request Path
                      </span>
                      <span className="text-[13px] font-mono ">
                        {sampleLogs[0].path}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="left">
                      Copy Request Path
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger
                      className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
                      onClick={() => {
                        navigator.clipboard.writeText(sampleLogs[0].request_id);
                        toast.success("Request ID copied to clipboard");
                      }}
                    >
                      <span className="text-sm  text-content/65">
                        Request ID
                      </span>
                      <span className="text-[13px] font-mono">
                        {sampleLogs[0].request_id}
                      </span>
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
                          sampleLogs[0].request_headers.at(-1)
                        );
                        toast.success("Request user agent copied to clipboard");
                      }}
                    >
                      <span className="text-sm text-content/65">
                        Request User Agent
                      </span>
                      <span className="text-[13px] font-mono">
                        {sampleLogs[0].request_headers.at(-1)}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="left">
                      Copy Request User Agent
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger
                      className="flex w-full justify-between border-border border-b border-solid pr-3 py-[10px] items-center"
                      onClick={() => {
                        navigator.clipboard.writeText(
                          JSON.stringify(
                            getResponseBodyFieldOutcome(sampleLogs[0], "code")
                          )
                        );
                        toast.success("Request user agent copied to clipboard");
                      }}
                    >
                      <span className="text-sm text-content/65">Outcome</span>
                      <span className="text-[13px] font-mono">
                        <Badge
                          className={cn({
                            "bg-amber-2 text-amber-11  hover:bg-amber-3":
                              YELLOW_STATES.includes(
                                getResponseBodyFieldOutcome(
                                  sampleLogs[0],
                                  "code"
                                )
                              ),
                            "bg-red-2 text-red-11  hover:bg-red-3":
                              RED_STATES.includes(
                                getResponseBodyFieldOutcome(
                                  sampleLogs[0],
                                  "code"
                                )
                              ),
                          })}
                        >
                          {getResponseBodyFieldOutcome(sampleLogs[0], "code")}
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
                            getResponseBodyFieldOutcome(
                              sampleLogs[0],
                              "permissions"
                            )
                          )
                        );
                        toast.success("Request user agent copied to clipboard");
                      }}
                    >
                      <span className="text-sm text-content/65">
                        Permissions
                      </span>
                      <span className="text-[13px] font-mono flex gap-1">
                        {(
                          getResponseBodyFieldOutcome(
                            sampleLogs[0],
                            "permissions"
                          ) as string[]
                        ).map((permission) => (
                          <Badge variant="secondary">{permission}</Badge>
                        ))}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="left">
                      Copy Permissions
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>
                <TooltipProvider>
                  <Tooltip>
                    <TooltipTrigger
                      className="flex w-full justify-between pr-3 py-[10px] items-center"
                      onClick={() => {
                        navigator.clipboard.writeText(
                          JSON.stringify(
                            getResponseBodyFieldOutcome(sampleLogs[0], "meta")
                          )
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
                                  getResponseBodyFieldOutcome(
                                    sampleLogs[0],
                                    "meta"
                                  )
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
          )}
        </div>
      </div>
    </div>
  );
}

//TODO: parsing might fail add check
const getResponseBodyFieldOutcome = (log: Log, fieldName: any) => {
  //TODO: fix the type issue
  return (JSON.parse(log.response_body) as ResponseBody)[fieldName];
};

const getObjectsFromLogs = (log: Log) => {
  const obj: Record<string, unknown> = {};

  obj.responseBody = JSON.parse(log.response_body);
  obj.responseHeaders = log.response_headers;
  obj.requestHeaders = log.request_headers;

  return JSON.stringify(obj, null, 2);
};
