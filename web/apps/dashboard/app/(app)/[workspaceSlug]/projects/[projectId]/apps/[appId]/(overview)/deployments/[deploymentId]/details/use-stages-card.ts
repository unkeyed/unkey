import { isDeploymentInFlight } from "@/lib/collections/deploy/deployment-status";
import { trpc } from "@/lib/trpc/client";
import { useEffect, useMemo, useState } from "react";
import { toBuildLogLines } from "../(deployment-progress)/build-logs/build-log-lines";
import { useBuildLogsPanel } from "../use-build-logs";
import type { DeploymentPage } from "../use-deployment-page";
import {
  type OpenStage,
  type Stage,
  type StageKey,
  followFailedStage,
  initialOpenStage,
  resolveStages,
  toggleStage,
} from "./stages-model";
import { type TickerSources, stageTickerLine } from "./ticker-model";

const TICK_MS = 500;
const DEPLOYING_LOGS_POLL_MS = 2_000;

export type StagesCardPage = Pick<
  DeploymentPage,
  "deployment" | "steps" | "status" | "instances" | "domains" | "domainCount"
>;

export function useStagesCard({
  deployment,
  steps,
  status,
  instances,
  domainCount,
}: StagesCardPage) {
  const inFlight = isDeploymentInFlight(status);
  const now = useTickingNow(inFlight);
  const stages = resolveStages({
    steps,
    now,
    status,
    isPrebuilt: deployment.source === "oci",
    instances,
    desiredState: deployment.desiredState,
    domainCount,
  });

  const failedStage = stages.find((s) => s.resolution.status === "error")?.key ?? null;
  const [openStage, setOpenStage] = useState<OpenStage>(() => initialOpenStage(failedStage));
  const followed = followFailedStage(openStage, failedStage);
  if (followed !== openStage) {
    setOpenStage(followed);
  }
  const isOpen = (stage: Stage) => stage.expandable && followed.open === stage.key;
  const toggle = (key: StageKey) => setOpenStage((state) => toggleStage(state, key));

  const deploying = steps?.deploying;
  const deployingInFlight = inFlight && Boolean(deploying) && !deploying?.endedAt;
  const runtimeLogs = trpc.deploy.deployment.runtimeLogs.useQuery(
    { deploymentId: deployment.id, limit: 5 },
    { enabled: deployingInFlight, refetchInterval: DEPLOYING_LOGS_POLL_MS },
  );

  const buildStage = stages.find((s) => s.key === "build");
  const buildPanelOpen = buildStage !== undefined && isOpen(buildStage);
  // The ticker needs the newest line at once
  const { logs: buildLogs, panelProps: buildPanel } = useBuildLogsPanel(deployment, {
    status,
    alwaysReadsToEnd: !buildPanelOpen,
    enabled: buildPanelOpen || buildStage?.resolution.status === "started",
  });
  const buildEntries = buildLogs.data?.entries;
  const buildLines = useMemo(() => toBuildLogLines(buildEntries ?? [], "all"), [buildEntries]);
  const tickerSources: TickerSources = {
    buildLines,
    buildLogsFailed: buildLogs.isError && buildEntries === undefined,
    runtimeLogs: runtimeLogs.data?.logs ?? [],
    instances,
  };

  return {
    stages,
    failedStage,
    isOpen,
    toggle,
    buildPanel,
    tickerFor: (stage: Stage) => stageTickerLine(stage, tickerSources),
  };
}

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
