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
} from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useParams, useRouter } from "next/navigation";
import { parseAsString, useQueryStates } from "nuqs";
import { useProjectData } from "../(overview)/data-provider";

const searchParamsParsers = {
  branch: parseAsString.withDefault(""),
  sha: parseAsString.withDefault(""),
  sender: parseAsString.withDefault(""),
  message: parseAsString.withDefault(""),
  repo: parseAsString.withDefault(""),
};

export default function AuthorizeDeploymentPage() {
  const params = useParams<{ workspaceSlug: string; projectId: string }>();
  const router = useRouter();
  const { project } = useProjectData();

  const [{ branch, sha, sender, message, repo }] = useQueryStates(searchParamsParsers);

  const authorize = trpc.deploy.deployment.authorize.useMutation({
    onSuccess: () => {
      router.push(`/${params.workspaceSlug}/projects/${params.projectId}/deployments`);
    },
  });

  const shortSha = sha.slice(0, 7);
  const commitURL = `https://github.com/${repo}/commit/${sha}`;
  const branchURL = `https://github.com/${repo}/tree/${branch}`;
  const senderURL = `https://github.com/${sender}`;

  if (!branch || !sha || !/^[0-9a-f]{40}$/.test(sha)) {
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

  const isStaleCommit = authorize.error?.message?.includes("no longer the HEAD");
  const newHead = isStaleCommit ? parseNewHead(authorize.error?.message) : null;

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

  if (isStaleCommit && newHead) {
    const newAuthorizeURL = `/${params.workspaceSlug}/projects/${params.projectId}/authorize?${new URLSearchParams(
      {
        branch,
        sha: newHead.sha,
        sender: newHead.author,
        message: newHead.message,
        repo,
      },
    ).toString()}`;

    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="w-full max-w-md space-y-6">
          <div className="text-center space-y-3">
            <ShieldAlert iconSize="2xl-thin" className="text-warning-9 mx-auto" />
            <h2 className="text-lg font-semibold text-content">Commit is outdated</h2>
            <p className="text-sm text-content-subtle">
              The branch <strong className="text-content font-mono">{branch}</strong> has new
              commits since this link was created. The latest commit is{" "}
              <a
                href={`https://github.com/${repo}/commit/${newHead.sha}`}
                target="_blank"
                rel="noopener noreferrer"
                className="font-mono text-accent-11 hover:text-accent-12"
              >
                {newHead.sha.slice(0, 7)}
              </a>
              .
            </p>
          </div>
          <div className="flex gap-3">
            <Button
              variant="primary"
              size="xlg"
              className="flex-1"
              onClick={() => {
                authorize.reset();
                router.push(newAuthorizeURL);
              }}
            >
              View Latest Commit
            </Button>
            <Button
              variant="outline"
              size="xlg"
              className="flex-1"
              onClick={() =>
                router.push(`/${params.workspaceSlug}/projects/${params.projectId}/deployments`)
              }
            >
              Dismiss
            </Button>
          </div>
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
            <strong className="text-content">{project?.name ?? "this project"}</strong>. A team
            member must authorize this deployment before it can proceed.
          </p>
        </div>

        {/* Commit details */}
        <div className="border border-border rounded-lg divide-y divide-border">
          <div className="flex items-center gap-3 px-4 py-3">
            <CodeBranch iconSize="md-thin" className="text-content-subtle shrink-0" />
            <span className="text-sm text-content-subtle">Branch</span>
            <a
              href={branchURL}
              target="_blank"
              rel="noopener noreferrer"
              className="ml-auto text-sm font-mono text-accent-11 hover:text-accent-12 bg-background-subtle px-2 py-0.5 rounded"
            >
              {branch}
            </a>
          </div>

          <div className="flex items-center gap-3 px-4 py-3">
            <CodeCommit iconSize="md-thin" className="text-content-subtle shrink-0" />
            <span className="text-sm text-content-subtle">Commit</span>
            <a
              href={commitURL}
              target="_blank"
              rel="noopener noreferrer"
              className="ml-auto text-sm font-mono text-accent-11 hover:text-accent-12 bg-background-subtle px-2 py-0.5 rounded"
            >
              {shortSha}
            </a>
          </div>

          {message && (
            <div className="px-4 py-3">
              <p className="text-sm text-content truncate">{message}</p>
            </div>
          )}

          <div className="flex items-center gap-3 px-4 py-3">
            <User iconSize="md-thin" className="text-content-subtle shrink-0" />
            <a
              href={senderURL}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sm text-accent-11 hover:text-accent-12"
            >
              {sender}
            </a>
          </div>
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
                commitSha: sha,
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

function parseNewHead(
  errorMessage: string | undefined,
): { sha: string; message: string; author: string } | null {
  if (!errorMessage) {
    return null;
  }
  const shaMatch = errorMessage.match(/current_head_sha=([0-9a-f]{40})/);
  const messageMatch = errorMessage.match(
    /current_head_message=(.+?)(?:\s+current_head_author=|$)/,
  );
  const authorMatch = errorMessage.match(/current_head_author=(\S+)/);
  if (!shaMatch) {
    return null;
  }
  return {
    sha: shaMatch[1],
    message: messageMatch?.[1] ?? "",
    author: authorMatch?.[1] ?? "",
  };
}
