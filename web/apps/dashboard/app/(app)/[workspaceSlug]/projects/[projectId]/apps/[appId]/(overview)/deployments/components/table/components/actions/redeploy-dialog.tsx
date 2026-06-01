"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useProjectData } from "../../../../../data-provider";
import { DeploymentSection } from "./components/deployment-section";

type RedeployDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  selectedDeployment: Deployment;
};

export const RedeployDialog = ({ isOpen, onClose, selectedDeployment }: RedeployDialogProps) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { projectId } = useProjectData();

  const redeploy = trpc.deploy.deployment.redeploy.useMutation({
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      onClose();
      router.push(
        `/${workspace.slug}/projects/${selectedDeployment.projectId}/deployments/${data.deploymentId}`,
      );
    },
    onError: (error) => {
      toast.error("Redeploy failed", {
        description: error.message,
      });
    },
  });

  const handleRedeploy = async () => {
    await redeploy
      .mutateAsync({
        deploymentId: selectedDeployment.id,
      })
      .catch((error) => {
        console.error("Redeploy error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Redeploy"
      subTitle="Trigger a fresh build and deployment from the same branch"
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handleRedeploy}
          disabled={redeploy.isLoading}
          loading={redeploy.isLoading}
          className="w-full rounded-lg"
        >
          Redeploy
        </Button>
      }
    >
      <div className="flex flex-col gap-9">
        <DeploymentSection title="Deployment" deployment={selectedDeployment} isCurrent={false} />
      </div>
    </DialogContainer>
  );
};
