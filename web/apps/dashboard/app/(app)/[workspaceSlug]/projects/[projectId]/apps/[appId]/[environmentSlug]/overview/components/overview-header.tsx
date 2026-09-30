"use client";
import { githubUrl } from "@/lib/github-url";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconPlusOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  PageHeader,
  PageHeaderActions,
  PageHeaderContent,
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
import { CreateDeploymentButton } from "../../navigations/create-deployment-button";
import { ProductionCardActionsMenu } from "./production-card-actions-menu";
import { useProductionCard } from "./production-card-context";

export function OverviewHeader({ actions }: { actions?: ReactNode }) {
  const { app } = useAppCurrentDeployment();
  const { environment } = useAppEnvironment();

  return (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle className="flex min-w-0 items-center gap-2.5">
          <span className="truncate">{app?.name ?? "App"}</span>
          <span
            className={cn(
              ENVIRONMENT_BADGE_CLASS,
              "h-5 font-normal tracking-normal capitalize",
              environmentPillClass(environment.kind),
            )}
          >
            {environment.slug}
          </span>
        </PageHeaderTitle>
      </PageHeaderContent>
      {actions && <PageHeaderActions>{actions}</PageHeaderActions>}
    </PageHeader>
  );
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
      <CreateDeploymentButton
        renderTrigger={({ onClick }) => (
          <Button variant="primary" size="sm" onClick={onClick}>
            <IconPlusOutline18 />
            New deployment
          </Button>
        )}
      />
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
