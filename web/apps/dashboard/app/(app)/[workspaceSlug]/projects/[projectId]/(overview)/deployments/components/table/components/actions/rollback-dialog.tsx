"use client";

import { type Deployment, collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useProjectData } from "../../../../../data-provider";
import { DeploymentSection } from "./components/deployment-section";
import { DomainsSection } from "./components/domains-section";

type RollbackDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  targetDeployment: Deployment;
  liveDeployment: Deployment;
};

export const RollbackDialog = ({
  isOpen,
  onClose,
  targetDeployment,
  liveDeployment,
}: RollbackDialogProps) => {
  const utils = trpc.useUtils();

  const { projectId } = useProjectData();
  const domains = useLiveQuery(
    (q) =>
      q
        .from({ domain: collection.domains })
        .where(({ domain }) => eq(domain.projectId, projectId))
        .where(({ domain }) => inArray(domain.sticky, ["environment", "live"])),
    [projectId],
  );

  const rollback = trpc.deploy.deployment.rollback.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Rollback completed", {
        description: `Successfully rolled back to deployment ${targetDeployment.id}`,
      });
      // hack to revalidate
      try {
        collection.projects.utils.refetch();
        collection.deployments.utils.refetch();
        collection.domains.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }

      onClose();
    },
    onError: (error) => {
      toast.error("Rollback failed", {
        description: error.message,
      });
    },
  });

  const handleRollback = async () => {
    await rollback
      .mutateAsync({
        targetDeploymentId: targetDeployment.id,
      })
      .catch((error) => {
        console.error("Rollback error:", error);
      });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Rollback to version"
      subTitle="Switch the active deployment to a target stable version"
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handleRollback}
          disabled={rollback.isLoading}
          loading={rollback.isLoading}
          className="w-full rounded-lg"
        >
          Rollback to target version
        </Button>
      }
    >
      <div className="space-y-9">
        <DeploymentSection
          title="Live Deployment"
          deployment={liveDeployment}
          isLive={true}
          showSignal={true}
        />
        <DomainsSection domains={domains.data} />
        <DeploymentSection title="Target Deployment" deployment={targetDeployment} isLive={false} />
      </div>
    </DialogContainer>
  );
};
