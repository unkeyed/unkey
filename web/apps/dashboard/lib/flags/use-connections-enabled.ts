"use client";

import { useFlag } from "@/lib/flags/provider";

export function useConnectionsEnabled(): boolean {
  const connections = useFlag("connections");
  const appV2 = useFlag("appV2");
  return connections && appV2;
}
