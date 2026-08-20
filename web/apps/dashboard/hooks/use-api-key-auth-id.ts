"use client";

import { trpc } from "@/lib/trpc/client";

/**
 * Resolves the keyspace id behind an API. `keyAuthId` is exposed nowhere in the
 * generated SDK, so this workspace-scoped tRPC query is the only client source.
 *
 * The loading flag is separate from the id on purpose: an undefined id means
 * "still resolving", "the query failed", or "this API has no keyspace", and a
 * caller that has to distinguish them cannot do it from the id alone.
 */
export function useApiKeyAuthId(apiId: string | undefined): {
  keyAuthId: string | undefined;
  isLoading: boolean;
} {
  const query = trpc.api.queryApiKeyDetails.useQuery(
    { apiId: apiId ?? "" },
    { enabled: Boolean(apiId) },
  );

  return {
    keyAuthId: query.data?.currentApi?.keyAuthId ?? undefined,
    isLoading: Boolean(apiId) && query.isLoading,
  };
}
