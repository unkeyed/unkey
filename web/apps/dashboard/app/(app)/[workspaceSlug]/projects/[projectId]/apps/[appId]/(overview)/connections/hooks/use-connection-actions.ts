"use client";

import { showSettingsBanner } from "@/lib/collections/deploy/pending-redeploy";
import { queryKeys } from "@/lib/query-keys";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { match } from "@unkey/match";
import { toast } from "@unkey/ui";
import type { FormMode } from "../model/mode";
import type { ConnectionOp } from "../model/plan";
import { type RunResult, failureDescription, needsRedeploy, runInOrder } from "../model/run";

type ConnectionAction = FormMode | "remove";

const TOASTS: Record<
  ConnectionAction,
  { loading: string; success: string; error: string; done: string }
> = {
  add: {
    loading: "Adding connection...",
    success: "Connection added",
    error: "Failed to add connection",
    done: "added",
  },
  edit: {
    loading: "Saving connection...",
    success: "Connection saved",
    error: "Failed to save connection",
    done: "saved",
  },
  remove: {
    loading: "Deleting connection...",
    success: "Connection deleted",
    error: "Failed to delete connection",
    done: "deleted",
  },
};

type ConnectionChange = {
  action: ConnectionAction;
  ops: readonly ConnectionOp[];
  description?: string;
};

type ConnectionScope = { projectId: string; appId: string };

function apply(op: ConnectionOp, { projectId, appId }: ConnectionScope) {
  const connections = getUnkeyClient().connections;
  return match(op)
    .with({ type: "create" }, ({ environment, targetAppId, rule, name }) =>
      connections.createConnection({
        project: projectId,
        app: appId,
        environment: environment.id,
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
    mutationFn: async ({ ops }: ConnectionChange) => {
      const result = await runInOrder(ops, (op) => apply(op, scope));
      if (needsRedeploy(result.applied)) {
        showSettingsBanner();
      }
      return result;
    },
    onMutate: ({ action }) => ({ toastId: toast.loading(TOASTS[action].loading) }),
    onSuccess: ({ applied, failure }, { action, description }, context) => {
      const toasts = TOASTS[action];
      if (failure) {
        toast.error(toasts.error, {
          id: context?.toastId,
          description: failureDescription(
            applied,
            failure.op,
            toasts.done,
            getErrorMessage(failure.error),
          ),
        });
        return;
      }
      toast.success(toasts.success, { id: context?.toastId, description });
    },
    onError: (error, { action }, context) => {
      toast.error(TOASTS[action].error, {
        id: context?.toastId,
        description: getErrorMessage(error),
      });
    },
    onSettled: () =>
      Promise.all([
        queryClient.invalidateQueries({ queryKey: queryKeys.connections.app(projectId, appId) }),
        queryClient.invalidateQueries({ queryKey: queryKeys.deployments.connectionPinsAll }),
      ]),
  });

  const run = (change: ConnectionChange, onResult?: (result: RunResult) => void) => {
    if (mutation.isLoading) {
      return;
    }
    mutation.mutate(change, { onSuccess: onResult });
  };

  return { pending: mutation.isLoading ? (mutation.variables ?? null) : null, run };
}

export type ConnectionMutation = ReturnType<typeof useConnectionMutation>;
