"use client";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import { useMemo } from "react";

import { DEFAULT_DRAGGABLE_WIDTH } from "@/app/(app)/logs/constants";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { useFetchRequestDetails } from "./components/hooks/use-logs-query";
import { extractResponseField, safeParseJson } from "@/app/(app)/logs/utils";
import { LogHeader } from "@/app/(app)/logs/components/table/log-details/components/log-header";
import { LogMetaSection } from "@/app/(app)/logs/components/table/log-details/components/log-meta";
import { LogFooter } from "@/app/(app)/logs/components/table/log-details/components/log-footer";
import { LogSection } from "@/app/(app)/logs/components/table/log-details/components/log-section";

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

type Props = {
  distanceToTop: number;
  selectedLog: KeyDetailsLog | null;
  onLogSelect: (log: KeyDetailsLog | null) => void;
};

export const KeyDetailsDrawer = ({
  distanceToTop,
  onLogSelect,
  selectedLog,
}: Props) => {
  const panelStyle = useMemo(
    () => createPanelStyle(distanceToTop),
    [distanceToTop]
  );

  const { log } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  const handleClose = () => {
    onLogSelect(null);
  };

  if (!selectedLog) {
    return;
  }
  if (!log) {
    return;
  }

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
