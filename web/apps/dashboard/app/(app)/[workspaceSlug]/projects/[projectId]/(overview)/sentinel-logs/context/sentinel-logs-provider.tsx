"use client";
import type { SentinelLogsResponse } from "@unkey/clickhouse/src/sentinel";
import { type PropsWithChildren, createContext, useContext, useState } from "react";
import { useProject } from "../../layout-provider";

type SentinelLogsContextType = {
  selectedLog: SentinelLogsResponse | null;
  setSelectedLog: (log: SentinelLogsResponse | null) => void;
};

const SentinelLogsContext = createContext<SentinelLogsContextType | null>(null);

export const SentinelLogsProvider = ({ children }: PropsWithChildren) => {
  const { setIsDetailsOpen } = useProject();
  const [selectedLog, setSelectedLog] = useState<SentinelLogsResponse | null>(null);

  return (
    <SentinelLogsContext.Provider
      value={{
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
