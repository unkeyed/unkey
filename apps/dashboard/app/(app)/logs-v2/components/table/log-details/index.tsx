"use client";

import type { Log } from "@unkey/clickhouse/src/logs";
import { useMemo } from "react";
import { DEFAULT_DRAGGABLE_WIDTH } from "../../../constants";
import { extractResponseField, safeParseJson } from "../../../utils";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import { LogMetaSection } from "./components/log-meta";
import { LogSection } from "./components/log-section";
import ResizablePanel from "./resizable-panel";

const PANEL_MAX_WIDTH = 600;
const PANEL_MIN_WIDTH = 400;

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

type Props = {
  log: Log | null;
  onClose: () => void;
  distanceToTop: number;
};

export const LogDetails = ({ log, onClose, distanceToTop }: Props) => {
  const panelStyle = useMemo(
    () => createPanelStyle(distanceToTop),
    [distanceToTop]
  );

  if (!log) {
    return null;
  }

  return (
    <ResizablePanel
      minW={PANEL_MIN_WIDTH}
      maxW={PANEL_MAX_WIDTH}
      onClose={onClose}
      className="absolute right-0 bg-background font-mono drop-shadow-2xl overflow-y-auto z-[3]"
      style={panelStyle}
    >
      <LogHeader log={log} onClose={onClose} />

      <div className="space-y-3 border-b-[1px] border-border py-4">
        <div className="mt-[-24px]" />
        <LogSection details={log.request_headers} title="Request Header" />
        <LogSection
          details={JSON.stringify(safeParseJson(log.request_body), null, 2)}
          title="Request Body"
        />
        <LogSection details={log.response_headers} title="Response Header" />
        <LogSection
          details={JSON.stringify(safeParseJson(log.response_body), null, 2)}
          title="Response Body"
        />
      </div>
      <LogFooter log={log} />
      <LogMetaSection
        content={JSON.stringify(extractResponseField(log, "meta"), null, 2)}
      />
    </ResizablePanel>
  );
};
