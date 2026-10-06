"use client";

import {
  ProjectDataProvider,
  useProjectData,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/data-provider";
import {
  type DisplayDomain,
  getDomainPriority,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/components/domain-priority";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import type { Route } from "next";
import { type ReactNode, createContext, useContext, useEffect, useState } from "react";
import type { SourceKind } from "../../wizard-model";
import { useFirstDeploy } from "../deploy/use-first-deploy";
import { type InstanceSummary, isDeploymentSettled, summarizeInstances } from "./instance-state";
import {
  type LogGroup,
  type RunView,
  type StepRecords,
  logGroups,
  nextRevealedSteps,
  pollIntervals,
  runView,
} from "./run-model";

export const BUILD_LOG_LIMIT = 20;
const STEP_REVEAL_MS = 120;

function useRevealedSteps(target: StepRecords, instant: boolean): StepRecords {
  const [revealed, setRevealed] = useState<StepRecords>(target);
  if (instant && nextRevealedSteps(revealed, target) !== null) {
    setRevealed(target);
  }
  useEffect(() => {
    const next = instant ? null : nextRevealedSteps(revealed, target);
    if (!next) {
      return;
    }
    const timer = setTimeout(() => setRevealed(next), STEP_REVEAL_MS);
    return () => clearTimeout(timer);
  }, [revealed, target, instant]);
  return revealed;
}
const RUNTIME_LOG_LIMIT = 100;

const emptySteps: StepRecords = {};

export type DeployRun = {
  deployment: Deployment | undefined;
  instances: InstanceSummary | null;
  view: RunView;
  groups: LogGroup[];
  logsLoading: boolean;
  buildLogsCapped: boolean;
  source: SourceKind;
  primaryDomain: DisplayDomain | null;
  appHref: Route;
  deploymentHref: Route;
  retry: ReturnType<typeof useFirstDeploy>;
};

type DeployRunProps = {
  projectId: string;
  appId: string;
  source: SourceKind;
  deploymentId: string;
  onDeploymentCreated: (deploymentId: string) => void;
  children: ReactNode;
};

const DeployRunContext = createContext<DeployRun | null>(null);

export function useDeployRun(): DeployRun {
  const run = useContext(DeployRunContext);
  if (!run) {
    throw new Error("useDeployRun must be used inside DeployRunProvider");
  }
  return run;
}

export function DeployRunProvider(props: DeployRunProps) {
  return (
    <ProjectDataProvider projectId={props.projectId} appId={props.appId}>
      <DeployRunReader {...props} />
    </ProjectDataProvider>
  );
}

function isSettledRow(row: Deployment): boolean {
  return isDeploymentSettled(row.status, summarizeInstances(row));
}

function useNow(ticking: boolean) {
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    if (!ticking) {
      return;
    }
    const timer = setInterval(() => setNow(Date.now()), 500);
    return () => clearInterval(timer);
  }, [ticking]);
  return now;
}

function DeployRunReader({
  projectId,
  appId,
  source,
  deploymentId,
  onDeploymentCreated,
  children,
}: DeployRunProps) {
  const workspace = useWorkspaceNavigation();
  const { getDeploymentById, domains, customDomains } = useProjectData();
  const freshRow = trpc.deploy.deployment.list.useQuery(
    { projectId, deploymentIds: [deploymentId] },
    {
      refetchInterval: (data) => {
        const row = data?.deployments.at(0);
        return pollIntervals({
          settled: row ? isSettledRow(row) : false,
          createdAt: row?.createdAt ?? null,
          now: Date.now(),
        }).steps;
      },
      refetchIntervalInBackground: true,
      refetchOnWindowFocus: false,
    },
  );
  const deployment = freshRow.data?.deployments.at(0) ?? getDeploymentById(deploymentId);
  const status = deployment?.status ?? "pending";
  const inFlight = deployment ? !isSettledRow(deployment) : true;
  const now = useNow(inFlight);
  const poll = pollIntervals({
    settled: !inFlight,
    createdAt: deployment?.createdAt ?? null,
    now,
  });

  const steps = trpc.deploy.deployment.steps.useQuery(
    { deploymentId },
    { refetchInterval: poll.steps, refetchIntervalInBackground: true, refetchOnWindowFocus: false },
  );
  const buildSteps = trpc.deploy.deployment.buildSteps.useQuery(
    { deploymentId, includeStepLogs: true },
    { refetchInterval: poll.logs },
  );
  const runtimeLogs = trpc.deploy.deployment.runtimeLogs.useQuery(
    { deploymentId, limit: RUNTIME_LOG_LIMIT },
    { refetchInterval: poll.logs },
  );
  const retry = useFirstDeploy(projectId, appId, source, onDeploymentCreated);

  const revealedSteps = useRevealedSteps(steps.data ?? emptySteps, !inFlight);
  const builds = buildSteps.data?.steps ?? [];
  const instances = deployment ? summarizeInstances(deployment) : null;
  const view = runView({
    status,
    instances,
    source,
    steps: revealedSteps,
    buildError: builds.findLast((step) => Boolean(step.error))?.error ?? null,
    now,
  });
  const scope = { workspaceSlug: workspace.slug, projectId, appId };

  const run: DeployRun = {
    deployment,
    instances,
    view,
    groups: logGroups(builds, runtimeLogs.data?.logs ?? []),
    logsLoading: buildSteps.isLoading || runtimeLogs.isLoading,
    buildLogsCapped:
      builds.reduce((count, step) => count + (step.logs?.length ?? 0), 0) >= BUILD_LOG_LIMIT,
    source,
    primaryDomain: deployment
      ? getDomainPriority({
          domains: domains.filter((d) => d.deploymentId === deploymentId),
          customDomains,
          environmentId: deployment.environmentId,
          deploymentId,
          currentDeploymentId: deploymentId,
        }).primary
      : null,
    appHref: routes.projects.apps.overview(scope),
    deploymentHref: routes.projects.apps.deployment({ ...scope, deploymentId }),
    retry,
  };
  return <DeployRunContext.Provider value={run}>{children}</DeployRunContext.Provider>;
}
