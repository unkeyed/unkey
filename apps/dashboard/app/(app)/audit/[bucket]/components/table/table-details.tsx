"use client";

import { LogSection } from "@/app/(app)/logs/components/table/log-details/components/log-section";
import { memo, useMemo, useState } from "react";
import { useDebounceCallback } from "usehooks-ts";
import ResizablePanel from "../../../../logs/components/table/log-details/resizable-panel";
import { LogFooter } from "./log-footer";
import { LogHeader } from "./log-header";
import type { Data } from "./types";

type Props = {
  log: Data | null;
  onClose: () => void;
  distanceToTop: number;
};

const DEFAULT_DRAGGABLE_WIDTH = 450;
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
    [distanceToTop, panelWidth],
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

export const LogDetails = memo(
  _LogDetails,
  (prev, next) => prev.log?.auditLog.id === next.log?.auditLog.id,
);
