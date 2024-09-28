import { useState } from "react";
import { createHighlighter } from "shiki";
import { useDebounceCallback } from "usehooks-ts";
import { DEFAULT_DRAGGABLE_WIDTH } from "../constants";
import type { Log } from "../data";
import { LogBody } from "./components/log-body";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import ResizablePanel from "./resizable-panel";

export const highlighter = await createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["json"],
});

type Props = {
  log: Log | null;
  onClose: () => void;
};

export const LogDetails = ({ log, onClose }: Props) => {
  const [panelWidth, setPanelWidth] = useState(DEFAULT_DRAGGABLE_WIDTH);
  const debouncedSetPanelWidth = useDebounceCallback((newWidth) => {
    setPanelWidth(newWidth);
  }, 150);

  if (!log) {
    return null;
  }

  return (
    <ResizablePanel
      onResize={debouncedSetPanelWidth}
      className="absolute top-[245px] right-0 bg-background border-l border-t border-solid font-mono border-border shadow-md overflow-y-auto z-[3]"
      style={{
        width: `${panelWidth}px`,
        height: "calc(100vh - 245px)",
        paddingBottom: "1rem",
      }}
    >
      <LogHeader log={log} onClose={onClose} />
      <LogBody log={log} />
      <LogFooter log={log} />
    </ResizablePanel>
  );
};
