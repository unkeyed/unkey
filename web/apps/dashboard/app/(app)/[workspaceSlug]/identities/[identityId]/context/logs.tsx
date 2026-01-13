"use client";

import { type PropsWithChildren, createContext, useContext, useState } from "react";

type IdentityLogsContextType = {
  isLive: boolean;
  toggleLive: (value?: boolean) => void;
};

const IdentityDetailsLogsContext = createContext<IdentityLogsContextType | null>(null);

export const IdentityDetailsLogsProvider = ({ children }: PropsWithChildren) => {
  const [isLive, setIsLive] = useState(false);

  const toggleLive = (value?: boolean) => {
    setIsLive((prev) => (typeof value !== "undefined" ? value : !prev));
  };

  return (
    <IdentityDetailsLogsContext.Provider
      value={{
        isLive,
        toggleLive,
      }}
    >
      {children}
    </IdentityDetailsLogsContext.Provider>
  );
};

export const useIdentityDetailsLogsContext = () => {
  const context = useContext(IdentityDetailsLogsContext);
  if (!context) {
    throw new Error(
      "useIdentityDetailsLogsContext must be used within an IdentityDetailsLogsProvider",
    );
  }
  return context;
};
