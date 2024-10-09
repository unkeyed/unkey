"use client";
import { useState, useMemo, useRef } from "react";
import { useDebounceCallback, useOnClickOutside } from "usehooks-ts";
import { DEFAULT_DRAGGABLE_WIDTH } from "../constants";
import type { Log } from "../data";
import { LogBody } from "./components/log-body";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import ResizablePanel from "./resizable-panel";

type Props = {
  log: Log | null;
  onClose: () => void;
  distanceToTop: number;
};

const PANEL_WIDTH_SET_DELAY = 150;

export const LogDetails = ({ log, onClose, distanceToTop }: Props) => {
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
      <LogBody log={log} />
      <LogFooter log={log} />
    </ResizablePanel>
  );
};
