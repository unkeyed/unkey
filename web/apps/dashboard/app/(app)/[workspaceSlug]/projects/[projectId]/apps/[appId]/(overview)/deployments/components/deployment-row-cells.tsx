"use client";

import { Avatar } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/git-avatar";
import type { Deployment, Environment } from "@/lib/collections";
import { DEPLOYMENT_STATUS_LABELS } from "@/lib/collections/deploy/deployment-status";
import { githubUrl } from "@/lib/github-url";
import { shortenId } from "@/lib/shorten-id";
import { cn } from "@/lib/utils";
import {
  BracketsCurly,
  CircleQuestion,
  CodeBranch,
  CodeCommit,
  Github,
  Laptop2,
  Layers2,
  SquareTerminal,
} from "@unkey/icons";
import type { IconProps } from "@unkey/icons/src/props";
import { InfoTooltip, TimestampInfo } from "@unkey/ui";
import type { Route } from "next";
import dynamic from "next/dynamic";
import Link from "next/link";
import type { FC, ReactNode } from "react";
import { DeploymentStatusIndicator } from "../../../components/deployment-status-dot";
import { imageTag } from "./image-reference";
import { ActionColumnSkeleton } from "./table/components/skeletons";

const DeploymentListTableActions = dynamic(
  () =>
    import("./table/components/actions/deployment-list-table-action.popover.constants").then(
      (mod) => mod.DeploymentListTableActions,
    ),
  {
    loading: () => <ActionColumnSkeleton />,
    ssr: false,
  },
);

const CHIP_CLASS =
  "inline-flex h-5.5 min-w-0 items-center gap-1.5 rounded-md border border-grayA-5 px-2 text-xs leading-none text-accent-12";

function Interactive({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <span className={cn("relative z-20 inline-flex items-center", className)}>{children}</span>
  );
}

export function IdChip({ deployment, href }: { deployment: Deployment; href: Route }) {
  return (
    <InfoTooltip
      content={DEPLOYMENT_STATUS_LABELS[deployment.status]}
      variant="inverted"
      position={{ side: "top" }}
      asChild
    >
      <Link
        href={href}
        className={cn(CHIP_CLASS, "relative z-20 font-mono transition-colors hover:bg-grayA-2")}
      >
        <DeploymentStatusIndicator status={deployment.status} />
        {shortenId(deployment.id)}
      </Link>
    </InfoTooltip>
  );
}

type Origin = { icon: FC<IconProps>; label: string; tooltip: string };

const ORIGINS: Record<Deployment["trigger"], Origin | "git"> = {
  github: "git",
  unknown: "git",
  cli: { icon: SquareTerminal, label: "Unkey CLI", tooltip: "Deployed via the Unkey CLI" },
  api: { icon: BracketsCurly, label: "Unkey API", tooltip: "Deployed via API (root key)" },
  dashboard: { icon: Laptop2, label: "via Dashboard", tooltip: "Deployed via dashboard" },
  unkey: { icon: CircleQuestion, label: "Unkey Team", tooltip: "Deployed by the Unkey team" },
};

function nonGitOrigin(deployment: Deployment): Origin | undefined {
  const origin = ORIGINS[deployment.trigger];
  return origin === "git" ? undefined : origin;
}

export function OriginCell({ deployment }: { deployment: Deployment }) {
  const origin = nonGitOrigin(deployment);
  if (!origin) {
    return null;
  }
  const Icon = origin.icon;

  return (
    <InfoTooltip
      content={origin.tooltip}
      variant="inverted"
      position={{ side: "top" }}
      triggerClassName="relative z-20 flex min-w-0 items-center gap-2"
    >
      <Icon iconSize="sm-regular" className="shrink-0 text-gray-9" />
      <span className="truncate font-mono text-[13px] text-accent-12">{origin.label}</span>
    </InfoTooltip>
  );
}

export function SourceChip({
  deployment,
  repoFullName,
}: {
  deployment: Deployment;
  repoFullName: string | null;
}) {
  const origin = nonGitOrigin(deployment);
  if (origin) {
    const Icon = origin.icon;
    return (
      <span className={CHIP_CLASS}>
        <Icon iconSize="sm-regular" className="shrink-0 text-gray-9" />
        <span className="truncate font-mono" title={origin.label}>
          {origin.label}
        </span>
      </span>
    );
  }

  const href = githubUrl.deployment({
    repoFullName,
    forkRepoFullName: deployment.forkRepositoryFullName,
    prNumber: deployment.prNumber,
    sha: deployment.gitCommitSha,
  });
  if (!href) {
    return null;
  }
  const label = deployment.prNumber ? `#${deployment.prNumber}` : "Source";

  return (
    <Interactive className="min-w-0">
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className={cn(CHIP_CLASS, "transition-colors hover:bg-grayA-2")}
      >
        <Github iconSize="sm-regular" className="shrink-0 text-gray-9" />
        <span className="truncate" title={label}>
          {label}
        </span>
      </a>
    </Interactive>
  );
}

export function BranchCell({
  branch,
  repoFullName,
}: {
  branch: string;
  repoFullName?: string | null;
}) {
  const href = githubUrl.branch(repoFullName, branch);
  const text = (
    <span className="truncate font-mono text-[13px] text-accent-12" title={branch}>
      {branch}
    </span>
  );

  return (
    <span className="flex min-w-0 items-center gap-2">
      <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-9" />
      {href ? (
        <Interactive className="min-w-0">
          <a
            href={href}
            target="_blank"
            rel="noopener noreferrer"
            className="flex min-w-0 hover:underline decoration-dashed underline-offset-3"
          >
            {text}
          </a>
        </Interactive>
      ) : (
        text
      )}
    </span>
  );
}

export function CommitSha({
  deployment,
  repoFullName,
}: {
  deployment: Deployment;
  repoFullName: string | null;
}) {
  if (!deployment.gitCommitSha) {
    return null;
  }
  const href = githubUrl.commit(
    deployment.forkRepositoryFullName || repoFullName,
    deployment.gitCommitSha,
  );
  const body = (
    <>
      <CodeCommit iconSize="sm-regular" className="shrink-0 text-gray-9" />
      <span className="font-mono text-xs text-accent-12">
        {deployment.gitCommitSha.slice(0, 7)}
      </span>
    </>
  );

  if (!href) {
    return <span className="inline-flex items-center gap-1.5">{body}</span>;
  }

  return (
    <Interactive>
      <a
        href={href}
        target="_blank"
        rel="noopener noreferrer"
        className="inline-flex items-center gap-1.5 hover:underline decoration-dashed underline-offset-3"
      >
        {body}
      </a>
    </Interactive>
  );
}

export function ImageRef({ image }: { image: string }) {
  return (
    <span className="flex min-w-0 items-center gap-1.5" title={image}>
      <Layers2 iconSize="sm-regular" className="shrink-0 text-gray-9" />
      <span className="truncate font-mono text-xs text-accent-12">{imageTag(image)}</span>
    </span>
  );
}

export function AuthorCell({
  deployment,
  withHandle = false,
}: {
  deployment: Deployment;
  withHandle?: boolean;
}) {
  if (nonGitOrigin(deployment)) {
    return null;
  }

  return (
    <span className="flex shrink-0 items-center gap-2">
      <Avatar
        src={deployment.gitCommitAuthorAvatarUrl}
        alt={deployment.gitCommitAuthorHandle ?? "Author"}
      />
      {withHandle && deployment.gitCommitAuthorHandle && (
        <span className="max-w-28 truncate text-[13px] text-accent-12">
          {deployment.gitCommitAuthorHandle}
        </span>
      )}
    </span>
  );
}

export function RowTime({ value }: { value: number }) {
  return (
    <Interactive className="shrink-0">
      <TimestampInfo
        value={value}
        displayType="relative"
        side="left"
        align="center"
        className="text-[13px] text-gray-9"
      />
    </Interactive>
  );
}

export function RowMenu({
  deployment,
  environment,
  currentDeployment,
  isRolledBack,
}: {
  deployment: Deployment;
  environment: Environment | undefined;
  currentDeployment: Deployment | undefined;
  isRolledBack: boolean;
}) {
  return (
    <Interactive>
      <DeploymentListTableActions
        selectedDeployment={deployment}
        environment={environment}
        currentDeployment={currentDeployment}
        isRolledBack={isRolledBack}
      />
    </Interactive>
  );
}
