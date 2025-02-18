"use client";

import { useMemo } from "react";
import type { AuditData } from "../../../audit.type";
import { DEFAULT_DRAGGABLE_WIDTH, PANEL_MAX_WIDTH, PANEL_MIN_WIDTH } from "../../../constants";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import { LogSection } from "./components/log-section";
import { ResizablePanel } from "./resizable-panel";

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  width: `${DEFAULT_DRAGGABLE_WIDTH}px`,
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

type Props = {
  distanceToTop: number;
  selectedLog: AuditData | null;
  setSelectedLog: (log: AuditData | null) => void;
};

export const AuditLogDetails = ({ distanceToTop, selectedLog, setSelectedLog }: Props) => {
  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);

  if (!selectedLog) {
    return null;
  }

  const handleClose = () => {
    setSelectedLog(null);
  };

  return (
    <ResizablePanel
      minW={PANEL_MIN_WIDTH}
      maxW={PANEL_MAX_WIDTH}
      onClose={handleClose}
      className="absolute right-0 bg-gray-1 dark:bg-black font-mono drop-shadow-2xl overflow-y-auto z-20 p-4"
      style={panelStyle}
    >
      <LogHeader log={selectedLog} onClose={handleClose} />
      <div className="space-y-3 py-4">
        <div className="mt-[-24px]" />

        <LogFooter log={selectedLog} />
        {selectedLog.auditLog.targets.map((target) => {
          const title = String(target.type).charAt(0).toUpperCase() + String(target.type).slice(1);

          return (
            <LogSection key={target.id} details={JSON.stringify(target, null, 2)} title={title} />
          );
        })}
      </div>
    </ResizablePanel>
  );
};
