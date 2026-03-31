"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { queryClient } from "@/lib/collections/client";
import { useSettingsHasSaved } from "@/lib/collections/deploy/environment-settings";
import { trpc } from "@/lib/trpc/client";
import { Hammer2, XMark } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { GlowIcon } from "../../components/glow-icon";
import { useProjectData } from "../data-provider";

export function PendingRedeployBanner() {
  const [dismissed, setDismissed] = useState(false);
  const { project, deployments, projectId } = useProjectData();
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const hasSaved = useSettingsHasSaved();
  const visible = hasSaved && !dismissed;

  const currentDeployment = project?.currentDeploymentId
    ? deployments.find((d) => d.id === project.currentDeploymentId)
    : undefined;

  const redeploy = trpc.deploy.deployment.redeploy.useMutation({
    onSuccess: async (data) => {
      if (!currentDeployment) {
        return;
      }
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      router.push(
        `/${workspace.slug}/projects/${currentDeployment.projectId}/deployments/${data.deploymentId}`,
      );
    },
    onError: (error) => {
      toast.error("Redeploy failed", { description: error.message });
    },
  });

  if (!visible || !currentDeployment) {
    return null;
  }

  return (
    <div className="fixed top-6 right-6 z-50 animate-fade-slide-in">
      <div className="relative flex items-start gap-4 rounded-xl border border-gray-4 bg-gray-1 p-4 shadow-lg w-100">
        <button
          type="button"
          onClick={() => setDismissed(true)}
          className="absolute top-3 right-3 text-gray-9 hover:text-gray-11 transition-colors cursor-pointer"
          aria-label="Dismiss"
        >
          <XMark className="size-4" />
        </button>

        <GlowIcon
          icon={<Hammer2 iconSize="sm-medium" className="size-[18px]" />}
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
              redeploy.mutate({ deploymentId: currentDeployment.id });
            }}
          >
            Redeploy
          </Button>
        </div>
      </div>
    </div>
  );
}
