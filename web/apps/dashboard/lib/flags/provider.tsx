"use client";

import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { V2WorkspaceGetFlagsResponseBody } from "@unkey/api/models/components";
import { usePathname } from "next/navigation";
import { type PropsWithChildren, createContext, useContext } from "react";
import * as flags from ".";

type FlagMap = V2WorkspaceGetFlagsResponseBody["data"];
const FlagContext = createContext<ReturnType<typeof useWorkspaceFlags> | null>(null);

function useWorkspaceFlags(orgId: string | null, userId: string | null) {
  const pathname = usePathname();
  const client = useQueryClient();
  const queryKey = ["workspace", "flags", orgId, userId];
  const enabled = Boolean(orgId && userId) && !pathname?.startsWith("/auth");
  const query = useQuery({
    queryKey,
    enabled,
    keepPreviousData: false,
    queryFn: async ({ signal }) => {
      const response = await getUnkeyClient().workspace.getFlags({ signal });
      return response.data;
    },
  });

  return {
    ...query,
    data: enabled ? query.data : undefined,
    isLoading: enabled && query.isLoading,
    cancel: () => client.cancelQueries({ queryKey }),
    async applyUpdate({ slug, value }: { slug: string; value: boolean }) {
      await client.cancelQueries({ queryKey });
      client.setQueryData<FlagMap>(queryKey, (current) => {
        const entry = current?.[slug];
        return entry ? { ...current, [slug]: { ...entry, value } } : current;
      });
    },
  };
}

export function FlagProvider({
  orgId,
  userId,
  children,
}: PropsWithChildren<{
  orgId: string | null;
  userId: string | null;
}>) {
  const value = useWorkspaceFlags(orgId, userId);
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
