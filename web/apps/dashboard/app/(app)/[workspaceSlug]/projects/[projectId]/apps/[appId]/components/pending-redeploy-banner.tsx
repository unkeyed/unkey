"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { queryClient } from "@/lib/collections/client";
import {
  dismissSettingsBanner,
  useSettingsBannerVisible,
} from "@/lib/collections/deploy/environment-settings";
import { routes } from "@/lib/navigation/routes";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { cn } from "@/lib/utils";
import { useMutation } from "@tanstack/react-query";
import { Hammer2, XMark } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useProjectData } from "../(overview)/data-provider";
import { GlowIcon } from "../components/glow-icon";

export function PendingRedeployBanner() {
  const { project, deployments, projectId } = useProjectData();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const visible = useSettingsBannerVisible();
  const currentDeploymentId = project?.currentDeploymentId;

  const currentDeployment = currentDeploymentId
    ? deployments.find((d) => d.id === currentDeploymentId)
    : undefined;

  const show = visible && !!currentDeployment;

  const redeploy = useMutation({
    mutationFn: async (deployment: {
      id: string;
      projectId: string;
      appId: string;
      environmentId: string;
    }) => {
      const res = await getUnkeyClient().deployments.createDeployment({
        idempotencyKey: crypto.randomUUID(),
        v2DeploymentsCreateDeploymentRequestBody: {
          project: deployment.projectId,
          app: deployment.appId,
          environment: deployment.environmentId,
          deployment: { deploymentId: deployment.id },
        },
      });
      return { deploymentId: res.result.data.deploymentId };
    },
    onSuccess: async (data) => {
      if (!currentDeployment) {
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      router.push(
        routes.projects.apps.deployment({
          workspaceSlug: workspace.slug,
          projectId: currentDeployment.projectId,
          appId: currentDeployment.appId,
          deploymentId: data.deploymentId,
        }),
      );
      dismissSettingsBanner();
    },
    onError: (error) => {
      toast.error("Redeploy failed", { description: getErrorMessage(error) });
    },
  });

  useEffect(
    function getDismissedAutomatically() {
      if (!visible || !currentDeploymentId) {
        return;
      }
      const timer = setTimeout(() => dismissSettingsBanner(), 10_000);
      return () => {
        clearTimeout(timer);
      };
    },
    [visible, currentDeploymentId],
  );

  return (
    <div
      aria-hidden={!show}
      inert={!show || undefined}
      className={cn(
        "fixed top-6 right-6 z-50 transition-[transform,opacity] duration-300 ease-out",
        show ? "translate-x-0 opacity-100" : "translate-x-[calc(100%+24px)] opacity-0",
      )}
    >
      <div className="relative flex items-start gap-4 rounded-xl border border-gray-4 bg-gray-1 p-4 shadow-lg w-100">
        <button
          type="button"
          onClick={() => dismissSettingsBanner()}
          className="absolute top-3 right-3 text-gray-9 hover:text-gray-11 transition-colors cursor-pointer"
          aria-label="Dismiss"
        >
          <XMark className="size-4" />
        </button>

        <GlowIcon
          icon={<Hammer2 iconSize="sm-medium" className="size-4.5" />}
          className="w-9 h-9 shrink-0"
        />

        <div className="flex flex-col gap-3 flex-1 min-w-0">
          <div className="flex flex-col gap-1 pr-5">
            <span className="text-sm font-semibold text-gray-12 leading-5">Changes detected</span>
            <span className="text-xs text-gray-11 leading-4">
              Redeploy to apply your latest changes to production.
            </span>
          </div>
          <Button
            variant="primary"
            size="md"
            className="w-full"
            disabled={redeploy.isLoading}
            loading={redeploy.isLoading}
            onClick={() => {
              if (gated) {
                openPaywall();
                return;
              }
              if (currentDeployment) {
                redeploy.mutate(currentDeployment);
              }
            }}
          >
            Redeploy
          </Button>
        </div>
      </div>
      {planGate}
    </div>
  );
}
