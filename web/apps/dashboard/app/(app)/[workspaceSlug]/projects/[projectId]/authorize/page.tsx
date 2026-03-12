"use client";

import { trpc } from "@/lib/trpc/client";
import {
  CircleCheck,
  CircleXMark,
  CodeBranch,
  CodeCommit,
  Github,
  ShieldAlert,
  User,
  XMark,
} from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { parseAsString, useQueryStates } from "nuqs";

const searchParamsParsers = {
  deploymentId: parseAsString.withDefault(""),
};

export default function AuthorizeDeploymentPage() {
  const params = useParams<{ workspaceSlug: string; projectId: string }>();
  const router = useRouter();

  const [{ deploymentId }] = useQueryStates(searchParamsParsers);

  const deployment = trpc.deploy.deployment.getById.useQuery(
    { deploymentId },
    { enabled: !!deploymentId },
  );

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      router.push(`/${params.workspaceSlug}/projects/${params.projectId}/deployments`);
    },
  });

  if (!deploymentId) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center space-y-2">
          <ShieldAlert iconSize="2xl-thin" className="text-warning-9 mx-auto" />
          <h2 className="text-lg font-semibold text-content">Invalid authorization link</h2>
          <p className="text-content-subtle text-sm">
            This page should be accessed from a GitHub commit status link.
          </p>
        </div>
      </div>
    );
  }

  if (deployment.isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center space-y-2">
          <p className="text-sm text-content-subtle">Loading deployment details...</p>
        </div>
      </div>
    );
  }

  if (deployment.error) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center space-y-2">
          <CircleXMark iconSize="2xl-thin" className="text-error-9 mx-auto" />
          <h2 className="text-lg font-semibold text-content">Deployment not found</h2>
          <p className="text-content-subtle text-sm">{deployment.error.message}</p>
        </div>
      </div>
    );
  }

  const data = deployment.data;

  // Already authorized — deployment is no longer awaiting approval
  if (data.status !== "awaiting_approval") {
    const isSuccess = data.status !== "failed";
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="text-center space-y-3">
          {isSuccess ? (
            <CircleCheck iconSize="2xl-thin" className="text-success-9 mx-auto" />
          ) : (
            <CircleXMark iconSize="2xl-thin" className="text-error-9 mx-auto" />
          )}
          <h2 className="text-lg font-semibold text-content">
            {isSuccess ? "Deployment Already Authorized" : "Deployment Failed"}
          </h2>
          <p className="text-content-subtle text-sm">
            {isSuccess
              ? "This deployment has already been authorized and is in progress."
              : "This deployment has failed."}
          </p>
          <Button
            variant="outline"
            size="xlg"
            onClick={() =>
              router.push(`/${params.workspaceSlug}/projects/${params.projectId}/deployments`)
            }
          >
            View Deployments
          </Button>
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

  const shortSha = data.gitCommitSha?.slice(0, 7) ?? "";

  return (
    <div className="flex items-center justify-center min-h-[60vh]">
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
          {data.gitBranch && (
            <div className="flex items-center gap-3 px-4 py-3">
              <CodeBranch iconSize="md-thin" className="text-content-subtle shrink-0" />
              <span className="text-sm text-content-subtle">Branch</span>
              <span className="ml-auto text-sm font-mono text-content bg-background-subtle px-2 py-0.5 rounded">
                {data.gitBranch}
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

          {data.gitCommitMessage && (
            <div className="px-4 py-3">
              <p className="text-sm text-content truncate">{data.gitCommitMessage}</p>
            </div>
          )}

          {data.gitCommitAuthorHandle && (
            <div className="flex items-center gap-3 px-4 py-3">
              {data.gitCommitAuthorAvatarUrl ? (
                <img
                  src={data.gitCommitAuthorAvatarUrl}
                  alt={data.gitCommitAuthorHandle}
                  className="w-5 h-5 rounded-full shrink-0"
                />
              ) : (
                <User iconSize="md-thin" className="text-content-subtle shrink-0" />
              )}
              <span className="text-sm text-content">{data.gitCommitAuthorHandle}</span>
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
            onClick={() => authorize.mutate({ deploymentId })}
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
