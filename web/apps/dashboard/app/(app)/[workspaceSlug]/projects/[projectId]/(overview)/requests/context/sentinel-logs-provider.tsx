"use client";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { type PropsWithChildren, createContext, useContext, useState } from "react";

type SentinelLogsContextType = {
  selectedLog: SentinelLogsResponse | null;
  setSelectedLog: (log: SentinelLogsResponse | null) => void;
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
};

const SentinelLogsContext = createContext<SentinelLogsContextType | null>(null);

export const SentinelLogsProvider = ({ children }: PropsWithChildren) => {
  const [selectedLog, setSelectedLog] = useState<SentinelLogsResponse | null>(null);
  const [isLive, setIsLive] = useState(false);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  return (
    <SentinelLogsContext.Provider
      value={{
        selectedLog,
        setSelectedLog,
        isLive,
        toggleLive,
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
