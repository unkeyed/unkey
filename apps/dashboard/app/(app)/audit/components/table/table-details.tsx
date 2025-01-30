"use client";

import { LogSection } from "@/app/(app)/logs/components/table/log-details/components/log-section";
import ResizablePanel from "@/app/(app)/logs/components/table/log-details/resizable-panel";
import { useMemo } from "react";
import { LogFooter } from "./log-footer";
import { LogHeader } from "./log-header";
import type { Data } from "./types";

type Props = {
  log: Data | null;
  onClose: () => void;
  distanceToTop: number;
};

const PANEL_MAX_WIDTH = 600;
const PANEL_MIN_WIDTH = 400;

const createPanelStyle = (distanceToTop: number) => ({
  top: `${distanceToTop}px`,
  width: "500px",
  height: `calc(100vh - ${distanceToTop}px)`,
  paddingBottom: "1rem",
});

export const LogDetails = ({ log, onClose, distanceToTop }: Props) => {
  const panelStyle = useMemo(() => createPanelStyle(distanceToTop), [distanceToTop]);

  if (!log) {
    return null;
  }

  return (
    <ResizablePanel
      minW={PANEL_MIN_WIDTH}
      maxW={PANEL_MAX_WIDTH}
      onClose={onClose}
      className="absolute right-0 bg-gray-1 dark:bg-black font-mono drop-shadow-2xl overflow-y-auto z-20 p-4"
      style={panelStyle}
    >
      <LogHeader log={log} onClose={onClose} />
      <div className="space-y-3 py-4">
        <div className="mt-[-24px]" />

        <LogFooter log={log} />
        {log.auditLog.targets.map((target) => {
          const title = String(target.type).charAt(0).toUpperCase() + String(target.type).slice(1);

          return (
            <LogSection key={target.id} details={JSON.stringify(target, null, 2)} title={title} />
          );
        })}
      </div>
    </ResizablePanel>
  );
};
