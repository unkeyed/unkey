"use client";
import { ResizablePanel } from "@/components/logs/details/resizable-panel";
import { useEffect, useMemo, useState } from "react";

import { LogFooter } from "@/app/(app)/logs/components/table/log-details/components/log-footer";
import { LogHeader } from "@/app/(app)/logs/components/table/log-details/components/log-header";
import { LogSection } from "@/app/(app)/logs/components/table/log-details/components/log-section";
import { DEFAULT_DRAGGABLE_WIDTH } from "@/app/(app)/logs/constants";
import { safeParseJson } from "@/app/(app)/logs/utils";
import type { KeyDetailsLog } from "@unkey/clickhouse/src/verifications";
import { toast } from "@unkey/ui";
import { useFetchRequestDetails } from "./components/hooks/use-logs-query";

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

export const KeyDetailsDrawer = ({ distanceToTop, onLogSelect, selectedLog }: Props) => {
  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);
  const { log, error } = useFetchRequestDetails({
    requestId: selectedLog?.request_id,
  });

  const [errorShown, setErrorShown] = useState(false);

  useEffect(() => {
    if (!errorShown && selectedLog) {
      if (error) {
        toast.error("Error Loading Log Details", {
          description: `${
            error.message ||
            "An unexpected error occurred while fetching log data. Please try again."
          }`,
        });
        setErrorShown(true);
      } else if (!log) {
        toast.error("Log Data Unavailable", {
          description:
            "Could not retrieve log information for this key. The log may have been deleted or is still processing.",
        });
        setErrorShown(true);
      }
    }

    if (!selectedLog) {
      setErrorShown(false);
    }
  }, [error, log, selectedLog, errorShown]);

  const handleClose = () => {
    onLogSelect(null);
  };

  if (!selectedLog) {
    return null;
  }

  if (error || !log) {
    return null;
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
    </ResizablePanel>
  );
};
