import { trpc } from "@/lib/trpc/client";
import type { Router } from "@/lib/trpc/routers";
import type { QueryClient } from "@tanstack/react-query";
import { getQueryKey } from "@trpc/react-query";
import type { inferRouterOutputs } from "@trpc/server";

type IdentityListData = inferRouterOutputs<Router>["identity"]["query"];
type CachedIdentity = IdentityListData["identities"][number];
type IdentityInfiniteData = {
  pages: IdentityListData[];
  pageParams: unknown[];
};

const setIdentityListData = (
  queryClient: QueryClient,
  updater: (page: IdentityListData) => IdentityListData,
) => {
  queryClient.setQueriesData<IdentityInfiniteData>(
    { queryKey: getQueryKey(trpc.identity.query) },
    (old) => {
      if (!old?.pages) {
        return old;
      }
      return {
        ...old,
        pages: old.pages.map(updater),
      };
    },
  );
};

export const removeIdentityFromCache = (queryClient: QueryClient, identityId: string) => {
  setIdentityListData(queryClient, (page) => ({
    ...page,
    identities: page.identities.filter((identity) => identity.id !== identityId),
    totalCount: Math.max(0, page.totalCount - 1),
  }));
};

export const updateIdentityInCache = (
  queryClient: QueryClient,
  identityId: string,
  updater: (identity: CachedIdentity) => CachedIdentity,
) => {
  setIdentityListData(queryClient, (page) => ({
    ...page,
    identities: page.identities.map((identity) =>
      identity.id === identityId ? updater(identity) : identity,
    ),
  }));
};
