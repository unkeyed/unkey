"use client";

import { useProjectData } from "../(overview)/data-provider";
import { trpc } from "@/lib/trpc/client";
import { Button } from "@unkey/ui";
import { CircleCheck, CircleXMark, CodeBranch, CodeCommit, Github, ShieldAlert, User } from "@unkey/icons";
import { useParams, useRouter, useSearchParams } from "next/navigation";

export default function AuthorizeDeploymentPage() {
  const params = useParams<{ workspaceSlug: string; projectId: string }>();
  const searchParams = useSearchParams();
  const router = useRouter();
  const { project } = useProjectData();

  const branch = searchParams.get("branch");
  const sha = searchParams.get("sha");
  const sender = searchParams.get("sender");
  const message = searchParams.get("message");

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      router.push(`/${params.workspaceSlug}/projects/${params.projectId}/deployments`);
    },
  });

  const shortSha = sha?.slice(0, 7);

  if (!branch) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center space-y-2">
          <ShieldAlert iconSize="2xl-thin" className="text-warning-9 mx-auto" />
          <h2 className="text-lg font-semibold text-content">Missing branch parameter</h2>
          <p className="text-content-subtle text-sm">
            This page should be accessed from a GitHub check link.
          </p>
        </div>
      </div>
    );
  }

  if (authorize.isSuccess) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center space-y-3">
          <CircleCheck iconSize="2xl-thin" className="text-success-9 mx-auto" />
          <h2 className="text-lg font-semibold text-content">Deployment Authorized</h2>
          <p className="text-content-subtle text-sm">
            The deployment has been authorized and is now in progress.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <div className="w-full max-w-md space-y-6">
        {/* Logo connection indicator */}
        <div className="flex items-center justify-center gap-4">
          <div className="flex items-center justify-center w-12 h-12 rounded-full border border-border bg-background">
            <Github iconSize="xl-thin" />
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-warning-9" />
            <div className="w-8 border-t border-dashed border-warning-9" />
            <div className="w-2 h-2 rounded-full bg-warning-9" />
          </div>
          <div className="flex items-center justify-center w-12 h-12 rounded-full border border-border bg-background">
            <ShieldAlert iconSize="xl-thin" className="text-warning-9" />
          </div>
        </div>

        {/* Title */}
        <div className="text-center space-y-2">
          <h1 className="text-xl font-semibold text-content">Authorization Required</h1>
          <p className="text-sm text-content-subtle">
            An external contributor pushed to{" "}
            <strong className="text-content">{project?.name ?? "this project"}</strong>.
            A team member must authorize this deployment before it can proceed.
          </p>
        </div>

        {/* Commit details */}
        <div className="border border-border rounded-lg divide-y divide-border">
          <div className="flex items-center gap-3 px-4 py-3">
            <CodeBranch iconSize="md-thin" className="text-content-subtle shrink-0" />
            <span className="text-sm text-content-subtle">Branch</span>
            <code className="ml-auto text-sm font-mono text-content bg-background-subtle px-2 py-0.5 rounded">
              {branch}
            </code>
          </div>

          {shortSha && (
            <div className="flex items-center gap-3 px-4 py-3">
              <CodeCommit iconSize="md-thin" className="text-content-subtle shrink-0" />
              <span className="text-sm text-content-subtle">Commit</span>
              <code className="ml-auto text-sm font-mono text-content bg-background-subtle px-2 py-0.5 rounded">
                {shortSha}
              </code>
            </div>
          )}

          {message && (
            <div className="px-4 py-3">
              <p className="text-sm text-content truncate">{message}</p>
            </div>
          )}

          {sender && (
            <div className="flex items-center gap-3 px-4 py-3">
              <User iconSize="md-thin" className="text-content-subtle shrink-0" />
              <span className="text-sm text-content">{sender}</span>
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

        {/* Actions */}
        <div className="flex gap-3">
          <Button
            variant="primary"
            size="xlg"
            className="flex-1"
            loading={authorize.isLoading}
            onClick={() =>
              authorize.mutate({
                projectId: params.projectId,
                branch,
              })
            }
          >
            Authorize Deployment
          </Button>
          <Button
            variant="outline"
            size="xlg"
            className="flex-1"
            disabled={authorize.isLoading}
            onClick={() =>
              router.push(`/${params.workspaceSlug}/projects/${params.projectId}/deployments`)
            }
          >
            Dismiss
          </Button>
        </div>

        <p className="text-xs text-center text-content-subtle">
          Only workspace members can authorize deployments.
        </p>
      </div>
    </div>
  );
}
