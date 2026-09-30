"use client";
import { queryKeys } from "@/lib/query-keys";

import { getUnkeyClient } from "@/lib/unkey-client";
import {
  type InfiniteData,
  useInfiniteQuery,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import type { Unkey } from "@unkey/api";
import { type SessionPage, removeSessionGroup, toSessionPage } from "./sessions";
import { type PortalMutationOptions, toastUnless } from "./use-portal";

const SESSION_PAGE_SIZE = 50;

type RevokeSessionInput = Parameters<Unkey["portal"]["revokeSession"]>[0];
type RevokeSessionResult = Awaited<ReturnType<Unkey["portal"]["revokeSession"]>>["data"];

export function usePortalSessions(portalId: string, search: string) {
  const query = useInfiniteQuery({
    queryKey: queryKeys.portal.sessions(portalId, search),
    queryFn: async ({ pageParam, signal }) => {
      const cursor = typeof pageParam === "string" ? pageParam : undefined;
      const response = await getUnkeyClient().portal.listSessions(
        {
          portal: portalId,
          limit: SESSION_PAGE_SIZE,
          cursor,
          search: search || undefined,
        },
        { signal },
      );
      return toSessionPage(response);
    },
    getNextPageParam: (lastPage) => lastPage.cursor,
    keepPreviousData: true,
  });

  return {
    ...query,
    groups: query.data?.pages.flatMap((page) => page.groups) ?? [],
  };
}

// Ends every session one end user holds on the portal.
export function useRevokePortalSessions(portalId: string, options?: PortalMutationOptions) {
  const queryClient = useQueryClient();

  return useMutation<RevokeSessionResult, unknown, Omit<RevokeSessionInput, "portal">>({
    mutationFn: async (input) => {
      const response = await getUnkeyClient().portal.revokeSession({ ...input, portal: portalId });
      return response.data;
    },
    // Updates the cache instead of refetching: a revoke only changes this end
    // user, and an immediate refetch can hit a replica that still lists them.
    onSuccess: (_data, input) => {
      queryClient.setQueriesData<InfiniteData<SessionPage>>(
        { queryKey: queryKeys.portal.sessionLists(portalId) },
        (data) => removeSessionGroup(data, input.externalId),
      );
    },
    onError: toastUnless(options, "Failed to Revoke Sessions"),
  });
}
