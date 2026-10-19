"use client";

import { LoadError } from "@/components/load-error";
import {
  type DeploymentStatus,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import {
  type InstanceSummary,
  instanceLabel,
  instanceStateOf,
} from "@/lib/collections/deploy/instance-status";
import { fromQuery } from "@/lib/loadable/loadable";
import { trpc } from "@/lib/trpc/client";
import { formatCompoundDuration } from "@/lib/utils/metric-formatters";
import {
  IconCheckOutline12,
  IconChevronRightOutline12,
  IconMinusOutline12,
  IconXmarkOutline12,
} from "@unkey/icons";
import { match } from "@unkey/match";
import { Card, Loading } from "@unkey/ui";
import { cn } from "cn";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { toBuildLogLines } from "../(deployment-progress)/build-logs/build-log-lines";
import { DeploymentBuildLogs } from "../(deployment-progress)/build-logs/deployment-build-logs";
import { DeploymentContainerLogsTable } from "../(deployment-progress)/container-logs-table/deployment-container-logs-table";

import { useDeployment } from "../layout-provider";
import { useBuildLogs } from "../use-build-logs";
import type { StepsData } from "../use-deployment-status";
import { LogTicker } from "./log-ticker";
import {
  type ExpandableStageKey,
  type Stage,
  type StageKey,
  type StepResolution,
  resolveStages,
} from "./stages-model";
import { type TickerLine, stageTickerLine } from "./ticker-model";

type StageStatus = StepResolution["status"];

const TICK_MS = 500;
const DEPLOYING_LOGS_POLL_MS = 2_000;
const OPEN_LOGS_POLL_MS = 5_000;

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

function useTickingNow(ticking: boolean): number {
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (!ticking) {
      return;
    }
    const interval = setInterval(() => setNow(Date.now()), TICK_MS);
    return () => clearInterval(interval);
  }, [ticking]);
  return now;
}

function runtimeLogsPollMs(deployingInFlight: boolean, instancesOpen: boolean): number | false {
  if (deployingInFlight) {
    return DEPLOYING_LOGS_POLL_MS;
  }
  return instancesOpen ? OPEN_LOGS_POLL_MS : false;
}

type DeploymentStagesCardProps = {
  steps: StepsData | undefined;
  status: DeploymentStatus;
  instances: InstanceSummary | null;
  domainCount: number;
};

export function DeploymentStagesCard({
  steps,
  status,
  instances,
  domainCount,
}: DeploymentStagesCardProps) {
  const { deployment } = useDeployment();
  const inFlight = isDeploymentInFlight(status);
  const now = useTickingNow(inFlight);
  const stages = resolveStages({
    steps,
    now,
    status,
    isPrebuilt: deployment.source === "oci",
    instances,
    domainCount,
  });

  const failedStage = stages.find((s) => s.resolution.status === "error")?.key ?? null;
  const [choice, setChoice] = useState<{
    madeUnder: StageKey | null;
    open: StageKey | null;
  } | null>(null);
  const open = choice?.madeUnder === failedStage ? choice.open : failedStage;
  const toggle = (key: StageKey) =>
    setChoice({ madeUnder: failedStage, open: open === key ? null : key });

  const deploying = steps?.deploying;
  const deployingInFlight = inFlight && Boolean(deploying) && !deploying?.endedAt;
  const instancesOpen = open === "instances";
  const runtimeLogsQuery = trpc.deploy.deployment.runtimeLogs.useQuery(
    { deploymentId: deployment.id, limit: 50 },
    {
      enabled: deployingInFlight || instancesOpen,
      refetchInterval: runtimeLogsPollMs(deployingInFlight, instancesOpen),
    },
  );

  const buildTicks =
    stages.some((s) => s.key === "build" && s.resolution.status === "started") && open !== "build";
  const buildLogs = useBuildLogs(deployment, { readsToEnd: true, enabled: buildTicks });
  const buildEntries = buildLogs.data?.entries;
  const buildLines = useMemo(() => toBuildLogLines(buildEntries ?? [], "all"), [buildEntries]);
  const tickerSources = {
    buildLines,
    runtimeLogs: runtimeLogsQuery.data?.logs ?? [],
    instances,
  };

  const panel = (key: ExpandableStageKey) =>
    match(key)
      .with("build", () => (
        // A remount on a build failure starts the logs at the last error
        <DeploymentBuildLogs key={failedStage === "build" ? "failed" : "live"} />
      ))
      .with("instances", () => (
        <>
          <InstanceList />
          {match(fromQuery(runtimeLogsQuery))
            .with({ status: "loading" }, () => <DeploymentContainerLogsTable logs={[]} isLoading />)
            .with({ status: "failed" }, ({ retry }) => (
              <LoadError title="Could not load runtime logs" frame="none" onRetry={retry} />
            ))
            .with({ status: "ready" }, ({ data }) => (
              <DeploymentContainerLogsTable logs={data.logs} isLoading={false} />
            ))
            .exhaustive()}
        </>
      ))
      .exhaustive();

  return (
    <Card className="flex flex-col divide-y divide-grayA-4 overflow-hidden">
      {stages.map((stage) => {
        const isOpen = stage.expandable && open === stage.key;
        return (
          <StageRow
            key={stage.key}
            stage={stage}
            ticker={stageTickerLine(stage, tickerSources)}
            open={isOpen}
            onToggle={() => toggle(stage.key)}
          >
            {isOpen && panel(stage.key)}
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
  onToggle: () => void;
  children: ReactNode;
};

function StageRow({ stage, ticker, open, onToggle, children }: StageRowProps) {
  const { label, resolution, expandable } = stage;
  const failed = resolution.status === "error";
  return (
    <div>
      <button
        type="button"
        disabled={!expandable}
        onClick={onToggle}
        aria-expanded={expandable ? open : undefined}
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
        <div className="border-t border-grayA-4 bg-grayA-2">
          {failed && (
            <p className="border-b border-grayA-4 bg-error-2 px-4 py-2.5 font-mono text-xs leading-5 text-error-11">
              {resolution.description}
            </p>
          )}
          {children}
        </div>
      )}
    </div>
  );
}

function InstanceList() {
  const { deployment } = useDeployment();
  if (deployment.instances.length === 0) {
    return null;
  }
  return (
    <ul className="flex flex-col border-b border-grayA-4">
      {deployment.instances.map((instance) => {
        const state = instanceStateOf(instance);
        return (
          <li
            key={instance.id}
            className="flex h-9 items-center gap-3 border-t border-grayA-4 px-4 text-xs first:border-t-0"
          >
            <span className="font-mono text-gray-12">{instance.id}</span>
            <span className="text-gray-10">{instance.region.name}</span>
            <span
              className={cn("ml-auto", state === "unhealthy" ? "text-error-11" : "text-gray-11")}
            >
              {instanceLabel[state]}
            </span>
          </li>
        );
      })}
    </ul>
  );
}
