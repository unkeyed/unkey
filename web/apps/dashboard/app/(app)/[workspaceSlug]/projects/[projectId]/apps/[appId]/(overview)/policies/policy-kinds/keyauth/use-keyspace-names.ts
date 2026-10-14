"use client";

import { trpc } from "@/lib/trpc/client";

export function useKeyspaceNames(): Record<string, string> {
  const { data: keyspaces = {} } = trpc.deploy.environmentSettings.getAvailableKeyspaces.useQuery();
  return Object.fromEntries(Object.entries(keyspaces).map(([id, ks]) => [id, ks.api.name]));
}
