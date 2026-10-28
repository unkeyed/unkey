"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { collection } from "@/lib/collections";
import { DEPLOYMENT_STATUSES } from "@/lib/collections/deploy/deployment-status";
import type { EnvironmentKind } from "@/lib/collections/deploy/environments";
import { showSettingsBanner } from "@/lib/collections/deploy/pending-redeploy";
import { queryKeys } from "@/lib/query-keys";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { and, eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { useQueryClient } from "@tanstack/react-query";
import { match } from "@unkey/match";
import { toast } from "@unkey/ui";
import { type ReactNode, useRef } from "react";
import { isRedeployableDeploymentStatus } from "../../deployments/components/table/components/actions/deployment-action-eligibility";
import { useAppCurrentDeployment } from "../../hooks/use-app-current-deployment";
import { type RedeploySource, useRedeployDeployment } from "../../hooks/use-redeploy-deployment";
import { connectionHostVariable } from "../connection-rules";
import { type ConnectionFormValues, type ConnectionOp, planRemove, planSave } from "../model/form";
import type { ConnectionEnvs, MergedConnection } from "../model/merge";

type Action = "add" | "update" | "remove";

const TOASTS: Record<Action, { loading: string; success: string; error: string }> = {
  add: {
    loading: "Adding connection...",
    success: "Connection added",
    error: "Failed to add connection",
  },
  update: {
    loading: "Saving connection...",
    success: "Connection saved",
    error: "Failed to save connection",
  },
  remove: {
    loading: "Removing connection...",
    success: "Connection removed",
    error: "Failed to remove connection",
  },
};

export type AfterSave = "prompt" | "redeploy";

export type ConnectionActions = {
  save: (
    before: MergedConnection | null,
    values: ConnectionFormValues,
    after: AfterSave,
  ) => Promise<boolean>;
  remove: (connection: MergedConnection) => Promise<boolean>;
  planGate: ReactNode;
};

export function useConnectionActions({
  projectId,
  appId,
  envs,
}: {
  projectId: string;
  appId: string;
  envs: ConnectionEnvs;
}): ConnectionActions {
  const queryClient = useQueryClient();
  const running = useRef(false);
  const previewRedeploySource = usePreviewRedeploySource(projectId, appId, envs.preview?.id);
  const { currentDeployment } = useAppCurrentDeployment();
  const redeploy = useRedeployDeployment();
  const { gated, openPaywall, planGate } = useDeployActionGate();

  const redeploySourceFor = (touched: ReadonlySet<EnvironmentKind>): RedeploySource | undefined => {
    if (touched.has("production")) {
      return currentDeployment;
    }
    return touched.has("preview") ? previewRedeploySource : undefined;
  };

  const redeployNow = (touched: ReadonlySet<EnvironmentKind>) => {
    const source = redeploySourceFor(touched);
    if (!source) {
      showRedeployBanner(touched);
      return;
    }
    if (gated) {
      openPaywall();
      return;
    }
    redeploy.mutate(source);
  };

  const showRedeployBanner = (touched: ReadonlySet<EnvironmentKind>) => {
    if (touched.has("production")) {
      showSettingsBanner();
      return;
    }
    if (touched.has("preview") && previewRedeploySource && envs.preview) {
      showSettingsBanner({
        deployment: previewRedeploySource,
        environmentSlug: envs.preview.slug,
      });
    }
  };

  const apply = (op: ConnectionOp, targetAppId: string) => {
    const connections = getUnkeyClient().connections;
    return match(op)
      .with({ type: "create" }, ({ environmentId, rule, name }) =>
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
  };

  const run = async ({
    action,
    targetAppId,
    ops,
    description,
    after = "prompt",
  }: {
    action: Action;
    targetAppId: string;
    ops: ConnectionOp[];
    description?: string;
    after?: AfterSave;
  }) => {
    if (ops.length === 0) {
      return true;
    }
    if (running.current) {
      return false;
    }
    running.current = true;
    const labels = TOASTS[action];
    const toastId = toast.loading(labels.loading);
    const touched = new Set<EnvironmentKind>();
    let saved = false;
    try {
      for (const op of ops) {
        await apply(op, targetAppId);
        touched.add(op.env);
      }
      toast.success(labels.success, { id: toastId, description });
      saved = true;
      return true;
    } catch (error) {
      toast.error(labels.error, {
        id: toastId,
        description: getErrorMessage(error),
      });
      return false;
    } finally {
      if (saved && after === "redeploy") {
        redeployNow(touched);
      } else {
        showRedeployBanner(touched);
      }
      await queryClient.invalidateQueries({
        queryKey: queryKeys.connections.app(projectId, appId),
      });
      running.current = false;
    }
  };

  return {
    planGate,
    save: (before, values, after) => {
      const ops = planSave(before, values, envs);
      const renamed = ops.some((op) => op.type === "update" && op.name !== undefined);
      return run({
        action: before ? "update" : "add",
        targetAppId: values.targetAppId,
        ops,
        after,
        description: renamed
          ? `Update your code to use ${connectionHostVariable(values.name.trim())}.`
          : undefined,
      });
    },
    remove: (connection) =>
      run({ action: "remove", targetAppId: connection.targetAppId, ops: planRemove(connection) }),
  };
}

function usePreviewRedeploySource(
  projectId: string,
  appId: string,
  environmentId: string | undefined,
) {
  const query = useLiveQuery(
    (q) =>
      environmentId === undefined
        ? null
        : q
            .from({ deployment: collection.deployments })
            .where(({ deployment }) =>
              and(
                eq(deployment.projectId, projectId),
                eq(deployment.appId, appId),
                eq(deployment.environmentId, environmentId),
                inArray(
                  deployment.status,
                  DEPLOYMENT_STATUSES.filter(isRedeployableDeploymentStatus),
                ),
              ),
            )
            .orderBy(({ deployment }) => deployment.createdAt, "desc")
            .limit(1),
    [projectId, appId, environmentId],
  );
  return query.data?.[0];
}
