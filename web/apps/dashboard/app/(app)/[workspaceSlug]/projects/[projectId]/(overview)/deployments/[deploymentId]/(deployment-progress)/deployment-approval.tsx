"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { trpc } from "@/lib/trpc/client";
import { CircleXMark, Github, ShieldAlert, XMark } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useProjectData } from "../../../data-provider";

export function DeploymentApproval({ deployment }: { deployment: Deployment }) {
  const { refetchDeployments, project } = useProjectData();

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      refetchDeployments();
    },
  });

  const prUrl =
    project?.repositoryFullName && deployment.prNumber
      ? `https://github.com/${project.repositoryFullName}/pull/${deployment.prNumber}`
      : null;

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

        {/* PR Link */}
        {prUrl && (
          <div className="border border-border rounded-lg">
            <div className="flex items-center gap-3 px-4 py-3">
              <Github iconSize="md-thin" className="text-content-subtle shrink-0" />
              <a
                href={prUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-accent-11 hover:underline"
              >
                Pull Request #{deployment.prNumber}
              </a>
            </div>
          </div>
        )}

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
