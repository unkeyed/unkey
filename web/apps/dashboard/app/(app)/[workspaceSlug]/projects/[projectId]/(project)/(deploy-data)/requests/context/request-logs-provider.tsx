"use client";
import type { RequestLogsResponse } from "@unkey/clickhouse/src/frontline";
import { type PropsWithChildren, createContext, useCallback, useContext, useState } from "react";

type RequestLogsContextType = {
  selectedLog: RequestLogsResponse | null;
  setSelectedLog: (log: RequestLogsResponse | null) => void;
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
  // Bumped by the refresh control; the logs query watches this to re-anchor its
  // window so a refresh surfaces logs that arrived since the last anchor.
  refreshNonce: number;
  refresh: () => void;
};

const RequestLogsContext = createContext<RequestLogsContextType | null>(null);

export const RequestLogsProvider = ({ children }: PropsWithChildren) => {
  const [selectedLog, setSelectedLog] = useState<RequestLogsResponse | null>(null);
  const [isLive, setIsLive] = useState(false);
  const [refreshNonce, setRefreshNonce] = useState(0);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  const refresh = useCallback(() => {
    setRefreshNonce((prev) => prev + 1);
  }, []);

  return (
    <RequestLogsContext.Provider
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
    </RequestLogsContext.Provider>
  );
};

export const useRequestLogsContext = () => {
  const context = useContext(RequestLogsContext);
  if (!context) {
    throw new Error("useRequestLogsContext must be used within a RequestLogsProvider");
  }
  return context;
};
