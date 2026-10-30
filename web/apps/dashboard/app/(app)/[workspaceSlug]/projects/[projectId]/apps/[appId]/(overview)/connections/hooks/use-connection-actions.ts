"use client";

import { showSettingsBanner } from "@/lib/collections/deploy/pending-redeploy";
import { queryKeys } from "@/lib/query-keys";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { match } from "@unkey/match";
import { toast } from "@unkey/ui";
import type { ConnectionOp } from "../model/form";
import type { FormMode } from "../model/mode";
import type { ListedConnection } from "../model/target";

type ConnectionAction = FormMode | "remove";

const TOASTS: Record<ConnectionAction, { loading: string; success: string; error: string }> = {
  add: {
    loading: "Adding connection...",
    success: "Connection added",
    error: "Failed to add connection",
  },
  edit: {
    loading: "Saving connection...",
    success: "Connection saved",
    error: "Failed to save connection",
  },
  remove: {
    loading: "Deleting connection...",
    success: "Connection deleted",
    error: "Failed to delete connection",
  },
};

type ConnectionChange = {
  action: ConnectionAction;
  op: ConnectionOp;
  description?: string;
};

export function removalOf(connection: ListedConnection): ConnectionChange {
  return { action: "remove", op: { type: "delete", id: connection.id } };
}

type ConnectionScope = { projectId: string; appId: string; environmentId: string };

function apply(op: ConnectionOp, { projectId, appId, environmentId }: ConnectionScope) {
  const connections = getUnkeyClient().connections;
  return match(op)
    .with({ type: "create" }, ({ targetAppId, rule, name }) =>
      connections.createConnection({
        project: projectId,
        app: appId,
        environment: environmentId,
        targetApp: targetAppId,
        ...rule,
        name,
      }),
    )
    .with({ type: "update" }, ({ id, rule, name }) =>
      connections.updateConnection({ connectionId: id, ...rule, name }),
    )
    .with({ type: "delete" }, ({ id }) => connections.deleteConnection({ connectionId: id }))
    .exhaustive();
}

export function useConnectionMutation(scope: ConnectionScope) {
  const { projectId, appId } = scope;
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: ({ op }: ConnectionChange) => apply(op, scope),
    onMutate: ({ action }) => ({ toastId: toast.loading(TOASTS[action].loading) }),
    onSuccess: (_result, { action, description }, context) => {
      toast.success(TOASTS[action].success, { id: context?.toastId, description });
      showSettingsBanner();
    },
    onError: (error, { action }, context) => {
      toast.error(TOASTS[action].error, {
        id: context?.toastId,
        description: getErrorMessage(error),
      });
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.connections.app(projectId, appId) });
      void queryClient.invalidateQueries({ queryKey: queryKeys.deployments.connectionPinsAll });
    },
  });

  const run = (change: ConnectionChange, onDone?: () => void) => {
    if (mutation.isLoading) {
      return;
    }
    mutation.mutate(change, { onSuccess: onDone });
  };

  return { pending: mutation.isLoading ? (mutation.variables ?? null) : null, run };
}

export type ConnectionMutation = ReturnType<typeof useConnectionMutation>;
