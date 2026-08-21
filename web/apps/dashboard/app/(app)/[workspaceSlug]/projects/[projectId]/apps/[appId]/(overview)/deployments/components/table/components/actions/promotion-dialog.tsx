"use client";

import { type Deployment, collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient, noRetry } from "@/lib/unkey-client";
import { eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { useMutation } from "@tanstack/react-query";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { DeploymentSection } from "./components/deployment-section";
import { DomainsSection } from "./components/domains-section";

type PromotionDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  targetDeployment: Deployment;
  currentDeployment: Deployment;
};

export const PromotionDialog = ({
  isOpen,
  onClose,
  targetDeployment,
  currentDeployment,
}: PromotionDialogProps) => {
  const utils = trpc.useUtils();
  const domains = useLiveQuery(
    (q) =>
      q
        .from({ domain: collection.domains })
        .where(({ domain }) => eq(domain.projectId, currentDeployment.projectId))
        .where(({ domain }) => inArray(domain.sticky, ["environment", "live"]))
        .where(({ domain }) => eq(domain.deploymentId, currentDeployment.id)),
    [currentDeployment.projectId, currentDeployment.id],
  );
  const promote = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.promoteDeployment({ deploymentId }, noRetry),
    onSuccess: () => {
      utils.invalidate();
      toast.success("Promotion completed", {
        description: `Successfully promoted to deployment ${targetDeployment.id}`,
      });
      // hack to revalidate
      try {
        collection.projects.utils.refetch();
        collection.apps.utils.refetch();
        collection.deployments.utils.refetch();
        collection.domains.utils.refetch();
      } catch (error) {
        console.error("Refetch error:", error);
      }

      onClose();
    },
    onError: (error) => {
      toast.error("Promotion failed", {
        description: getErrorMessage(error),
      });
    },
  });

  const handlePromotion = async () => {
    await promote.mutateAsync(targetDeployment.id).catch((error) => {
      console.error("Promotion error:", error);
    });
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Promote to version"
      subTitle="Switch the active deployment to a target stable version"
      footer={
        <Button
          variant="primary"
          size="xlg"
          onClick={handlePromotion}
          disabled={promote.isLoading}
          loading={promote.isLoading}
          className="w-full rounded-lg"
        >
          {`Promote to ${targetDeployment.gitCommitSha ? shortenId(targetDeployment.gitCommitSha) : targetDeployment.id}`}
        </Button>
      }
    >
      <div className="flex flex-col gap-9">
        <DeploymentSection
          title="Current Deployment"
          deployment={currentDeployment}
          isCurrent={true}
        />
        <DomainsSection domains={domains.data} />
        <DeploymentSection
          title="Target Deployment"
          deployment={targetDeployment}
          isCurrent={false}
        />
      </div>
    </DialogContainer>
  );
};
