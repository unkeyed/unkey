"use client";

import { createContext, useContext, useState } from "react";
import type { RuntimeLog } from "../types";

type RuntimeLogsContextType = {
  selectedLog: RuntimeLog | null;
  setSelectedLog: (log: RuntimeLog | null) => void;
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
};

const RuntimeLogsContext = createContext<RuntimeLogsContextType | undefined>(undefined);

export function RuntimeLogsProvider({ children }: { children: React.ReactNode }) {
  const [selectedLog, setSelectedLog] = useState<RuntimeLog | null>(null);
  const [isLive, setIsLive] = useState(false);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  return (
    <RuntimeLogsContext.Provider value={{ selectedLog, setSelectedLog, isLive, toggleLive }}>
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
