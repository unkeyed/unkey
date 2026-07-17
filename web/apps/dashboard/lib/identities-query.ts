"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { getUnkeyClient } from "@/lib/unkey-client";
import {
  type InfiniteData,
  type QueryClient,
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import type {
  Identity,
  V2IdentitiesCreateIdentityRequestBody,
  V2IdentitiesUpdateIdentityRequestBody,
} from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";

const IDENTITY_PAGE_SIZE = 50;

export type IdentityPage = {
  identities: Identity[];
  cursor?: string;
};

export const identityQueryKeys = {
  all: ["identities"] as const,
  workspace: (workspaceId: string) => [...identityQueryKeys.all, workspaceId] as const,
  lists: (workspaceId: string) => [...identityQueryKeys.workspace(workspaceId), "list"] as const,
  list: (workspaceId: string, search: string) =>
    [...identityQueryKeys.lists(workspaceId), search] as const,
  details: (workspaceId: string) =>
    [...identityQueryKeys.workspace(workspaceId), "detail"] as const,
  detail: (workspaceId: string, identityId: string) =>
    [...identityQueryKeys.details(workspaceId), identityId] as const,
};

function mergeIdentityUpdate(
  cachedIdentity: Identity,
  updatedIdentity: Identity,
  input: V2IdentitiesUpdateIdentityRequestBody,
): Identity {
  return {
    ...updatedIdentity,
    meta: input.meta === undefined ? cachedIdentity.meta : updatedIdentity.meta,
    ratelimits:
      input.ratelimits === undefined ? cachedIdentity.ratelimits : updatedIdentity.ratelimits,
  };
}

export function replaceIdentityInPages(
  data: InfiniteData<IdentityPage>,
  updatedIdentity: Identity,
  input: V2IdentitiesUpdateIdentityRequestBody,
): InfiniteData<IdentityPage> {
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      identities: page.identities.map((identity) =>
        identity.id === updatedIdentity.id
          ? mergeIdentityUpdate(identity, updatedIdentity, input)
          : identity,
      ),
    })),
  };
}

export function removeIdentityFromPages(
  data: InfiniteData<IdentityPage>,
  identityId: string,
): InfiniteData<IdentityPage> {
  return {
    ...data,
    pages: data.pages.map((page) => ({
      ...page,
      identities: page.identities.filter((identity) => identity.id !== identityId),
    })),
  };
}

function findIdentityInListCache(
  queryClient: QueryClient,
  workspaceId: string,
  identityId: string,
): { identity: Identity; updatedAt: number } | undefined {
  const queries = queryClient.getQueriesData<InfiniteData<IdentityPage>>(
    identityQueryKeys.lists(workspaceId),
  );
  let cachedIdentity: { identity: Identity; updatedAt: number } | undefined;

  for (const [queryKey, data] of queries) {
    for (const page of data?.pages ?? []) {
      const identity = page.identities.find((candidate) => candidate.id === identityId);
      if (identity) {
        const candidate = {
          identity,
          updatedAt: queryClient.getQueryState(queryKey)?.dataUpdatedAt ?? 0,
        };
        if (!cachedIdentity || candidate.updatedAt > cachedIdentity.updatedAt) {
          cachedIdentity = candidate;
        }
      }
    }
  }

  return cachedIdentity;
}

function updateIdentityListCaches(
  queryClient: QueryClient,
  workspaceId: string,
  update: (data: InfiniteData<IdentityPage>) => InfiniteData<IdentityPage>,
) {
  queryClient.setQueriesData<InfiniteData<IdentityPage>>(
    identityQueryKeys.lists(workspaceId),
    (data) => (data ? update(data) : data),
  );
}

export function useIdentities({
  search = "",
  onError,
}: { search?: string; onError?: (error: unknown) => void } = {}) {
  const workspace = useWorkspaceNavigation();
  const normalizedSearch = search.trim();
  const query = useInfiniteQuery({
    queryKey: identityQueryKeys.list(workspace.id, normalizedSearch),
    queryFn: async ({ pageParam, signal }) => {
      const cursor = typeof pageParam === "string" ? pageParam : undefined;
      const response = await getUnkeyClient().identities.listIdentities(
        {
          limit: IDENTITY_PAGE_SIZE,
          cursor,
          search: normalizedSearch || undefined,
        },
        { signal },
      );
      const { data, pagination } = response.result;

      if (pagination.hasMore && !pagination.cursor) {
        throw new Error("Identity API returned a continuation page without a cursor");
      }

      return {
        identities: data,
        cursor: pagination.hasMore ? pagination.cursor : undefined,
      } satisfies IdentityPage;
    },
    getNextPageParam: (lastPage) => lastPage.cursor,
    onError,
  });

  return {
    ...query,
    identities: query.data?.pages.flatMap((page) => page.identities) ?? [],
  };
}

export function useIdentity(identityId: string) {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();
  const cachedIdentity = findIdentityInListCache(queryClient, workspace.id, identityId);

  return useQuery({
    queryKey: identityQueryKeys.detail(workspace.id, identityId),
    queryFn: async ({ signal }) => {
      try {
        const response = await getUnkeyClient().identities.getIdentity(
          { identity: identityId },
          { signal },
        );
        return response.data;
      } catch (error) {
        if (error instanceof NotFoundErrorResponse) {
          return null;
        }
        throw error;
      }
    },
    initialData: cachedIdentity?.identity,
    initialDataUpdatedAt: cachedIdentity?.updatedAt,
    staleTime: 0,
  });
}

export function useCreateIdentityMutation() {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: V2IdentitiesCreateIdentityRequestBody) => {
      const response = await getUnkeyClient().identities.createIdentity(input);
      return { identityId: response.data.identityId, externalId: input.externalId };
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: identityQueryKeys.lists(workspace.id) });
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({ queryKey: identityQueryKeys.lists(workspace.id) });
    },
  });
}

export function useUpdateIdentityMutation() {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: V2IdentitiesUpdateIdentityRequestBody) => {
      const response = await getUnkeyClient().identities.updateIdentity(input);
      return response.data;
    },
    onMutate: async () => {
      await Promise.all([
        queryClient.cancelQueries({ queryKey: identityQueryKeys.lists(workspace.id) }),
        queryClient.cancelQueries({ queryKey: identityQueryKeys.details(workspace.id) }),
      ]);
    },
    onSuccess: (identity, input) => {
      updateIdentityListCaches(queryClient, workspace.id, (data) =>
        replaceIdentityInPages(data, identity, input),
      );
      queryClient.setQueryData<Identity | null>(
        identityQueryKeys.detail(workspace.id, identity.id),
        (cachedIdentity) => {
          if (!cachedIdentity) {
            return cachedIdentity;
          }
          return mergeIdentityUpdate(cachedIdentity, identity, input);
        },
      );
    },
    onSettled: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: identityQueryKeys.lists(workspace.id) }),
        queryClient.invalidateQueries({ queryKey: identityQueryKeys.details(workspace.id) }),
      ]);
    },
  });
}

export function useDeleteIdentityMutation() {
  const workspace = useWorkspaceNavigation();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (identityId: string) => {
      await getUnkeyClient().identities.deleteIdentity({ identity: identityId });
      return identityId;
    },
    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: identityQueryKeys.lists(workspace.id) });
    },
    onSuccess: (identityId) => {
      updateIdentityListCaches(queryClient, workspace.id, (data) =>
        removeIdentityFromPages(data, identityId),
      );
      queryClient.removeQueries({
        queryKey: identityQueryKeys.detail(workspace.id, identityId),
        exact: true,
      });
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({ queryKey: identityQueryKeys.lists(workspace.id) });
    },
  });
}
