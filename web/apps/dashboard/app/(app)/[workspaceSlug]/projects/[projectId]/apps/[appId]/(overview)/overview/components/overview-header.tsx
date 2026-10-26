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
  Skeleton,
} from "@unkey/ui";
import Link from "next/link";
import type { ReactNode } from "react";
import { CreateDeploymentButton } from "../../navigations/create-deployment-button";
import { OverviewPageTitle } from "./overview-page-title";
import { ProductionCardActionsMenu } from "./production-card-actions-menu";
import { useProductionCard } from "./production-card-context";

export function OverviewHeader({ actions }: { actions?: ReactNode }) {
  return (
    <PageHeader>
      <PageHeaderContent>
        <PageHeaderTitle>
          <OverviewPageTitle />
        </PageHeaderTitle>
      </PageHeaderContent>
      {actions && <PageHeaderActions>{actions}</PageHeaderActions>}
    </PageHeader>
  );
}

function NewDeploymentButton() {
  return (
    <CreateDeploymentButton
      renderTrigger={({ onClick }) => (
        <Button variant="primary" size="md" onClick={onClick}>
          <IconPlusOutline18 />
          New deployment
        </Button>
      )}
    />
  );
}

export function ProductionActionsSkeleton() {
  return (
    <>
      <Skeleton className="h-8 w-36 rounded-md" />
      <Skeleton className="size-8 rounded-md" />
    </>
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
          size="md"
          render={<Link href={diagnostic.href} />}
          className="border-errorA-4 text-error-11"
        >
          <IconTriangleWarningOutline18 />
          {diagnostic.label}
        </Button>
      )}
      {!isRolledBack && rollbackTarget && (
        <Button variant="outline" size="md" onClick={openRollback}>
          <IconArrowDottedRotateAnticlockwiseOutline18 />
          Instant Rollback
        </Button>
      )}
      <NewDeploymentButton />
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
