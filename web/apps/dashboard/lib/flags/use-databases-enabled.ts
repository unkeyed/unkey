"use client";

import { useFlag } from "@/lib/flags/provider";
import { useConnectionsEnabled } from "@/lib/flags/use-connections-enabled";

export function useDatabasesEnabled(): boolean {
  const databases = useFlag("databases");
  const connections = useConnectionsEnabled();
  return databases && connections;
}
