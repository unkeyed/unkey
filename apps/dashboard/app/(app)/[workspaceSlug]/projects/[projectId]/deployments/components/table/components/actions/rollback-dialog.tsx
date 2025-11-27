"use client";
import { type Deployment, collection } from "@/lib/collections";
import { useTRPC } from "@/lib/trpc/client";
import { inArray, useLiveQuery } from "@tanstack/react-db";
import { Button, DialogContainer, toast } from "@unkey/ui";
import { useProject } from "../../../../../layout-provider";
import { DeploymentSection } from "./components/deployment-section";
import { DomainsSection } from "./components/domains-section";

import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

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
  const trpc = useTRPC();
  const queryClient = useQueryClient();

  const {
    collections: { domains: domainCollection },
  } = useProject();
  const domains = useLiveQuery((q) =>
    q
      .from({ domain: domainCollection })
      .where(({ domain }) => inArray(domain.sticky, ["environment", "live"])),
  );

  const rollback = useMutation(
    trpc.deploy.deployment.rollback.mutationOptions({
      onSuccess: () => {
        queryClient.invalidateQueries(trpc.pathFilter());
        toast.success("Rollback completed", {
          description: `Successfully rolled back to deployment ${targetDeployment.id}`,
        });
        // hack to revalidate
        try {
          // @ts-expect-error Their docs say it's here
          collection.projects.utils.refetch();
          // @ts-expect-error Their docs say it's here
          collection.deployments.utils.refetch();
          // @ts-expect-error Their docs say it's here
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
    }),
  );

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
          disabled={rollback.isPending}
          loading={rollback.isPending}
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
