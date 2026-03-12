"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import {
  CircleXMark,
  CodeBranch,
  CodeCommit,
  Github,
  ShieldAlert,
  User,
  XMark,
} from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useProjectData } from "../../../data-provider";

export function DeploymentApproval({ deployment }: { deployment: Deployment }) {
  const { refetchDeployments } = useProjectData();

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      refetchDeployments();
    },
  });

  const shortSha = deployment.gitCommitSha?.slice(0, 7) ?? "";

  return (
    <div className="flex items-center justify-center min-h-[40vh]">
      <div className="w-full max-w-md space-y-6">
        {/* Logo connection indicator */}
        <div className="flex items-center justify-center gap-4">
          <div className="flex items-center justify-center w-12 h-12 rounded-full border border-border bg-background">
            <Github iconSize="xl-thin" />
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-6 border-t border-dashed border-warning-9" />
            <div className="flex items-center justify-center w-6 h-6 rounded-full bg-warning-3 border border-warning-6">
              <XMark iconSize="sm-regular" className="text-warning-9" />
            </div>
            <div className="w-6 border-t border-dashed border-warning-9" />
          </div>
          <div className="flex items-center justify-center w-12 h-12 rounded-full border border-border bg-background">
            <ShieldAlert iconSize="xl-thin" className="text-warning-9" />
          </div>
        </div>

        {/* Title */}
        <div className="text-center space-y-2">
          <h1 className="text-xl font-semibold text-content">Authorization Required</h1>
          <p className="text-sm text-content-subtle">
            An external contributor pushed a commit. A team member must authorize this deployment
            before it can proceed.
          </p>
        </div>

        {/* Commit details */}
        <div className="border border-border rounded-lg divide-y divide-border">
          {deployment.gitBranch && (
            <div className="flex items-center gap-3 px-4 py-3">
              <CodeBranch iconSize="md-thin" className="text-content-subtle shrink-0" />
              <span className="text-sm text-content-subtle">Branch</span>
              <span className="ml-auto text-sm font-mono text-content bg-background-subtle px-2 py-0.5 rounded">
                {deployment.gitBranch}
              </span>
            </div>
          )}

          {shortSha && (
            <div className="flex items-center gap-3 px-4 py-3">
              <CodeCommit iconSize="md-thin" className="text-content-subtle shrink-0" />
              <span className="text-sm text-content-subtle">Commit</span>
              <span className="ml-auto text-sm font-mono text-content bg-background-subtle px-2 py-0.5 rounded">
                {shortSha}
              </span>
            </div>
          )}

          {deployment.gitCommitMessage && (
            <div className="px-4 py-3">
              <p className="text-sm text-content truncate">{deployment.gitCommitMessage}</p>
            </div>
          )}

          {deployment.gitCommitAuthorHandle && (
            <div className="flex items-center gap-3 px-4 py-3">
              {deployment.gitCommitAuthorAvatarUrl ? (
                <img
                  src={deployment.gitCommitAuthorAvatarUrl}
                  alt={deployment.gitCommitAuthorHandle}
                  className="w-5 h-5 rounded-full shrink-0"
                />
              ) : (
                <User iconSize="md-thin" className="text-content-subtle shrink-0" />
              )}
              <span className="text-sm text-content">{deployment.gitCommitAuthorHandle}</span>
            </div>
          )}
        </div>

        {/* Error */}
        {authorize.error && (
          <div className="flex items-start gap-2 p-3 bg-error-3 border border-error-6 rounded-lg">
            <CircleXMark iconSize="md-thin" className="text-error-9 mt-0.5 shrink-0" />
            <p className="text-sm text-error-11">{authorize.error.message}</p>
          </div>
        )}

        {/* Action */}
        <Button
          variant="primary"
          size="xlg"
          className="w-full"
          loading={authorize.isLoading}
          onClick={() => authorize.mutate({ deploymentId: deployment.id })}
        >
          Authorize Deployment
        </Button>

        <p className="text-xs text-center text-content-subtle">
          Only workspace members can authorize deployments.
        </p>
      </div>
    </div>
  );
}
