"use client";
import type { Log } from "@unkey/clickhouse/src/logs";
import { type PropsWithChildren, createContext, useContext, useState } from "react";
import { useProjectLayout } from "../../layout-provider";

type GatewayLogsContextType = {
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
  selectedLog: Log | null;
  setSelectedLog: (log: Log | null) => void;
};

const GatewayLogsContext = createContext<GatewayLogsContextType | null>(null);

export const GatewayLogsProvider = ({ children }: PropsWithChildren) => {
  const { setIsDetailsOpen } = useProjectLayout();
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);
  const [isLive, setIsLive] = useState(false);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  return (
    <GatewayLogsContext.Provider
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
    </GatewayLogsContext.Provider>
  );
};

export const useGatewayLogsContext = () => {
  const context = useContext(GatewayLogsContext);
  if (!context) {
    throw new Error("useGatewayLogsContext must be used within a GatewayLogsProvider");
  }
  return context;
};
