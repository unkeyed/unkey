"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { getUnkeyClient } from "@/lib/unkey-client";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  CreateLogdrainRequest,
  Logdrain,
  UpdateLogdrainRequest,
} from "@unkey/api/models/components";
import { NotFoundErrorResponse } from "@unkey/api/models/errors";

export const logdrainQueryKeys = {
  workspace: (workspaceId: string) => ["logdrains", workspaceId] as const,
  list: (workspaceId: string) => ["logdrains", workspaceId, "list"] as const,
  detail: (workspaceId: string, id: string) => ["logdrains", workspaceId, "detail", id] as const,
};

export function useLogdrains() {
  const workspace = useWorkspaceNavigation();
  return useQuery({
    queryKey: logdrainQueryKeys.list(workspace.id),
    queryFn: async ({ signal }) => {
      const drains: Logdrain[] = [];
      let cursor: string | undefined;
      do {
        const response = await getUnkeyClient().logdrains.listLogdrains(
          { limit: 100, cursor },
          { signal },
        );
        drains.push(...response.data);
        if (response.pagination.hasMore && !response.pagination.cursor) {
          throw new Error("Log drain API returned a continuation page without a cursor");
        }
        cursor = response.pagination.hasMore ? response.pagination.cursor : undefined;
      } while (cursor);
      return drains;
    },
  });
}

export function useLogdrain(id: string) {
  const workspace = useWorkspaceNavigation();
  const client = useQueryClient();
  return useQuery({
    queryKey: logdrainQueryKeys.detail(workspace.id, id),
    initialData: () =>
      client
        .getQueryData<Logdrain[]>(logdrainQueryKeys.list(workspace.id))
        ?.find((drain) => drain.id === id),
    initialDataUpdatedAt: 0,
    queryFn: async ({ signal }) => {
      try {
        return (await getUnkeyClient().logdrains.getLogdrain({ logdrainId: id }, { signal })).data;
      } catch (error) {
        if (error instanceof NotFoundErrorResponse) {
          return null;
        }
        throw error;
      }
    },
  });
}

type MutationCallbacks = { onSuccess: () => void; onError: (error: unknown) => void };

export function useCreateLogdrainMutation(callbacks: MutationCallbacks) {
  const workspace = useWorkspaceNavigation();
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: CreateLogdrainRequest) =>
      (await getUnkeyClient().logdrains.createLogdrain(input)).data,
    onMutate: () => client.cancelQueries({ queryKey: logdrainQueryKeys.list(workspace.id) }),
    onSuccess: callbacks.onSuccess,
    onError: callbacks.onError,
    onSettled: () =>
      client.invalidateQueries({
        queryKey: logdrainQueryKeys.list(workspace.id),
        refetchType: "all",
      }),
  });
}

export function useUpdateLogdrainMutation(callbacks: MutationCallbacks) {
  const workspace = useWorkspaceNavigation();
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: UpdateLogdrainRequest) =>
      (await getUnkeyClient().logdrains.updateLogdrain(input)).data,
    onMutate: () => client.cancelQueries({ queryKey: logdrainQueryKeys.workspace(workspace.id) }),
    onSuccess: callbacks.onSuccess,
    onError: callbacks.onError,
    onSettled: () =>
      client.invalidateQueries({
        queryKey: logdrainQueryKeys.workspace(workspace.id),
        refetchType: "all",
      }),
  });
}

export function useDeleteLogdrainMutation(callbacks: MutationCallbacks) {
  const workspace = useWorkspaceNavigation();
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (input: { logdrainId: string }) =>
      (await getUnkeyClient().logdrains.deleteLogdrain(input)).data,
    onMutate: () => client.cancelQueries({ queryKey: logdrainQueryKeys.list(workspace.id) }),
    onSuccess: (_data, input) => {
      client.removeQueries({
        queryKey: logdrainQueryKeys.detail(workspace.id, input.logdrainId),
      });
      callbacks.onSuccess();
    },
    onError: callbacks.onError,
    onSettled: () =>
      client.invalidateQueries({
        queryKey: logdrainQueryKeys.list(workspace.id),
        refetchType: "all",
      }),
  });
}

export function useLogdrainDeliveries(id: string) {
  const workspace = useWorkspaceNavigation();
  return useQuery({
    queryKey: [...logdrainQueryKeys.detail(workspace.id, id), "deliveries"],
    queryFn: async ({ signal }) =>
      (await getUnkeyClient().logdrains.getRecentDeliveries({ logdrainId: id }, { signal })).data,
  });
}

export function useLogdrainMetrics(id: string, hours: 1 | 24 | 168) {
  const workspace = useWorkspaceNavigation();
  return useQuery({
    queryKey: [...logdrainQueryKeys.detail(workspace.id, id), "metrics", hours],
    queryFn: async ({ signal }) =>
      (await getUnkeyClient().logdrains.getMetrics({ logdrainId: id, hours }, { signal })).data,
  });
}
