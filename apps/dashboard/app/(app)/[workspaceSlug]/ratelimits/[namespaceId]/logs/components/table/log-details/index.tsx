"use client";

import { extractResponseField, safeParseJson } from "@/app/(app)/[workspaceSlug]/logs/utils";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import { useMemo } from "react";
import { DEFAULT_DRAGGABLE_WIDTH } from "../../../constants";
import { useRatelimitLogsContext } from "../../../context/logs";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import { LogMetaSection } from "./components/log-meta";
import { LogSection } from "./components/log-section";

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

type Props = {
  distanceToTop: number;
};

export const RatelimitLogDetails = ({ distanceToTop }: Props) => {
  const { setSelectedLog, selectedLog: log } = useRatelimitLogsContext();
  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);

  if (!log) {
    return null;
  }

  const handleClose = () => {
    setSelectedLog(null);
  };

  return (
    <ResizablePanel
      onClose={handleClose}
      className="absolute right-0 bg-gray-1 dark:bg-black font-mono drop-shadow-2xl overflow-y-auto z-20 p-4"
      style={panelStyle}
    >
      <LogHeader log={log} onClose={handleClose} />

      <LogSection
        details={log.request_headers.length ? log.request_headers : "<EMPTY>"}
        title="Request Header"
      />
      <LogSection
        details={
          JSON.stringify(safeParseJson(log.request_body), null, 2) === "null"
            ? "<EMPTY>"
            : JSON.stringify(safeParseJson(log.request_body), null, 2)
        }
        title="Request Body"
      />
      <LogSection
        details={log.response_headers.length ? log.response_headers : "<EMPTY>"}
        title="Response Header"
      />
      <LogSection
        details={
          JSON.stringify(safeParseJson(log.response_body), null, 2) === "null"
            ? "<EMPTY>"
            : JSON.stringify(safeParseJson(log.response_body), null, 2)
        }
        title="Response Body"
      />
      <div className="mt-3" />
      <LogFooter log={log} />
      <LogMetaSection
        content={
          JSON.stringify(extractResponseField(log, "meta"), null, 2) === "null"
            ? "<EMPTY>"
            : JSON.stringify(extractResponseField(log, "meta"), null, 2)
        }
      />
    </ResizablePanel>
  );
};
