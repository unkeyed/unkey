"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import { ShieldAlert } from "@unkey/icons";
import { Button, Dialog, DialogContent } from "@unkey/ui";
import { useProjectData } from "../../../data-provider";

const chipClass =
  "font-mono text-xs bg-gray-3 px-1.5 py-0.5 rounded-[5px] text-gray-12 font-medium";

const chipLinkClass =
  "font-mono text-xs bg-gray-3 px-1.5 py-0.5 rounded-[5px] text-gray-12 font-medium decoration-dotted underline underline-offset-2 hover:bg-gray-4 transition-colors";

type DeploymentApprovalProps = {
  isOpen: boolean;
  onClose: () => void;
  deployment: Deployment;
};

export function DeploymentApproval({ isOpen, onClose, deployment }: DeploymentApprovalProps) {
  const { refetchDeployments, project, environments } = useProjectData();

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      refetchDeployments();
      onClose();
    },
  });

  const sourceRepo = deployment.forkRepositoryFullName || project?.repositoryFullName;

  const prUrl =
    project?.repositoryFullName && deployment.prNumber
      ? `https://github.com/${project.repositoryFullName}/pull/${deployment.prNumber}`
      : null;

  const commitUrl =
    sourceRepo && deployment.gitCommitSha
      ? `https://github.com/${sourceRepo}/commit/${deployment.gitCommitSha}`
      : null;

  const branchUrl =
    sourceRepo && deployment.gitBranch
      ? `https://github.com/${sourceRepo}/tree/${deployment.gitBranch}`
      : null;

  const branchName = deployment.gitBranch ?? "unknown";
  const commitSha = deployment.gitCommitSha?.slice(0, 7) ?? "unknown";
  const environment =
    environments.find((e) => e.id === deployment.environmentId)?.slug ?? "Preview";

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent
        className="max-w-[560px] border-gray-4 rounded-2xl! p-0 gap-0 overflow-hidden drop-shadow-2xl"
        style={{
          background:
            "radial-gradient(circle at 5% 15%, hsl(var(--grayA-3)) 0%, transparent 20%), hsl(var(--gray-1))",
        }}
      >
        <div className="flex flex-col items-center p-10">
          <div className="size-12 rounded-[14px] bg-gray-12 dark:bg-white flex items-center justify-center mb-4 shadow-[0_0_0_6px_hsl(var(--gray-2)),0_0_0_8px_hsl(var(--gray-4))]">
            <ShieldAlert className="text-white dark:text-black size-[22px]" iconSize="md-medium" />
          </div>

          <h1 className="text-[22px] font-bold tracking-tight text-gray-12 mb-2">
            Authorize Fork Deployment
          </h1>

          <p className="text-[14px] leading-relaxed text-gray-11 text-center mb-4 max-w-100">
            An external contributor pushed commit{" "}
            {commitUrl ? (
              <a
                href={commitUrl}
                target="_blank"
                rel="noopener noreferrer"
                className={chipLinkClass}
              >
                {commitSha}
              </a>
            ) : (
              <code className={chipClass}>{commitSha}</code>
            )}{" "}
            on branch{" "}
            {branchUrl ? (
              <a
                href={branchUrl}
                target="_blank"
                rel="noopener noreferrer"
                className={chipLinkClass}
              >
                {branchName}
              </a>
            ) : (
              <code className={chipClass}>{branchName}</code>
            )}{" "}
            targeting the <span className="font-semibold text-gray-12">{environment}</span>{" "}
            environment.
          </p>

          <div className="flex gap-4 mt-0">
            <Button
              variant="primary"
              size="xlg"
              className="px-8"
              loading={authorize.isLoading}
              onClick={() => authorize.mutate({ deploymentId: deployment.id })}
            >
              Approve Deployment
            </Button>
            {prUrl ? (
              <a href={prUrl} target="_blank" rel="noopener noreferrer">
                <Button variant="outline" size="xlg" className="px-7">
                  Review Pull Request
                </Button>
              </a>
            ) : (
              <Button variant="outline" size="xlg" className="px-7" disabled>
                Review Pull Request
              </Button>
            )}
          </div>

          {authorize.error && (
            <div className="mt-4 border border-errorA-4 bg-errorA-2 rounded-lg px-4 py-3">
              <p className="text-sm text-error-11">{authorize.error.message}</p>
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
