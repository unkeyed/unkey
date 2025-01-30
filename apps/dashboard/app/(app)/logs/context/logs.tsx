"use client";

import type { Log } from "@unkey/clickhouse/src/logs";
import { type PropsWithChildren, createContext, useContext, useState } from "react";

type DisplayProperty =
  | "time"
  | "response_status"
  | "method"
  | "path"
  | "response_body"
  | "request_id"
  | "host"
  | "request_headers"
  | "request_body"
  | "response_headers";

type LogsContextType = {
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
  selectedLog: Log | null;
  setSelectedLog: (log: Log | null) => void;
  displayProperties: Set<DisplayProperty>;
  toggleDisplayProperty: (property: DisplayProperty) => void;
};

const DEFAULT_DISPLAY_PROPERTIES: DisplayProperty[] = [
  "time",
  "response_status",
  "method",
  "path",
  "response_body",
];

const LogsContext = createContext<LogsContextType | null>(null);

export const LogsProvider = ({ children }: PropsWithChildren) => {
  const [selectedLog, setSelectedLog] = useState<Log | null>(null);
  const [isLive, setIsLive] = useState(false);
  const [displayProperties, setDisplayProperties] = useState<Set<DisplayProperty>>(
    new Set(DEFAULT_DISPLAY_PROPERTIES),
  );

  const toggleDisplayProperty = (property: DisplayProperty) => {
    setDisplayProperties((prev) => {
      const next = new Set(prev);
      if (next.has(property)) {
        next.delete(property);
      } else {
        next.add(property);
      }
      return next;
    });
  };

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
        displayProperties,
        toggleDisplayProperty,
      }}
    >
      {children}
    </LogsContext.Provider>
  );
};

export const useLogsContext = () => {
  const context = useContext(LogsContext);
  if (!context) {
    throw new Error("useLogsContext must be used within a LogsProvider");
  }
  return context;
};

export const isDisplayProperty = (value: string): value is DisplayProperty => {
  return [
    "time",
    "response_status",
    "method",
    "path",
    "response_body",
    "request_id",
    "host",
    "request_headers",
    "request_body",
    "response_headers",
  ].includes(value);
};
