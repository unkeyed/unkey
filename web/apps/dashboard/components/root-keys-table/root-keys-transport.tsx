"use client";

import { type PropsWithChildren, createContext, useContext } from "react";

export type RootKeysTransport = "legacy" | "v2";

const RootKeysTransportContext = createContext<RootKeysTransport>("legacy");

export function RootKeysTransportProvider({
  transport,
  children,
}: PropsWithChildren<{ transport: RootKeysTransport }>) {
  return (
    <RootKeysTransportContext.Provider value={transport}>
      {children}
    </RootKeysTransportContext.Provider>
  );
}

export const useRootKeysTransport = () => useContext(RootKeysTransportContext);
