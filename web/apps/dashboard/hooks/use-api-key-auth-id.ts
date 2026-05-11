"use client";

import { trpc } from "@/lib/trpc/client";

// Resolves the keyAuthId for a given API. The Keys sidebar item deep-links
// into /apis/{apiId}/keys/{keyAuthId} so the API leaf needs this to build
// a real href; until it resolves, the item renders disabled.
//
// Lifted up to the sidebar dispatcher so the build function (buildApiLinks)
// stays pure.
export function useApiKeyAuthId(apiId: string | undefined): string | undefined {
  const { data } = trpc.api.queryApiKeyDetails.useQuery(
    { apiId: apiId ?? "" },
    { enabled: !!apiId },
  );
  return data?.currentApi?.keyAuthId ?? undefined;
}
