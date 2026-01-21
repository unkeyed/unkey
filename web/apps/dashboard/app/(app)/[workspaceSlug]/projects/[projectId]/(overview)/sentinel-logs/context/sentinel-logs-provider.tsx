"use client";
import type { Log } from "@unkey/clickhouse/src/logs";
import { type PropsWithChildren, createContext, useContext, useState } from "react";
import { useProject } from "../../layout-provider";

type SentinelLogsContextType = {
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
  selectedLog: Log | null;
  setSelectedLog: (log: Log | null) => void;
};

const SentinelLogsContext = createContext<SentinelLogsContextType | null>(null);

export const SentinelLogsProvider = ({ children }: PropsWithChildren) => {
  const { setIsDetailsOpen } = useProject();
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);
  const [isLive, setIsLive] = useState(false);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  return (
    <SentinelLogsContext.Provider
      value={{
        isLive,
        toggleLive,
        selectedLog,
        setSelectedLog: (log) => {
          if (log) {
            setIsDetailsOpen(false);
          }
          setSelectedLog(log);
        },
      }}
    >
      {children}
    </SentinelLogsContext.Provider>
  );
};

export const useSentinelLogsContext = () => {
  const context = useContext(SentinelLogsContext);
  if (!context) {
    throw new Error("useSentinelLogsContext must be used within a SentinelLogsProvider");
  }
  return context;
};
