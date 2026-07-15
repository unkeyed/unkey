"use client";

import { type ReactNode, createContext, useContext } from "react";
import type { Flags } from "./resolve";

const FlagsContext = createContext<Flags | null>(null);

export function FlagsProvider({ value, children }: { value: Flags; children: ReactNode }) {
  return <FlagsContext.Provider value={value}>{children}</FlagsContext.Provider>;
}

export function useFlag<K extends keyof Flags>(key: K): Flags[K] {
  const ctx = useContext(FlagsContext);
  if (!ctx) {
    throw new Error("useFlag must be used inside FlagsProvider");
  }
  return ctx[key];
}

export function useFlags(): Flags {
  const ctx = useContext(FlagsContext);
  if (!ctx) {
    throw new Error("useFlags must be used inside FlagsProvider");
  }
  return ctx;
}
