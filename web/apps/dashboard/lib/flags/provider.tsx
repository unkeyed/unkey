"use client";

import { getUnkeyClient } from "@/lib/unkey-client";
import { type UseQueryResult, useQuery, useQueryClient } from "@tanstack/react-query";
import type { V2WorkspaceGetFlagsResponseBody } from "@unkey/api/models/components";
import { usePathname } from "next/navigation";
import { type PropsWithChildren, createContext, useContext } from "react";
import * as flags from ".";

type FlagMap = V2WorkspaceGetFlagsResponseBody["data"];
type FlagContextValue = {
  data: FlagMap | undefined;
  isLoading: boolean;
  error: UseQueryResult<FlagMap>["error"];
  refetch: UseQueryResult<FlagMap>["refetch"];
  cancel: () => Promise<void>;
  setCachedFlagValue: (update: { slug: string; value: boolean }) => Promise<void>;
};
const FlagContext = createContext<FlagContextValue | null>(null);

export function FlagProvider({
  orgId,
  children,
}: PropsWithChildren<{
  orgId: string | null;
}>) {
  const pathname = usePathname();
  const client = useQueryClient();
  const queryKey = ["workspace", "flags", orgId];
  const enabled = Boolean(orgId) && !pathname?.startsWith("/auth");
  const query = useQuery({
    queryKey,
    enabled,
    keepPreviousData: false,
    queryFn: async ({ signal }) => {
      const response = await getUnkeyClient().workspace.getFlags({ signal });
      return response.data;
    },
  });

  const value: FlagContextValue = {
    data: enabled ? query.data : undefined,
    isLoading: enabled && query.isLoading,
    error: query.error,
    refetch: query.refetch,
    cancel: () => client.cancelQueries({ queryKey }),
    async setCachedFlagValue({ slug, value }: { slug: string; value: boolean }) {
      await client.cancelQueries({ queryKey });
      client.setQueryData<FlagMap>(queryKey, (current) => {
        const entry = current?.[slug];
        return entry ? { ...current, [slug]: { ...entry, value } } : current;
      });
    },
  };
  return <FlagContext.Provider value={value}>{children}</FlagContext.Provider>;
}

export function useFlags() {
  const context = useContext(FlagContext);
  if (!context) {
    throw new Error("useFlags must be used within a FlagProvider");
  }
  return context;
}

export function useFlag(key: keyof typeof flags): boolean {
  const { data } = useFlags();
  const flag = flags[key];
  return data?.[flag.slug]?.value ?? flag.defaultValue;
}
