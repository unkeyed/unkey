"use client";

import type { Deployment } from "@/lib/collections";
import { githubUrl } from "@/lib/github-url";
import {
  Github,
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconEarthOutline18,
  IconLayers2Outline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
  PageHeaderDescription,
  PageHeaderTitle,
} from "@unkey/ui";
import { cn } from "cn";
import Link from "next/link";
import type { ReactNode } from "react";
import {
  ENVIRONMENT_BADGE_CLASS,
  environmentPillClass,
} from "../../../components/environment-badge";
import { useAppCurrentDeployment } from "../../../hooks/use-app-current-deployment";
import { useAppEnvironment } from "../../environment-context";
import { ProductionCardActionsMenu } from "./production-card-actions-menu";
import type { CardDomain } from "./production-card-context";
import { useProductionCard } from "./production-card-context";

type HeaderSource = { kind: "git"; repo: string; branch: string } | { kind: "oci"; image: string };

const META_ITEM_CLASS = "inline-flex min-w-0 items-center gap-1.5";
const META_ICON_CLASS = "size-3.5 shrink-0 text-gray-9";

export function OverviewHeader({
  deployment,
  primaryDomain = null,
  actions,
}: {
  deployment?: Deployment;
  primaryDomain?: CardDomain | null;
  actions?: ReactNode;
}) {
  const { app } = useAppCurrentDeployment();
  const { environment } = useAppEnvironment();
  const source = headerSource(app, deployment);

  return (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle className="flex min-w-0 items-center gap-2.5">
          <span className="truncate">{app?.name ?? "App"}</span>
          <span
            className={cn(
              ENVIRONMENT_BADGE_CLASS,
              "capitalize",
              environmentPillClass(environment.kind),
            )}
          >
            {environment.slug}
          </span>
        </PageHeaderTitle>
        {(source || primaryDomain) && (
          <PageHeaderDescription className="flex min-w-0 flex-wrap items-center gap-x-4 gap-y-1">
            {source && <SourceMeta source={source} />}
            {primaryDomain && (
              <a
                href={primaryDomain.url}
                target="_blank"
                rel="noopener noreferrer"
                className={cn(META_ITEM_CLASS, "hover:text-gray-12")}
              >
                <IconEarthOutline18 className={META_ICON_CLASS} />
                <span className="truncate font-mono">{primaryDomain.hostname}</span>
              </a>
            )}
          </PageHeaderDescription>
        )}
      </PageHeaderContent>
      {actions && <PageHeaderActions>{actions}</PageHeaderActions>}
    </PageHeader>
  );
}

function SourceMeta({ source }: { source: HeaderSource }) {
  return match(source)
    .with({ kind: "git" }, ({ repo, branch }) => (
      <a
        href={githubUrl.branch(repo, branch)}
        target="_blank"
        rel="noopener noreferrer"
        className={cn(META_ITEM_CLASS, "hover:text-gray-12")}
      >
        <Github className={META_ICON_CLASS} />
        <span className="truncate font-mono">
          {repo}:{branch}
        </span>
      </a>
    ))
    .with({ kind: "oci" }, ({ image }) => (
      <span className={META_ITEM_CLASS}>
        <IconLayers2Outline18 className={META_ICON_CLASS} />
        <span className="truncate font-mono">{image}</span>
      </span>
    ))
    .exhaustive();
}

function headerSource(
  app: ReturnType<typeof useAppCurrentDeployment>["app"],
  deployment: Deployment | undefined,
): HeaderSource | null {
  if (deployment?.source === "git") {
    const repo = deployment.forkRepositoryFullName || app?.repositoryFullName;
    return repo ? { kind: "git", repo, branch: deployment.gitBranch } : null;
  }
  if (deployment?.source === "oci") {
    const image = deployment.requestedImage ?? deployment.resolvedImage;
    return image ? { kind: "oci", image } : null;
  }
  if (app?.sourceType === "git" && app.repositoryFullName) {
    return { kind: "git", repo: app.repositoryFullName, branch: app.defaultBranch };
  }
  if (app?.sourceType === "oci" && app.imageReference) {
    return { kind: "oci", image: app.imageReference };
  }
  return null;
}

export function ProductionActions() {
  const {
    deployment,
    sourceRepo,
    status,
    diagnostic,
    deploymentHref,
    logsHref,
    requestsHref,
    rollbackTarget,
    openRollback,
    isRolledBack,
  } = useProductionCard();

  return (
    <>
      {diagnostic && (
        <Button
          variant="outline"
          size="sm"
          render={<Link href={diagnostic.href} />}
          className="border-errorA-4 text-error-11"
        >
          <IconTriangleWarningOutline18 />
          {diagnostic.label}
        </Button>
      )}
      {!isRolledBack && rollbackTarget && (
        <Button variant="outline" size="sm" onClick={openRollback}>
          <IconArrowDottedRotateAnticlockwiseOutline18 />
          Instant Rollback
        </Button>
      )}
      <ProductionCardActionsMenu
        deployment={deployment}
        status={status}
        deploymentHref={deploymentHref}
        commitUrl={match(deployment.source)
          .with("git", () => githubUrl.commit(sourceRepo, deployment.gitCommitSha))
          .with("oci", "unknown", () => undefined)
          .exhaustive()}
        logsHref={logsHref}
        requestsHref={requestsHref}
      />
    </>
  );
}
