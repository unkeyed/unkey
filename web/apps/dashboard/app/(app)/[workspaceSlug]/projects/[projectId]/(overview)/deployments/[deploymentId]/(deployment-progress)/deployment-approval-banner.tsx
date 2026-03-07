"use client";

import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { ShieldCheck } from "@unkey/icons";
import { Button, toast } from "@unkey/ui";
import { useDeployment } from "../layout-provider";

export function DeploymentApprovalBanner() {
  const { deployment } = useDeployment();
  const utils = trpc.useUtils();

  const approve = trpc.deploy.deployment.approve.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Deployment approved", {
        description: "The deployment will now proceed.",
      });
      try {
        collection.deployments.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }
    },
    onError: (error) => {
      toast.error("Failed to approve deployment", {
        description: error.message,
      });
    },
  });

  const reject = trpc.deploy.deployment.reject.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Deployment rejected", {
        description: "The deployment has been rejected.",
      });
      try {
        collection.deployments.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }
    },
    onError: (error) => {
      toast.error("Failed to reject deployment", {
        description: error.message,
      });
    },
  });

  const isLoading = approve.isLoading || reject.isLoading;

  return (
    <div className="border border-warningA-4 bg-warningA-2 rounded-[14px] p-5 flex flex-col gap-4">
      <div className="flex items-start gap-3">
        <div className="rounded-md bg-warningA-3 p-1.5 mt-0.5">
          <ShieldCheck iconSize="md-regular" className="text-warning-11" />
        </div>
        <div className="flex flex-col gap-1">
          <span className="text-sm font-medium text-warning-11">
            Deployment Authorization Required
          </span>
          <span className="text-xs text-gray-11">
            This deployment was triggered by an external contributor who is not a collaborator on
            the repository. A project member must authorize this deployment before it will proceed.
          </span>
        </div>
      </div>
      <div className="flex gap-2 ml-9">
        <Button
          variant="primary"
          size="sm"
          onClick={() => approve.mutateAsync({ deploymentId: deployment.id })}
          disabled={isLoading}
          loading={approve.isLoading}
          className="px-4"
        >
          Approve Deployment
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={() => reject.mutateAsync({ deploymentId: deployment.id })}
          disabled={isLoading}
          loading={reject.isLoading}
          className="px-4"
        >
          Reject
        </Button>
      </div>
    </div>
  );
}
