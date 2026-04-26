// deployment-logs-context.tsx
"use client";

import { type ReactNode, createContext, useContext, useState } from "react";

type DeploymentLogsContextValue = {
  isExpanded: boolean;
  setExpanded: (expanded: boolean) => void;
  toggleExpanded: () => void;
  logType: "sentinel" | "runtime";
  setLogType: (type: "sentinel" | "runtime") => void;
  isLive: boolean;
  setIsLive: (live: boolean) => void;
  toggleLive: () => void;
};

const DeploymentLogsContext = createContext<DeploymentLogsContextValue | null>(null);

export function DeploymentLogsProvider({ children }: { children: ReactNode }) {
  const [isExpanded, setExpanded] = useState(true);
  const [logType, setLogType] = useState<"sentinel" | "runtime">("runtime");
  const [isLive, setIsLive] = useState(false);

  const toggleExpanded = () => setExpanded((prev) => !prev);
  const toggleLive = () => setIsLive((prev) => !prev);

  return (
    <DeploymentLogsContext.Provider
      value={{
        isExpanded,
        setExpanded,
        toggleExpanded,
        logType,
        setLogType,
        isLive,
        setIsLive,
        toggleLive,
      }}
    >
      {children}
    </DeploymentLogsContext.Provider>
  );
}

export function useDeploymentLogsContext() {
  const context = useContext(DeploymentLogsContext);
  if (!context) {
    throw new Error("useDeploymentLogsContext must be used within DeploymentLogsProvider");
  }
  return context;
}
