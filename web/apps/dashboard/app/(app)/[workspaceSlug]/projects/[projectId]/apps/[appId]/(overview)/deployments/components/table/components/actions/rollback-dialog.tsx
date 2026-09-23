"use client";

import {
  useAppId,
  useProjectData,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import { getDomainPriority } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/domain-priority";
import { Avatar } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/git-avatar";
import { type Deployment, collection } from "@/lib/collections";
import { deploymentTitle } from "@/lib/collections/deploy/deployment-title";
import { rollbackCandidates } from "@/lib/collections/deploy/rollback";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { and, eq, inArray, useLiveQuery } from "@tanstack/react-db";
import { useMutation } from "@tanstack/react-query";
import { IconChevronDownOutline12 } from "@unkey/icons";
import { Badge, Button, DialogContainer, InfoTooltip, TimestampInfo, toast } from "@unkey/ui";
import { cn } from "cn";
import { type ReactNode, useId, useState } from "react";
import { RollbackDeploymentPair } from "./components/rollback-deployment-pair";

type RollbackDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  targetDeployment: Deployment;
  currentDeployment: Deployment;
};

export function RollbackDialog({
  isOpen,
  onClose,
  targetDeployment,
  currentDeployment,
}: RollbackDialogProps) {
  const { projectId, deployments, customDomains, awaitLiveDeployment } = useProjectData();
  const [selected, setSelected] = useState(targetDeployment);
  const [picking, setPicking] = useState(false);
  const pickerId = useId();

  const target = deployments.find((d) => d.id === selected.id) ?? selected;
  const others = rollbackCandidates(deployments, currentDeployment).filter(
    (d) => d.id !== selected.id,
  );
  const appId = useAppId();
  const domains = useLiveQuery(
    (q) =>
      q
        .from({ domain: collection.domains })
        .where(({ domain }) =>
          and(
            eq(domain.projectId, projectId),
            eq(domain.appId, appId),
            eq(domain.environmentId, currentDeployment.environmentId),
          ),
        )
        .where(({ domain }) => inArray(domain.sticky, ["environment", "live"])),
    [projectId, appId, currentDeployment.environmentId],
  );
  const { primary, additional } = getDomainPriority({
    domains: domains.data ?? [],
    customDomains,
    environmentId: currentDeployment.environmentId,
    deploymentId: currentDeployment.id,
    currentDeploymentId: currentDeployment.id,
  });

  const rollback = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.rollbackDeployment({ deploymentId }),
    onSuccess: (_result, deploymentId) => {
      awaitLiveDeployment({ deploymentId, rolledBack: true });
      toast.success("Rolled back", {
        description: "Production traffic moved to the selected deployment.",
      });
      onClose();
    },
    onError: (error) => {
      toast.error("Rollback failed", { description: getErrorMessage(error) });
    },
  });

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Instant rollback"
      footer={
        <div className="flex w-full items-center justify-end gap-2">
          <Button variant="outline" size="md" onClick={onClose} disabled={rollback.isLoading}>
            Cancel
          </Button>
          <Button
            variant="primary"
            size="md"
            loading={rollback.isLoading}
            disabled={rollback.isLoading}
            onClick={() => rollback.mutate(target.id)}
          >
            Roll back
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-5">
        <div className="flex flex-col gap-2">
          <Label>
            Rolling back{" "}
            {primary ? (
              <span className="font-medium text-gray-12">{primary.hostname}</span>
            ) : (
              "production"
            )}
            {additional.length > 0 && (
              <InfoTooltip
                variant="inverted"
                position={{ side: "top" }}
                triggerClassName="ml-1.5 inline-flex align-middle"
                content={
                  <span className="flex flex-col gap-0.5">
                    {additional.map((d) => (
                      <span key={d.id}>{d.hostname}</span>
                    ))}
                  </span>
                }
              >
                <Badge variant="secondary" size="sm">
                  +{additional.length}
                  <span className="sr-only"> more domains</span>
                </Badge>
              </InfoTooltip>
            )}
          </Label>
          <RollbackDeploymentPair current={currentDeployment} target={target} />
          {others.length > 0 && (
            <div className="flex flex-col rounded-lg border">
              <button
                type="button"
                onClick={() => setPicking((p) => !p)}
                aria-expanded={picking}
                aria-controls={pickerId}
                className="flex cursor-pointer items-center justify-center gap-1.5 px-3 py-2 text-[13px] font-medium text-gray-12 transition-colors hover:bg-grayA-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-grayA-7"
              >
                Choose another deployment
                <IconChevronDownOutline12
                  className={cn("size-3 text-gray-9 transition-transform", picking && "rotate-180")}
                />
              </button>
              {picking && (
                <ul id={pickerId} className="max-h-56 overflow-y-auto border-t">
                  {others.map((d) => (
                    <li
                      key={d.id}
                      className="relative flex items-center gap-3 border-b px-3 py-2 transition-colors last:border-b-0 hover:bg-grayA-2"
                    >
                      <button
                        type="button"
                        aria-label={`Select ${deploymentTitle(d)}`}
                        onClick={() => {
                          setSelected(d);
                          setPicking(false);
                        }}
                        className="absolute inset-0 z-10 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-grayA-7"
                      />
                      <span className="min-w-0 flex-1 truncate text-[13px] text-gray-12">
                        {deploymentTitle(d)}
                      </span>
                      <CommitMeta deployment={d} />
                      <span className="relative z-20 shrink-0">
                        <TimestampInfo
                          value={d.createdAt}
                          displayType="relative"
                          className="text-xs text-gray-11"
                        />
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          )}
        </div>

        <p className="text-xs text-gray-11">
          Automatic production deploys pause until you undo the rollback.
        </p>
      </div>
    </DialogContainer>
  );
}

function Label({ children }: { children: ReactNode }) {
  return <h3 className="text-[13px] text-gray-11">{children}</h3>;
}

function CommitMeta({ deployment }: { deployment: Deployment }) {
  return (
    <span className="flex shrink-0 items-center gap-1.5">
      {deployment.gitCommitSha && (
        <span className="font-mono text-xs text-gray-11">
          {deployment.gitCommitSha.slice(0, 7)}
        </span>
      )}
      {deployment.gitCommitAuthorHandle && (
        <Avatar
          src={deployment.gitCommitAuthorAvatarUrl}
          alt={deployment.gitCommitAuthorHandle}
          className="size-4"
        />
      )}
    </span>
  );
}
