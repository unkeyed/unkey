"use client";

import { trpc } from "@/lib/trpc/client";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useQuery } from "@tanstack/react-query";

// Resolves a keyspace's display name from both sources: the proxied
// apis.getApi name wins so demo/proxy keyspaces show their public name,
// falling back to the workspace-scoped queryApiKeyDetails name.
export function useApiName(apiId: string): { name: string | undefined; isLoading: boolean } {
  const details = trpc.api.queryApiKeyDetails.useQuery({ apiId }, { enabled: Boolean(apiId) });
  const proxied = useQuery({
    queryKey: ["dashboard-api-proxy", "apis.getApi", apiId],
    enabled: Boolean(apiId),
    queryFn: async () => {
      const response = await getUnkeyClient().apis.getApi({ apiId });
      return response.data;
    },
  });

  const name = proxied.data?.name ?? details.data?.currentApi?.name;
  return { name, isLoading: name === undefined && (details.isLoading || proxied.isLoading) };
}
