"use client";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { type PropsWithChildren, createContext, useCallback, useContext, useState } from "react";

type SentinelLogsContextType = {
  selectedLog: SentinelLogsResponse | null;
  setSelectedLog: (log: SentinelLogsResponse | null) => void;
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
  // Bumped by the refresh control; the logs query watches this to re-anchor its
  // window so a refresh surfaces logs that arrived since the last anchor.
  refreshNonce: number;
  refresh: () => void;
};

const SentinelLogsContext = createContext<SentinelLogsContextType | null>(null);

export const SentinelLogsProvider = ({ children }: PropsWithChildren) => {
  const [selectedLog, setSelectedLog] = useState<SentinelLogsResponse | null>(null);
  const [isLive, setIsLive] = useState(false);
  const [refreshNonce, setRefreshNonce] = useState(0);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  const refresh = useCallback(() => {
    setRefreshNonce((prev) => prev + 1);
  }, []);

  return (
    <SentinelLogsContext.Provider
      value={{
        selectedLog,
        setSelectedLog,
        isLive,
        toggleLive,
        refreshNonce,
        refresh,
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
