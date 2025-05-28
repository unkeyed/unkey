"use client";

import { type PropsWithChildren, createContext, useContext, useState } from "react";

type LogsContextType = {
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
};

const KeyDetailsLogsContext = createContext<LogsContextType | null>(null);

export const KeyDetailsLogsProvider = ({ children }: PropsWithChildren) => {
  const [isLive, setIsLive] = useState(false);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  return (
    <KeyDetailsLogsContext.Provider
      value={{
        isLive,
        toggleLive,
      }}
    >
      {children}
    </KeyDetailsLogsContext.Provider>
  );
};

export const useKeyDetailsLogsContext = () => {
  const context = useContext(KeyDetailsLogsContext);
  if (!context) {
    throw new Error("useLogsContext must be used within a LogsProvider");
  }
  return context;
};
