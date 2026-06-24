"use client";

import { githubUrl } from "@/lib/github-url";
import {
  ArrowDottedRotateAnticlockwise,
  ArrowOppositeDirectionY,
  ArrowUpRight,
  Layers3,
  Plus,
  TriangleWarning2,
} from "@unkey/icons";
import { Button, HoverCard, HoverCardContent, HoverCardTrigger } from "@unkey/ui";
import Link from "next/link";
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
          className="font-mono tracking-tight text-base font-semibold text-accent-12 truncate hover:underline decoration-dashed underline-offset-3"
        >
          {primaryDomain.hostname}
        </a>
      ) : (
        <span className="font-mono text-base font-semibold text-gray-9 truncate">
          No domain yet
        </span>
      )}
      {additionalDomains.length > 0 && (
        <HoverCard openDelay={0} closeDelay={100}>
          <HoverCardTrigger asChild>
            <span className="rounded-full px-1.5 py-0.5 bg-grayA-3 text-gray-12 text-[11px] leading-[18px] font-mono tabular-nums shrink-0 cursor-default">
              +{additionalDomains.length}
            </span>
          </HoverCardTrigger>
          <HoverCardContent align="start" className="w-64 p-1">
            <div className="flex flex-col max-h-64 overflow-y-auto">
              {additionalDomains.map((domain) => (
                <a
                  key={domain.hostname}
                  href={domain.url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 font-mono text-[13px] text-gray-12 hover:bg-grayA-3 transition-colors"
                >
                  <span className="truncate">{domain.hostname}</span>
                  <ArrowUpRight iconSize="sm-regular" className="shrink-0 text-gray-9" />
                </a>
              ))}
            </div>
          </HoverCardContent>
        </HoverCard>
      )}
      {addCustomDomainHref && (
        <Button variant="outline" size="sm" asChild className="shrink-0 border-dashed">
          <a href={addCustomDomainHref}>
            <Plus iconSize="sm-regular" />
            Add custom domain
          </a>
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
    logsHref,
    requestsHref,
    rollbackTarget,
    openRollback,
    isRolledBack,
  } = useProductionCard();

  return (
    <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 px-4 py-3 border-b border-gray-4">
      <DomainHero />
      <div className="flex items-center gap-2 shrink-0">
        {diagnostic && (
          <Button variant="outline" size="sm" asChild className="border-errorA-4 text-error-11">
            <Link href={diagnostic.href}>
              <TriangleWarning2 iconSize="sm-regular" />
              {diagnostic.label}
            </Link>
          </Button>
        )}
        <Button variant="outline" size="sm" asChild>
          <Link href={logsHref}>
            <Layers3 iconSize="sm-regular" />
            Logs
          </Link>
        </Button>
        <Button variant="outline" size="sm" asChild>
          <Link href={requestsHref}>
            <ArrowOppositeDirectionY iconSize="sm-regular" />
            Requests
          </Link>
        </Button>
        {!isRolledBack && rollbackTarget && (
          <Button variant="outline" size="sm" onClick={openRollback}>
            <ArrowDottedRotateAnticlockwise iconSize="sm-regular" />
            Instant Rollback
          </Button>
        )}
        <ProductionCardActionsMenu
          deployment={deployment}
          status={status}
          commitUrl={githubUrl.commit(sourceRepo, deployment.gitCommitSha)}
        />
      </div>
    </div>
  );
}
