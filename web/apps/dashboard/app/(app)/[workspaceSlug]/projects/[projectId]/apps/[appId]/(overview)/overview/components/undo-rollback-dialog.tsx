"use client";

import { type Deployment, collection } from "@/lib/collections";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { CodeBranch } from "@unkey/icons";
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

  useEffect(() => {
    if (isOpen) {
      setSelectedId(defaultId);
    }
  }, [isOpen, defaultId]);

  const promote = trpc.deploy.deployment.promote.useMutation({
    onSuccess: () => {
      toast.success("Rollback undone", {
        description: "Automatic production deploys have resumed.",
      });
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
      toast.error("Failed to undo rollback", {
        description: error.message,
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
          onClick={() => promote.mutate({ targetDeploymentId: selectedId })}
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
            <div className="min-w-0 flex items-baseline gap-2">
              <span className="font-mono text-xs font-semibold text-accent-12 shrink-0">
                {deployment.gitCommitSha ? shortenId(deployment.gitCommitSha) : deployment.id}
              </span>
              {deployment.gitCommitMessage && (
                <span className="text-xs text-grayA-9 truncate">{deployment.gitCommitMessage}</span>
              )}
            </div>
            {isCurrent && (
              <Badge variant="success" size="sm" className="shrink-0">
                Current
              </Badge>
            )}
          </div>
          <div className="mt-1.5 flex items-center gap-3 text-xs text-grayA-9">
            <span className="flex items-center gap-1.5 min-w-0">
              <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-12" />
              <span className="truncate">{deployment.gitBranch}</span>
            </span>
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
