"use client";
import { Card, CardContent } from "@/components/ui/card";
import { memo, useMemo, useState } from "react";
import { useDebounceCallback } from "usehooks-ts";
import { DEFAULT_DRAGGABLE_WIDTH } from "../../constants";
import type { Log } from "../../types";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import ResizablePanel from "./resizable-panel";

type Props = {
  log: Log | null;
  onClose: () => void;
  distanceToTop: number;
};

const PANEL_WIDTH_SET_DELAY = 150;

const _LogDetails = ({ log, onClose, distanceToTop }: Props) => {
  const [panelWidth, setPanelWidth] = useState(DEFAULT_DRAGGABLE_WIDTH);

  const debouncedSetPanelWidth = useDebounceCallback((newWidth) => {
    setPanelWidth(newWidth);
  }, PANEL_WIDTH_SET_DELAY);

  const panelStyle = useMemo(
    () => ({
      top: `${distanceToTop}px`,
      width: `${panelWidth}px`,
      height: `calc(100vh - ${distanceToTop}px)`,
      paddingBottom: "1rem",
    }),
    [distanceToTop, panelWidth]
  );

  if (!log) {
    return null;
  }

  return (
    <ResizablePanel
      onResize={debouncedSetPanelWidth}
      onClose={onClose}
      className="absolute right-0 bg-background border-l border-t border-solid font-mono border-border shadow-md overflow-y-auto z-[3]"
      style={panelStyle}
    >
      <LogHeader log={log} onClose={onClose} />

      <div className="space-y-3 border-b-[1px] border-border py-4">
        <div className="mt-[-24px]" />
        <Headers headers={log.request_headers} title="Request Header" />
        <Headers
          headers={flattenObject(JSON.parse(log.request_body))}
          title="Request Body"
        />
        <Headers headers={log.response_headers} title="Response Header" />
        <Headers
          headers={flattenObject(JSON.parse(log.response_body))}
          title="Response Body"
        />
      </div>
      <LogFooter log={log} />
    </ResizablePanel>
  );
};

// Without memo each time trpc makes a request LogDetails re-renders
export const LogDetails = memo(
  _LogDetails,
  (prev, next) => prev.log?.request_id === next.log?.request_id
);

function flattenObject(obj: object, prefix = ""): string[] {
  return Object.entries(obj).flatMap(([key, value]) => {
    const newKey = prefix ? `${prefix}.${key}` : key;
    if (typeof value === "object" && value !== null) {
      return flattenObject(value, newKey);
    }
    return `${newKey}:${value}`;
  });
}

const Headers = ({ headers, title }: { headers: string[]; title: string }) => {
  return (
    <div className="px-3 flex flex-col gap-[2px]">
      <span className="text-sm text-content/65 font-sans">{title}</span>

      <Card className="rounded-[5px] bg-background-subtle">
        <CardContent className="p-2 whitespace-pre-wrap text-[12px]">
          <pre className="font-mono text-[12px] text-gray-600 whitespace-pre">
            {headers.map((header) => {
              const [key, ...valueParts] = header.split(":");
              const value = valueParts.join(":").trim();
              return (
                <div key={header}>
                  <span className="text-content/65 capitalize">{key}</span>
                  <span className="text-content whitespace-pre-line">
                    : {value}
                  </span>
                </div>
              );
            })}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
};
