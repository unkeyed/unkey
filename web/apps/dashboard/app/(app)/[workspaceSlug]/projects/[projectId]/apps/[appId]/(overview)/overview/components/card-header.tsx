"use client";

import { githubUrl } from "@/lib/github-url";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconPlusOutline18,
  IconTriangleWarningOutline18,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { DomainOverflow } from "../../../components/domain-overflow";
import { ProductionCardActionsMenu } from "./production-card-actions-menu";
import { useProductionCard } from "./production-card-context";

function DomainHero() {
  const { primaryDomain, additionalDomains, addCustomDomainHref } = useProductionCard();
  return (
    <div className="flex items-center gap-2 min-w-0">
      {primaryDomain ? (
        <a
          href={primaryDomain.url}
          target="_blank"
          rel="noopener noreferrer"
          className="font-mono tracking-tight text-base font-semibold text-gray-12 truncate hover:underline decoration-dashed underline-offset-3"
        >
          {primaryDomain.hostname}
        </a>
      ) : (
        <span className="font-mono text-base font-semibold text-gray-9 truncate">
          No domain yet
        </span>
      )}
      <DomainOverflow domains={additionalDomains} />
      {addCustomDomainHref && (
        <Button
          variant="outline"
          size="sm"
          render={<Link href={addCustomDomainHref} />}
          className="shrink-0 border-dashed"
        >
          <IconPlusOutline18 />
          Add custom domain
        </Button>
      )}
    </div>
  );
}

export function ProductionCardHeader() {
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
    <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-3 border-b bg-raised rounded-t-lg">
      <DomainHero />
      <div className="flex items-center gap-2 shrink-0">
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
      </div>
    </div>
  );
}
