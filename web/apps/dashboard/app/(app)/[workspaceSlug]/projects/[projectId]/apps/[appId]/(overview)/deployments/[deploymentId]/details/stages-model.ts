import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import {
  type InstanceSummary,
  instanceLabel,
  instancesText,
} from "@/lib/collections/deploy/instance-status";
import { plural } from "@/lib/fmt";
import type { StepsData } from "../use-deployment-status";

export type StageKey = "build" | "instances" | "domains";

export type StepResolution = {
  status: "pending" | "started" | "completed" | "error" | "skipped";
  description: string;
  duration: number | undefined;
};

type StepData = NonNullable<NonNullable<StepsData>["building"]>;

const stageLabels: Record<"instances" | "domains", Record<StepResolution["status"], string>> = {
  instances: {
    pending: "Starting Instances",
    started: "Starting Instances",
    error: "Instances",
    skipped: "Instances",
    completed: "Instances",
  },
  domains: {
    pending: "Assigning Domains",
    started: "Assigning Domains",
    error: "Domains",
    skipped: "Domains",
    completed: "Domains",
  },
};

type StepMessages = { completed: string; inProgress: string; waiting: string };

type Phase =
  | { kind: "live" }
  | { kind: "failed" }
  | { kind: "settled" }
  | { kind: "stopped"; label: string };

const LIVE: Phase = { kind: "live" };
const SETTLED: Phase = { kind: "settled" };

const PHASE_BY_STATUS: Record<DeploymentStatus, Phase> = {
  pending: LIVE,
  starting: LIVE,
  building: LIVE,
  deploying: LIVE,
  network: LIVE,
  finalizing: LIVE,
  awaiting_approval: LIVE,
  skipped: LIVE,
  ready: SETTLED,
  stopped: SETTLED,
  failed: { kind: "failed" },
  superseded: { kind: "stopped", label: "Superseded" },
  cancelled: { kind: "stopped", label: "Cancelled" },
};

type StageBase = { label: string; resolution: StepResolution };

export type Stage = StageBase & { key: StageKey; expandable: boolean };

export type StagesInput = {
  steps: StepsData | undefined;
  now: number;
  status: DeploymentStatus;
  isPrebuilt: boolean;
  instances: InstanceSummary | null;
  desiredState: Deployment["desiredState"];
  domainCount: number | null;
};

export function resolveStages({
  steps,
  now,
  status,
  isPrebuilt,
  instances,
  desiredState,
  domainCount,
}: StagesInput): Stage[] {
  const { queued, starting, building, deploying, network, finalizing } = steps ?? {};
  const phase = PHASE_BY_STATUS[status];
  const failed = phase.kind === "failed";
  // A prebuilt image never builds, so its start-up failure belongs to instances
  const startFailure = failed ? firstFailed(queued, starting) : undefined;

  const build = resolveStep(
    (isPrebuilt ? undefined : startFailure) ?? building,
    now,
    isPrebuilt ? SETTLED : phase,
    {
      completed: isPrebuilt ? "Image was prebuilt" : "Build complete",
      inProgress: "Building",
      waiting: "Waiting to start",
    },
  );
  const instancesStep = resolveStep(
    (isPrebuilt ? startFailure : undefined) ?? deploying,
    now,
    phase,
    {
      completed: instancesText(instances, desiredState),
      inProgress: "Starting instances",
      waiting: "Waiting for build",
    },
  );
  const domains = resolveStep(
    (failed ? firstFailed(finalizing) : undefined) ?? network,
    now,
    phase,
    {
      completed:
        domainCount === null ? "Assigning domains" : `${plural(domainCount, "domain")} assigned`,
      inProgress: "Assigning domains",
      waiting: "Waiting for instances",
    },
  );

  const stages: Stage[] = [
    {
      key: "build",
      label: "Build Logs",
      resolution: build,
      expandable: !isPrebuilt,
    },
    {
      key: "instances",
      label: stageLabels.instances[instancesStep.status],
      resolution: withInstanceHealth(instancesStep, instances),
      expandable: instancesStep.status === "error" || Boolean(deploying) || status === "ready",
    },
    {
      key: "domains",
      label: stageLabels.domains[domains.status],
      resolution: domains,
      expandable:
        domains.status === "error" ||
        (domainCount !== 0 && (Boolean(network) || status === "ready")),
    },
  ];
  return failed && steps !== undefined ? withFailure(stages) : stages;
}

function firstFailed(...candidates: Array<StepData | null | undefined>): StepData | undefined {
  return candidates.find((step) => step?.error && !step.completed) ?? undefined;
}

function withFailure(stages: Stage[]): Stage[] {
  if (stages.some((stage) => stage.resolution.status === "error")) {
    return stages;
  }
  const unfinished = stages.findIndex((stage) => stage.resolution.status !== "completed");
  const failedAt = unfinished === -1 ? stages.length - 1 : unfinished;
  return stages.map((stage, i) =>
    i === failedAt
      ? {
          ...stage,
          expandable: true,
          resolution: { ...stage.resolution, status: "error", description: "Deployment failed" },
        }
      : stage,
  );
}

function resolveStep(
  step: StepData | null | undefined,
  now: number,
  phase: Phase,
  messages: StepMessages,
): StepResolution {
  if (!step) {
    return missingStep(phase, messages);
  }
  const duration = (step.endedAt ?? now) - step.startedAt;
  if (step.error && phase.kind !== "stopped") {
    return { status: "error", description: step.error, duration };
  }
  if (step.completed) {
    return { status: "completed", description: messages.completed, duration };
  }
  return phase.kind === "stopped"
    ? { status: "skipped", description: step.error ?? phase.label, duration }
    : { status: "started", description: messages.inProgress, duration };
}

function missingStep(phase: Phase, messages: StepMessages): StepResolution {
  switch (phase.kind) {
    case "settled":
      return { status: "completed", description: messages.completed, duration: undefined };
    case "failed":
      return { status: "skipped", description: "Skipped", duration: undefined };
    case "stopped":
      return { status: "skipped", description: phase.label, duration: undefined };
    case "live":
      return { status: "pending", description: messages.waiting, duration: undefined };
  }
}

function withInstanceHealth(
  step: StepResolution,
  instances: InstanceSummary | null,
): StepResolution {
  if (step.status !== "completed" || instances?.state !== "unhealthy") {
    return step;
  }
  return {
    ...step,
    status: "error",
    description: `${step.description} · ${instanceLabel.unhealthy}`,
  };
}

export type OpenStage = {
  open: StageKey | null;
  autoOpened: readonly StageKey[];
  requested: StageKey | null;
  pinned: boolean;
};

export function initialOpenStage({
  failed,
  requested,
}: {
  failed: StageKey | null;
  requested: StageKey | null;
}): OpenStage {
  return {
    open: requested ?? failed,
    autoOpened: failed ? [failed] : [],
    requested,
    pinned: requested !== null,
  };
}

// The request leaves the URL before steps load, so its pin holds until they do and
// absorbs the failure they bring; a later failure opens its own stage
export function followRequestedStage(
  state: OpenStage,
  requested: StageKey | null,
  stepsLoaded: boolean,
): OpenStage {
  if (requested) {
    return requested === state.requested
      ? state
      : { ...state, requested, open: requested, pinned: true };
  }
  const pinned = state.pinned && !stepsLoaded;
  return state.requested === null && pinned === state.pinned
    ? state
    : { ...state, requested: null, pinned };
}

export function followFailedStage(state: OpenStage, failed: StageKey | null): OpenStage {
  if (!failed || state.autoOpened.includes(failed)) {
    return state;
  }
  return {
    ...state,
    open: state.pinned ? state.open : failed,
    autoOpened: [...state.autoOpened, failed],
  };
}

export function toggleStage(state: OpenStage, key: StageKey): OpenStage {
  return { ...state, open: state.open === key ? null : key, pinned: false };
}
