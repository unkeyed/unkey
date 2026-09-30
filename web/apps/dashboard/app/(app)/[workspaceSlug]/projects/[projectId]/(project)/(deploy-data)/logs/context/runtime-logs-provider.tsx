"use client";

import { createContext, useCallback, useContext, useState } from "react";
import type { RuntimeLog } from "../types";

type RuntimeLogsContextType = {
  selectedLog: RuntimeLog | null;
  setSelectedLog: (log: RuntimeLog | null) => void;
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
  // Bumped by the refresh control; the logs query watches this to re-anchor its
  // window so a refresh surfaces logs that arrived since the last anchor.
  refreshNonce: number;
  refresh: () => void;
};

const RuntimeLogsContext = createContext<RuntimeLogsContextType | undefined>(undefined);

export function RuntimeLogsProvider({ children }: { children: React.ReactNode }) {
  const [selectedLog, setSelectedLog] = useState<RuntimeLog | null>(null);
  const [isLive, setIsLive] = useState(false);
  const [refreshNonce, setRefreshNonce] = useState(0);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  const refresh = useCallback(() => {
    setRefreshNonce((prev) => prev + 1);
  }, []);

  return (
    <RuntimeLogsContext.Provider
      value={{ selectedLog, setSelectedLog, isLive, toggleLive, refreshNonce, refresh }}
    >
      {children}
    </RuntimeLogsContext.Provider>
  );
}

export function useRuntimeLogs() {
  const context = useContext(RuntimeLogsContext);
  if (!context) {
    throw new Error("useRuntimeLogs must be used within RuntimeLogsProvider");
  }
  return context;
}
