"use client";

import { trpc } from "@/lib/trpc/client";

/**
 * Resolves the keyspace id behind an API. `keyAuthId` is exposed nowhere in the
 * generated SDK, so this workspace-scoped tRPC query is the only client source.
 *
 * The flags are separate from the id on purpose: an undefined id means "still
 * resolving" (`isLoading`), "the lookup failed" (`isError`), or "this API has no
 * keyspace" (neither flag set). A caller that has to distinguish them cannot do
 * it from the id alone, and the last one is a dead end while the middle one is
 * retryable through `refetch`.
 */
export function useApiKeyAuthId(apiId: string | undefined): {
  keyAuthId: string | undefined;
  isLoading: boolean;
  isError: boolean;
  refetch: () => void;
} {
  const query = trpc.api.queryApiKeyDetails.useQuery(
    { apiId: apiId ?? "" },
    { enabled: Boolean(apiId) },
  );

  return {
    keyAuthId: query.data?.currentApi?.keyAuthId ?? undefined,
    isLoading: Boolean(apiId) && query.isLoading,
    isError: Boolean(apiId) && query.isError,
    refetch: () => {
      void query.refetch();
    },
  };
}
