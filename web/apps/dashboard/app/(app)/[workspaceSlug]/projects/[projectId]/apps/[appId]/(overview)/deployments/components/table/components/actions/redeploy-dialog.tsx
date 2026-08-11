"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { match } from "@unkey/match";
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

  const redeploy = useMutation({
    mutationFn: async () => {
      const res = await getUnkeyClient().deployments.createDeployment({
        project: selectedDeployment.projectId,
        app: selectedDeployment.appId,
        environment: selectedDeployment.environmentId,
        deployment: { deploymentId: selectedDeployment.id },
      });
      return { deploymentId: res.data.deploymentId };
    },
    onSuccess: async (data) => {
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      onClose();
      router.push(
        routes.projects.apps.deployment({
          workspaceSlug: workspace.slug,
          projectId: selectedDeployment.projectId,
          appId: selectedDeployment.appId,
          deploymentId: data.deploymentId,
        }),
      );
    },
    onError: (error) => {
      toast.error("Redeploy failed", {
        description: getErrorMessage(error),
      });
    },
  });

  const handleRedeploy = async () => {
    await redeploy.mutateAsync().catch((error) => {
      console.error("Redeploy error:", error);
    });
  };

  const subtitle = match(selectedDeployment.source)
    .with("git", () => "Trigger a fresh build and deployment from the same commit")
    .with("docker", () => "Create a deployment from the same resolved Docker image")
    .with("unknown", () => "Create a new deployment from this deployment")
    .exhaustive();

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Redeploy"
      subTitle={subtitle}
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
