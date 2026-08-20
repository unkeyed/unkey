"use client";

import { type Deployment, collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { cn } from "@/lib/utils";
import { useMutation } from "@tanstack/react-query";
import { CodeBranch, Cube } from "@unkey/icons";
import { match } from "@unkey/match";
import { Badge, Button, DialogContainer, TimestampInfo, toast } from "@unkey/ui";
import { useEffect, useState } from "react";
import { Avatar } from "../../../components/git-avatar";

type UndoRollbackDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  deployments: Deployment[];
  currentDeploymentId: string;
};

export function UndoRollbackDialog({
  isOpen,
  onClose,
  deployments,
  currentDeploymentId,
}: UndoRollbackDialogProps) {
  const defaultId =
    deployments.find((d) => d.id !== currentDeploymentId)?.id ?? currentDeploymentId;
  const [selectedId, setSelectedId] = useState(defaultId);

  // biome-ignore lint/correctness/useExhaustiveDependencies: seed the default only when the dialog opens, not when the live deployments list refetches
  useEffect(() => {
    if (isOpen) {
      setSelectedId(defaultId);
    }
  }, [isOpen]);

  const promote = useMutation({
    mutationFn: (deploymentId: string) =>
      getUnkeyClient().deployments.promoteDeployment({ deploymentId }),
    onSuccess: () => {
      toast.success("Rollback undone", {
        description: "Automatic production deploys have resumed.",
      });
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
      toast.error("Failed to undo rollback", {
        description: getErrorMessage(error),
      });
    },
  });

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onClose}
      title="Undo rollback?"
      subTitle="Promote a deployment and resume automatic production deploys"
      footer={
        <Button
          variant="primary"
          size="xlg"
          className="w-full rounded-lg"
          loading={promote.isLoading}
          disabled={promote.isLoading}
          onClick={() => promote.mutate(selectedId)}
        >
          Undo rollback
        </Button>
      }
    >
      <div className="flex flex-col gap-3">
        <p className="text-[13px] text-gray-11">
          Choose the deployment to make live. The selected deployment is promoted to production and
          automatic deploys resume.
        </p>
        <div className="flex flex-col gap-2 max-h-[320px] overflow-y-auto">
          {deployments.map((deployment) => (
            <DeploymentOption
              key={deployment.id}
              deployment={deployment}
              isCurrent={deployment.id === currentDeploymentId}
              selected={deployment.id === selectedId}
              onSelect={() => setSelectedId(deployment.id)}
            />
          ))}
        </div>
      </div>
    </DialogContainer>
  );
}

type DeploymentOptionProps = {
  deployment: Deployment;
  isCurrent: boolean;
  selected: boolean;
  onSelect: () => void;
};

function DeploymentOption({ deployment, isCurrent, selected, onSelect }: DeploymentOptionProps) {
  const description = match(deployment.source)
    .with("git", () => (
      <>
        <span className="font-mono text-xs font-semibold text-accent-12 shrink-0">
          {deployment.gitCommitSha ? shortenId(deployment.gitCommitSha) : deployment.id}
        </span>
        {deployment.gitCommitMessage && (
          <span className="text-xs text-grayA-9 truncate">{deployment.gitCommitMessage}</span>
        )}
      </>
    ))
    .with("oci", "unknown", () => (
      <span className="font-mono text-xs font-semibold text-accent-12 shrink-0">
        {deployment.id}
      </span>
    ))
    .exhaustive();

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "w-full text-left rounded-[14px] border p-3 transition-colors",
        selected ? "border-grayA-8 bg-grayA-2" : "border-grayA-4 hover:border-grayA-6",
      )}
    >
      <div className="flex items-start gap-3">
        <span
          className={cn(
            "mt-0.5 size-4 shrink-0 rounded-full border flex items-center justify-center",
            selected ? "border-accent-12" : "border-grayA-6",
          )}
        >
          {selected && <span className="size-2 rounded-full bg-accent-12" />}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0 flex items-baseline gap-2">{description}</div>
            {isCurrent && (
              <Badge variant="success" size="sm" className="shrink-0">
                Current
              </Badge>
            )}
          </div>
          <div className="mt-1.5 flex items-center gap-3 text-xs text-grayA-9 min-w-0">
            <DeploymentSource deployment={deployment} />
            <TimestampInfo
              value={deployment.createdAt}
              displayType="relative"
              className="ml-auto"
            />
          </div>
        </div>
      </div>
    </button>
  );
}

function DeploymentSource({ deployment }: { deployment: Deployment }) {
  return match(deployment.source)
    .with("oci", () => (
      <span className="flex items-center gap-1.5 min-w-0">
        <Cube iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span
          className="truncate"
          title={deployment.requestedImage ?? deployment.resolvedImage ?? undefined}
        >
          {deployment.requestedImage ?? deployment.resolvedImage ?? "OCI image deployment"}
        </span>
      </span>
    ))
    .with("git", () => (
      <>
        {deployment.gitBranch && (
          <span className="flex items-center gap-1.5 min-w-0">
            <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-12" />
            <span className="truncate">{deployment.gitBranch}</span>
          </span>
        )}
        <span className="flex items-center gap-1.5 min-w-0">
          <Avatar
            src={deployment.gitCommitAuthorAvatarUrl}
            alt={deployment.gitCommitAuthorHandle ?? "author"}
            className="size-4"
          />
          {deployment.gitCommitAuthorHandle && (
            <span className="truncate">{deployment.gitCommitAuthorHandle}</span>
          )}
        </span>
      </>
    ))
    .with("unknown", () => (
      <span className="flex items-center gap-1.5 min-w-0">
        <Cube iconSize="sm-regular" className="shrink-0 text-gray-12" />
        <span>Deployment artifact</span>
      </span>
    ))
    .exhaustive();
}
