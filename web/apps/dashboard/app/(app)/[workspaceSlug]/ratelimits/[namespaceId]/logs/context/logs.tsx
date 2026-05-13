"use client";

import { type PropsWithChildren, createContext, useContext, useState } from "react";
import type { EnrichedRatelimitLog } from "../components/table/hooks/use-logs-query";

type LogsContextType = {
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
  selectedLog: EnrichedRatelimitLog | null;
  setSelectedLog: (log: EnrichedRatelimitLog | null) => void;
  namespaceId: string;
};

const LogsContext = createContext<LogsContextType | null>(null);

export const RatelimitLogsProvider = ({
  children,
  namespaceId,
}: PropsWithChildren<{ namespaceId: string }>) => {
  const [selectedLog, setSelectedLog] = useState<EnrichedRatelimitLog | null>(null);
  const [isLive, setIsLive] = useState(false);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  return (
    <LogsContext.Provider
      value={{
        isLive,
        toggleLive,
        selectedLog,
        setSelectedLog,
        namespaceId,
      }}
    >
      {children}
    </LogsContext.Provider>
  );
};

export const useRatelimitLogsContext = () => {
  const context = useContext(LogsContext);
  if (!context) {
    throw new Error("useLogsContext must be used within a LogsProvider");
  }
  return context;
};
