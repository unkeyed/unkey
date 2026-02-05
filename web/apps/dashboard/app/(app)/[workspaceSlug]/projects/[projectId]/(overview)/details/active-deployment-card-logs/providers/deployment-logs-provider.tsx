// deployment-logs-context.tsx
"use client";

import { type ReactNode, createContext, useContext, useState } from "react";

type DeploymentLogsContextValue = {
  isExpanded: boolean;
  setExpanded: (expanded: boolean) => void;
  toggleExpanded: () => void;
  logType: "sentinel" | "runtime";
  setLogType: (type: "sentinel" | "runtime") => void;
};

const DeploymentLogsContext = createContext<DeploymentLogsContextValue | null>(null);

export function DeploymentLogsProvider({ children }: { children: ReactNode }) {
  const [isExpanded, setExpanded] = useState(true);
  const [logType, setLogType] = useState<"sentinel" | "runtime">("runtime");

  const toggleExpanded = () => setExpanded((prev) => !prev);

  return (
    <DeploymentLogsContext.Provider
      value={{ isExpanded, setExpanded, toggleExpanded, logType, setLogType }}
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
