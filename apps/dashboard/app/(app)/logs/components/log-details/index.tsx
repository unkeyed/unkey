"use client";

import { memo, useMemo, useState } from "react";
import { useDebounceCallback } from "usehooks-ts";
import { DEFAULT_DRAGGABLE_WIDTH } from "../../constants";
import type { Log } from "../../types";
import { LogFooter } from "./components/log-footer";
import ResizablePanel from "./resizable-panel";
import { getResponseBodyFieldOutcome } from "../../utils";
import { LogMetaSection } from "./components/log-meta";
import { LogHeader } from "./components/log-header";
import { LogSection } from "./components/log-section";

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
        <LogSection details={log.request_headers} title="Request Header" />
        <LogSection
          details={flattenObject(JSON.parse(log.request_body))}
          title="Request Body"
        />
        <LogSection details={log.response_headers} title="Response Header" />
        <LogSection
          details={flattenObject(JSON.parse(log.response_body))}
          title="Response Body"
        />
      </div>
      <LogFooter log={log} />
      <LogMetaSection
        content={JSON.stringify(
          getResponseBodyFieldOutcome(log, "meta"),
          null,
          2
        )}
      />
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
