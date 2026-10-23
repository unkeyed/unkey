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
  const { building, deploying, network } = steps ?? {};
  const isFailed = status === "failed";
  const isSettled = status === "ready" || status === "stopped";

  const build = resolveStep(
    building,
    { now, isFailed, implicitlyComplete: isPrebuilt || isSettled },
    {
      completed: isPrebuilt ? "Image was prebuilt" : "Build complete",
      inProgress: "Building",
      waiting: "Waiting to start",
    },
  );
  const instancesStep = resolveStep(
    deploying,
    { now, isFailed, implicitlyComplete: isSettled },
    {
      completed: instancesText(instances, desiredState),
      inProgress: "Starting instances",
      waiting: "Waiting for build",
    },
  );
  const domains = resolveStep(
    network,
    { now, isFailed, implicitlyComplete: isSettled },
    {
      completed:
        domainCount === null ? "Assigning domains" : `${plural(domainCount, "domain")} assigned`,
      inProgress: "Assigning domains",
      waiting: "Waiting for instances",
    },
  );

  return [
    { key: "build", label: "Build Logs", resolution: build, expandable: !isPrebuilt },
    {
      key: "instances",
      label: stageLabels.instances[instancesStep.status],
      resolution: withInstanceHealth(instancesStep, instances),
      expandable: Boolean(deploying) || status === "ready",
    },
    {
      key: "domains",
      label: stageLabels.domains[domains.status],
      resolution: domains,
      expandable: domainCount !== 0 && (Boolean(network) || status === "ready"),
    },
  ];
}

function resolveStep(
  step: StepData | null | undefined,
  {
    now,
    isFailed,
    implicitlyComplete,
  }: { now: number; isFailed: boolean; implicitlyComplete: boolean },
  messages: StepMessages,
): StepResolution {
  if (!step) {
    if (implicitlyComplete) {
      return { status: "completed", description: messages.completed, duration: undefined };
    }
    return isFailed
      ? { status: "skipped", description: "Skipped", duration: undefined }
      : { status: "pending", description: messages.waiting, duration: undefined };
  }
  const duration = (step.endedAt ?? now) - step.startedAt;
  if (step.error) {
    return { status: "error", description: step.error, duration };
  }
  return step.completed
    ? { status: "completed", description: messages.completed, duration }
    : { status: "started", description: messages.inProgress, duration };
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
};

// On landing a failed stage wins over a requested one
export function initialOpenStage({
  failed,
  requested,
}: {
  failed: StageKey | null;
  requested: StageKey | null;
}): OpenStage {
  return { open: failed ?? requested, autoOpened: failed ? [failed] : [], requested };
}

export function followRequestedStage(state: OpenStage, requested: StageKey | null): OpenStage {
  if (requested === state.requested) {
    return state;
  }
  return { ...state, requested, open: requested ?? state.open };
}

export function followFailedStage(state: OpenStage, failed: StageKey | null): OpenStage {
  if (!failed || state.autoOpened.includes(failed)) {
    return state;
  }
  return { ...state, open: failed, autoOpened: [...state.autoOpened, failed] };
}

export function toggleStage(state: OpenStage, key: StageKey): OpenStage {
  return { ...state, open: state.open === key ? null : key };
}
