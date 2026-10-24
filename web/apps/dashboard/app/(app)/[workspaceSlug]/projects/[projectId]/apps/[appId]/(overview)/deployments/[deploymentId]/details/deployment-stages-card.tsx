"use client";

import { isPrebuilt } from "@/lib/collections/deploy/deployments";
import { formatCompoundDuration } from "@/lib/utils/metric-formatters";
import {
  IconCheckOutline12,
  IconChevronRightOutline12,
  IconMinusOutline12,
  IconXmarkOutline12,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Card, Loading, Skeleton } from "@unkey/ui";
import { cn } from "cn";
import { type ReactNode, useEffect, useId, useRef } from "react";
import { DeploymentBuildLogs } from "../(deployment-progress)/build-logs/deployment-build-logs";
import { FailureActions } from "../(deployment-progress)/failure-actions";
import type { DisplayDomain } from "../../../../components/display-domain";
import { DomainListItem } from "../../../../components/domain-list-item";
import { useDeployment } from "../layout-provider";
import { InstancesDiagram } from "./instances-diagram";
import { LogTicker } from "./log-ticker";
import type { Stage, StageKey, StepResolution } from "./stages-model";
import type { TickerLine } from "./ticker-model";
import { type StagesCardPage, useStagesCard } from "./use-stages-card";

type StageStatus = StepResolution["status"];

const stageGlyph: Record<StageStatus, ReactNode> = {
  completed: (
    <span className="grid size-4 place-items-center rounded-full bg-success-9 text-white">
      <IconCheckOutline12 className="size-2.5" />
    </span>
  ),
  started: <Loading size={14} />,
  error: (
    <span className="grid size-4 place-items-center rounded-full bg-error-9 text-white">
      <IconXmarkOutline12 className="size-2.5" />
    </span>
  ),
  pending: <span className="size-4 rounded-full border-[1.5px] border-dashed border-gray-8" />,
  skipped: (
    <span className="grid size-4 place-items-center rounded-full bg-grayA-3 text-gray-9">
      <IconMinusOutline12 className="size-2.5" />
    </span>
  ),
};

const stageStatusLabel: Record<StageStatus, string> = {
  completed: "Complete",
  started: "In progress",
  error: "Failed",
  pending: "Waiting",
  skipped: "Skipped",
};

export function DeploymentStagesCard({ page }: { page: StagesCardPage }) {
  const { stages, failedStage, isOpen, toggle, buildPanel, scrollTarget, tickerFor } =
    useStagesCard(page);

  const panel = (key: StageKey, status: StageStatus) =>
    match(key)
      .with("build", () =>
        isPrebuilt(page.deployment) ? null : (
          // A remount on a build failure starts the logs at the last error
          <DeploymentBuildLogs key={failedStage === "build" ? "failed" : "live"} {...buildPanel} />
        ),
      )
      .with("instances", () => <InstancesDiagram stage={status} />)
      .with("domains", () => (
        <DomainList domains={page.domains.all} loading={page.domainCount === null} />
      ))
      .exhaustive();

  return (
    <Card className="flex flex-col divide-y divide-grayA-4 overflow-hidden">
      {stages.map((stage) => {
        const open = isOpen(stage);
        return (
          <StageRow
            key={stage.key}
            stage={stage}
            ticker={tickerFor(stage)}
            open={open}
            scrollTo={stage.key === scrollTarget}
            onToggle={() => toggle(stage.key)}
          >
            {open && panel(stage.key, stage.resolution.status)}
          </StageRow>
        );
      })}
    </Card>
  );
}

type StageRowProps = {
  stage: Stage;
  ticker: TickerLine | null;
  open: boolean;
  scrollTo: boolean;
  onToggle: () => void;
  children: ReactNode;
};

function StageRow({ stage, ticker, open, scrollTo, onToggle, children }: StageRowProps) {
  const { label, resolution, expandable } = stage;
  const failed = resolution.status === "error";
  const panelId = useId();
  const rowRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (scrollTo) {
      rowRef.current?.scrollIntoView({ block: "start" });
    }
  }, [scrollTo]);
  return (
    <div ref={rowRef}>
      <button
        type="button"
        disabled={!expandable}
        onClick={onToggle}
        aria-expanded={expandable ? open : undefined}
        aria-controls={expandable && open ? panelId : undefined}
        className="flex w-full items-center gap-3 px-4 py-2.5 text-left text-xs transition-colors focus-visible:outline-hidden focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-gray-7 disabled:cursor-default enabled:hover:bg-grayA-2"
      >
        <IconChevronRightOutline12
          className={cn(
            "size-3 shrink-0 text-gray-9 transition-transform duration-200 ease-out motion-reduce:transition-none",
            open && "rotate-90",
            !expandable && "opacity-0",
          )}
        />
        <span className="w-36 shrink-0 text-sm font-medium text-gray-12">{label}</span>
        <span className="min-w-0 flex-1">
          <LogTicker
            line={open ? null : ticker}
            fallback={
              <span className={cn("block truncate", failed ? "text-error-11" : "text-gray-11")}>
                {resolution.description}
              </span>
            }
          />
        </span>
        {resolution.duration !== undefined && (
          <span className="font-mono tabular-nums text-gray-10">
            {formatCompoundDuration(resolution.duration)}
          </span>
        )}
        <span className="grid size-4 shrink-0 place-items-center">
          {stageGlyph[resolution.status]}
          <span className="sr-only">{stageStatusLabel[resolution.status]}</span>
        </span>
      </button>
      {open && (
        <div id={panelId} className="border-t border-grayA-4 bg-grayA-2">
          {failed && <StageFailure stage={stage.key} description={resolution.description} />}
          {children}
        </div>
      )}
    </div>
  );
}

const DOMAIN_SKELETON_ROWS = ["a", "b"];

function DomainList({ domains, loading }: { domains: readonly DisplayDomain[]; loading: boolean }) {
  return (
    <ul className="flex flex-col divide-y divide-grayA-4">
      {loading
        ? DOMAIN_SKELETON_ROWS.map((id) => (
            <li key={id} className="flex h-9 items-center gap-2 px-4">
              <Skeleton className="h-3 w-48 rounded" />
            </li>
          ))
        : domains.map((d) => <DomainListItem key={d.id} domain={d} className="h-9 px-4" />)}
    </ul>
  );
}

function StageFailure({ stage, description }: { stage: StageKey; description: string }) {
  const { deployment } = useDeployment();
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-grayA-4 bg-error-2 px-4 py-2.5">
      <p className="min-w-0 flex-1 font-mono text-xs leading-5 text-error-11">{description}</p>
      <span className="flex shrink-0 items-center gap-2">
        <FailureActions deployment={deployment} stage={stage} error={description} />
      </span>
    </div>
  );
}
