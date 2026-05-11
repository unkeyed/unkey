"use client";

import { trpc } from "@/lib/trpc/client";

export function useApiKeyAuthId(apiId: string | undefined): string | undefined {
  const { data } = trpc.api.queryApiKeyDetails.useQuery(
    { apiId: apiId ?? "" },
    { enabled: !!apiId },
  );
  return data?.currentApi?.keyAuthId ?? undefined;
}
