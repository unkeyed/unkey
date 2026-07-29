"use client";

import type { Deployment } from "@/lib/collections/deploy/deployments";
import { githubUrl } from "@/lib/github-url";
import { trpc } from "@/lib/trpc/client";
import { ShieldAlert } from "@unkey/icons";
import { match } from "@unkey/match";
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

  // A deployment can land in awaiting_approval for two distinct reasons:
  //   1. fork PR — `forkRepositoryFullName` is populated by detectForkRepo
  //   2. operator opt-in via FORCE_DEPLOYMENT_APPROVAL=true — same status,
  //      no fork metadata, often a same-repo push to main
  // `prNumber` is set for same-repo PRs too, so it can't gate this copy.
  const gitMetadata = match(deployment.source)
    .with("git_build", () => {
      const sourceRepo = deployment.forkRepositoryFullName || project?.repositoryFullName;
      return {
        isFork: Boolean(deployment.forkRepositoryFullName),
        prUrl: githubUrl.pull(project?.repositoryFullName, deployment.prNumber),
        commitUrl: githubUrl.commit(sourceRepo, deployment.gitCommitSha),
        branchUrl: githubUrl.branch(sourceRepo, deployment.gitBranch),
        branchName: deployment.gitBranch ?? "unknown",
        commitSha: deployment.gitCommitSha?.slice(0, 7) ?? "unknown",
      };
    })
    .with("docker_image", "unknown", () => null)
    .exhaustive();
  const environment =
    environments.find((e) => e.id === deployment.environmentId)?.slug ?? "Preview";
  const title = match(deployment.source)
    .with("git_build", () =>
      gitMetadata?.isFork ? "Authorize Fork Deployment" : "Authorize Deployment",
    )
    .with("docker_image", () => "Authorize Docker Deployment")
    .with("unknown", () => "Authorize Deployment")
    .exhaustive();
  const description = match(deployment.source)
    .with("git_build", () => (
      <p className="text-[14px] leading-relaxed text-gray-11 text-center mb-4 max-w-100">
        {gitMetadata?.isFork ? "An external contributor pushed commit " : "Commit "}
        {gitMetadata?.commitUrl ? (
          <a
            href={gitMetadata.commitUrl}
            target="_blank"
            rel="noopener noreferrer"
            className={chipLinkClass}
          >
            {gitMetadata.commitSha}
          </a>
        ) : (
          <code className={chipClass}>{gitMetadata?.commitSha}</code>
        )}{" "}
        on branch{" "}
        {gitMetadata?.branchUrl ? (
          <a
            href={gitMetadata.branchUrl}
            target="_blank"
            rel="noopener noreferrer"
            className={chipLinkClass}
          >
            {gitMetadata.branchName}
          </a>
        ) : (
          <code className={chipClass}>{gitMetadata?.branchName}</code>
        )}{" "}
        {gitMetadata?.isFork ? "targeting" : "is awaiting approval before deploying to"} the{" "}
        <span className="font-semibold text-gray-12">{environment}</span> environment.
      </p>
    ))
    .with("docker_image", () => (
      <p className="text-[14px] leading-relaxed text-gray-11 text-center mb-4 max-w-100">
        Docker image{" "}
        <code className={chipClass}>
          {deployment.requestedImage ?? deployment.image ?? "unknown"}
        </code>{" "}
        is awaiting approval before deploying to the{" "}
        <span className="font-semibold text-gray-12">{environment}</span> environment.
      </p>
    ))
    .with("unknown", () => (
      <p className="text-[14px] leading-relaxed text-gray-11 text-center mb-4 max-w-100">
        This deployment is awaiting approval before deploying to the{" "}
        <span className="font-semibold text-gray-12">{environment}</span> environment.
      </p>
    ))
    .exhaustive();

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

          <h1 className="text-[22px] font-bold tracking-tight text-gray-12 mb-2">{title}</h1>

          {description}

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
            {gitMetadata?.prUrl ? (
              <a href={gitMetadata.prUrl} target="_blank" rel="noopener noreferrer">
                <Button variant="outline" size="xlg" className="px-7">
                  Review Pull Request
                </Button>
              </a>
            ) : gitMetadata?.isFork ? (
              // Fork without a PR shouldn't normally happen but render
              // the disabled affordance so the modal layout stays
              // balanced.
              <Button variant="outline" size="xlg" className="px-7" disabled>
                Review Pull Request
              </Button>
            ) : null}
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
