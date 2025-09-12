"use client";

import { trpc } from "@/lib/trpc/client";
import type { Deployment } from "@/lib/collections";
import { CircleInfo, Cloud, CodeBranch, CodeCommit } from "@unkey/icons";
import { Badge, Button, DialogContainer, toast } from "@unkey/ui";
import { useState } from "react";

type RollbackDialogProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  deployment: Deployment;
  currentDeployment: Deployment;
  hostname?: string;
};

export const RollbackDialog = ({
  isOpen,
  onOpenChange,
  deployment,
  currentDeployment,
  hostname,
}: RollbackDialogProps) => {
  const [isLoading, setIsLoading] = useState(false);

  const utils = trpc.useUtils();
  const rollback = trpc.deploy.rollback.useMutation({
    onSuccess: () => {
      utils.invalidate();
      toast.success("Rollback completed", {
        description: `Successfully rolled back to deployment ${deployment.id}`,
      });
      onOpenChange(false);
    },
    onError: (error) => {
      toast.error("Rollback failed", {
        description: error.message,
      });
    },
    onSettled: () => {
      setIsLoading(false);
    },
  });

  const handleRollback = async () => {
    if (!hostname) {
      toast.error("Missing hostname", {
        description: "Cannot perform rollback without hostname information",
      });
      return;
    }

    setIsLoading(true);
    try {
      await rollback.mutateAsync({
        hostname,
        targetVersionId: deployment.id,
      });
    } catch (error) {
      console.error("Rollback error:", error);
    }
  };

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title="Rollback to version"
      subTitle="Switch the active deployment to a target stable version"
      footer={
        <div className="flex flex-col items-center w-full gap-2">
          <Button
            variant="primary"
            size="xlg"
            onClick={handleRollback}
            disabled={isLoading || rollback.isLoading}
            loading={isLoading || rollback.isLoading}
            className="w-full rounded-lg"
          >
            Rollback to target version
          </Button>
          <div className="text-xs text-gray-9">Rollbacks usually complete within seconds</div>
        </div>
      }
    >
      <div className="space-y-6">
        {/* Current active deployment */}
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-medium text-gray-12">Current active deployment</h3>
            <CircleInfo size="sm-regular" className="text-gray-9" />
          </div>

          <div className="bg-gray-2 border border-gray-6 rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Cloud size="md-regular" className="text-gray-11 bg-gray-3 rounded" />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-12">{currentDeployment.id}</span>
                    <Badge variant="success" className="text-successA-11 font-medium">
                      <div className="flex items-center gap-2">Active</div>
                    </Badge>
                  </div>
                  <div className="text-xs text-gray-11">
                    {currentDeployment?.gitCommitMessage || "Current active deployment"}
                  </div>
                </div>
              </div>
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded text-xs text-gray-11">
                  <CodeBranch size="sm-regular" />
                  <span>{currentDeployment.gitBranch}</span>
                </div>
                <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded text-xs text-gray-11">
                  <CodeCommit size="sm-regular" />
                  <span>{currentDeployment.gitCommitSha}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Target version */}
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <h3 className="text-sm font-medium text-gray-12">Target version</h3>
            <CircleInfo size="sm-regular" className="text-gray-9" />
          </div>

          <div className="bg-gray-2 border border-gray-6 rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Cloud size="md-regular" className="text-gray-11 bg-gray-3 rounded" />
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-12">{deployment.id}</span>
                    <Badge variant="primary" className="text-primaryA-11 font-medium">
                      <div className="flex items-center gap-1">Inactive</div>
                    </Badge>
                  </div>
                  <div className="text-xs text-gray-11">
                    {deployment.gitCommitMessage || "Target deployment"}
                  </div>
                </div>
              </div>
              <div className="flex flex-col gap-1.5">
                <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded text-xs text-gray-11">
                  <CodeBranch size="sm-regular" />
                  <span>{deployment.gitBranch}</span>
                </div>
                <div className="flex items-center gap-1.5 px-2 py-1 bg-gray-3 rounded text-xs text-gray-11">
                  <CodeCommit size="sm-regular" />
                  <span>{deployment.gitCommitSha}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </DialogContainer>
  );
};
