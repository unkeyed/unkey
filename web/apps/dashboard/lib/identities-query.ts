"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  Identity,
  V2IdentitiesCreateIdentityRequestBody,
  V2IdentitiesUpdateIdentityRequestBody,
} from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";

const IDENTITY_PAGE_SIZE = 50;

type IdentityPage = {
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
    ...(normalizedSearch && { cacheTime: 0 }),
    onError,
  });

  return {
    ...query,
    identities: query.data?.pages.flatMap((page) => page.identities) ?? [],
  };
}

export function useIdentity(identityId: string) {
  const workspace = useWorkspaceNavigation();

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
      await queryClient.invalidateQueries({
        queryKey: identityQueryKeys.lists(workspace.id),
        refetchType: "all",
      });
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
      await queryClient.cancelQueries({ queryKey: identityQueryKeys.workspace(workspace.id) });
    },
    onSettled: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: identityQueryKeys.lists(workspace.id),
          refetchType: "all",
        }),
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
    onSuccess: (_data, identityId) => {
      queryClient.removeQueries({
        queryKey: identityQueryKeys.detail(workspace.id, identityId),
        exact: true,
      });
    },
    onSettled: async () => {
      await queryClient.invalidateQueries({
        queryKey: identityQueryKeys.lists(workspace.id),
        refetchType: "all",
      });
    },
  });
}
