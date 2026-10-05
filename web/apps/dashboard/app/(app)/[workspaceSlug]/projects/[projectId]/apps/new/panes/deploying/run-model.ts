import type { DeploymentStatus } from "@/lib/collections/deploy/deployment-status";
import { formatCompoundDuration } from "@/lib/utils/metric-formatters";
import type { SourceKind } from "../../wizard-model";

const STAGES = [
  { key: "queued", label: "Queued" },
  { key: "building", label: "Building" },
  { key: "deploying", label: "Starting instances" },
  { key: "network", label: "Assigning domains" },
  { key: "finalizing", label: "Finalizing" },
] as const;

export type StageKey = (typeof STAGES)[number]["key"];
export type StageState = "done" | "active" | "failed" | "waiting" | "skipped";
export type RunOutcome = "running" | "live" | "failed" | "blocked";

type InstanceHealth = { running: number; unhealthy: boolean; error: string };

type StepRecord = {
  startedAt: number;
  endedAt: number | null;
  error: string | null;
};

export type StepRecords = Partial<Record<StageKey, StepRecord | null>>;

export type Stage = {
  key: StageKey;
  label: string;
  state: StageState;
  durationMs: number | null;
  note: string | null;
  error: string | null;
};

export type RunView = {
  outcome: RunOutcome;
  stages: Stage[];
  failedStage: Stage | null;
  firstError: string | null;
  elapsedMs: number | null;
};

const outcomeByStatus: Record<DeploymentStatus, RunOutcome> = {
  pending: "running",
  starting: "running",
  building: "running",
  deploying: "running",
  network: "running",
  finalizing: "running",
  ready: "live",
  failed: "failed",
  cancelled: "failed",
  stopped: "failed",
  superseded: "failed",
  skipped: "failed",
  awaiting_approval: "blocked",
};

const earlyFailure: Partial<Record<DeploymentStatus, string>> = {
  failed: "The deployment failed before it started building.",
  cancelled: "The deployment was cancelled before it started building.",
  stopped: "The deployment was stopped before it started building.",
  superseded: "A newer deployment replaced this one before it started building.",
  skipped: "This deployment was skipped before it started building.",
};

function failEarly(stages: Stage[], status: DeploymentStatus, steps: StepRecords): Stage[] {
  const reason = earlyFailure[status];
  const noSteps = STAGES.every(({ key }) => !steps[key]);
  if (!reason || !noSteps || stages.some((stage) => stage.state === "failed")) {
    return stages;
  }
  return stages.map((stage, index) =>
    index === 0 ? { ...stage, state: "failed", note: null, error: reason } : stage,
  );
}

type RunInput = {
  status: DeploymentStatus;
  health: InstanceHealth | null;
  source: SourceKind;
  steps: StepRecords;
  buildError: string | null;
  now: number;
};

const DEPLOY_STAGE = STAGES.findIndex((stage) => stage.key === "deploying");

function resolveOutcome(status: DeploymentStatus, health: InstanceHealth | null): RunOutcome {
  const outcome = outcomeByStatus[status];
  if (outcome !== "live" || health === null) {
    return outcome;
  }
  if (health.unhealthy) {
    return "failed";
  }
  return health.running > 0 ? "live" : "running";
}

type SettleInput = {
  status: DeploymentStatus;
  health: InstanceHealth | null;
  outcome: RunOutcome;
  steps: StepRecords;
  now: number;
};

function settleInstances(stages: Stage[], input: SettleInput): Stage[] {
  const { status, health, outcome, steps, now } = input;
  if (health === null) {
    return stages;
  }
  if (outcomeByStatus[status] === "live" && health.unhealthy) {
    return stages.map((stage, index) =>
      index === DEPLOY_STAGE
        ? { ...stage, state: "failed", note: null, error: health.error }
        : stage,
    );
  }
  const waitingForInstance =
    outcome === "running" && health.running === 0 && stages[DEPLOY_STAGE].state === "done";
  if (!waitingForInstance) {
    return stages;
  }
  const deployStep = steps.deploying;
  return stages.map((stage, index) => {
    if (index === DEPLOY_STAGE) {
      return {
        ...stage,
        state: "active",
        note: null,
        durationMs: deployStep ? now - deployStep.startedAt : null,
      };
    }
    if (index > DEPLOY_STAGE && !steps[stage.key]?.endedAt) {
      return { ...stage, state: "waiting", note: null, durationMs: null };
    }
    return stage;
  });
}

function recordedState(step: StepRecord, outcome: RunOutcome, finishedLater: boolean): StageState {
  if (step.error) {
    return "failed";
  }
  if (step.endedAt !== null || finishedLater || outcome === "live") {
    return "done";
  }
  return outcome === "failed" ? "failed" : "active";
}

function missingStage(
  key: StageKey,
  outcome: RunOutcome,
  reachedLater: boolean,
  source: SourceKind,
): Pick<Stage, "state" | "note"> {
  if (key === "building" && reachedLater && source === "oci") {
    return { state: "skipped", note: "Prebuilt image" };
  }
  if (reachedLater || outcome === "live") {
    return { state: "done", note: null };
  }
  if (outcome === "failed") {
    return { state: "skipped", note: "Skipped" };
  }
  return { state: "waiting", note: null };
}

export function runView({ status, health, source, steps, buildError, now }: RunInput): RunView {
  const outcome = resolveOutcome(status, health);
  const lastFinished = STAGES.findLastIndex(({ key }) => {
    const step = steps[key];
    return Boolean(step && step.endedAt !== null && !step.error);
  });
  const derived = STAGES.map(({ key, label }, index): Stage => {
    const step = steps[key];
    if (step) {
      const state = recordedState(step, outcome, index < lastFinished);
      const settled = state !== "active";
      return {
        key,
        label,
        state,
        durationMs:
          step.endedAt !== null
            ? step.endedAt - step.startedAt
            : settled
              ? null
              : now - step.startedAt,
        note: null,
        error: step.error ?? (state === "failed" && key === "building" ? buildError : null),
      };
    }
    const reachedLater = STAGES.slice(index + 1).some((later) => Boolean(steps[later.key]));
    return {
      key,
      label,
      durationMs: null,
      error: null,
      ...missingStage(key, outcome, reachedLater, source),
    };
  });

  const settled = settleInstances(derived, { status, health, outcome, steps, now });
  const started = failEarly(settled, status, steps);
  const failedAt = started.findIndex((stage) => stage.state === "failed");
  const stages =
    failedAt === -1
      ? started
      : started.map((stage, index) =>
          index > failedAt && stage.state !== "failed"
            ? { ...stage, state: "skipped" as const, note: "Skipped", durationMs: null }
            : stage,
        );

  const recorded = STAGES.flatMap(({ key }) => {
    const step = steps[key];
    return step ? [step] : [];
  });
  const firstStart = recorded.length > 0 ? Math.min(...recorded.map((s) => s.startedAt)) : null;
  const ends = recorded.flatMap((s) => (s.endedAt === null ? [] : [s.endedAt]));
  const lastEnd = ends.length > 0 ? Math.max(...ends) : now;
  const failedStage = stages.find((stage) => stage.state === "failed") ?? null;

  return {
    outcome,
    stages,
    failedStage,
    firstError: buildError ?? failedStage?.error ?? null,
    elapsedMs: firstStart === null ? null : (outcome === "running" ? now : lastEnd) - firstStart,
  };
}

export type LogTone = "plain" | "error" | "warn";

export type LogGroup = {
  id: string;
  title: string;
  failed: boolean;
  lines: { id: string; offsetMs: number; text: string; tone: LogTone }[];
};

type BuildStepInput = {
  step_id: string;
  name: string;
  started_at: number;
  cached: boolean;
  error: string | null;
  logs?: { time: number; message: string }[];
};

type RuntimeLogInput = { time: number; severity: string; message: string };

const severityTone: Record<string, LogTone> = {
  ERROR: "error",
  FATAL: "error",
  WARN: "warn",
  WARNING: "warn",
};

export function logGroups(
  buildSteps: readonly BuildStepInput[],
  runtimeLogs: readonly RuntimeLogInput[],
): LogGroup[] {
  const times = [...buildSteps.map((s) => s.started_at), ...runtimeLogs.map((log) => log.time)];
  const origin = times.length > 0 ? Math.min(...times) : 0;

  const build = [...buildSteps]
    .sort((a, b) => a.started_at - b.started_at)
    .map((step, position): LogGroup => {
      const groupId = `${position}-${step.step_id}`;
      return {
        id: groupId,
        title: step.cached ? `${step.name} (cached)` : step.name,
        failed: Boolean(step.error),
        lines: [
          ...(step.logs ?? []).map((log, index) => ({
            id: `${groupId}-${index}`,
            offsetMs: log.time - origin,
            text: log.message,
            tone: "plain" as const,
          })),
          ...(step.error
            ? [
                {
                  id: `${groupId}-error`,
                  offsetMs: (step.logs?.at(-1)?.time ?? step.started_at) - origin,
                  text: step.error,
                  tone: "error" as const,
                },
              ]
            : []),
        ],
      };
    });

  if (runtimeLogs.length === 0) {
    return build;
  }
  const runtime: LogGroup = {
    id: "runtime",
    title: "Runtime logs",
    failed: false,
    lines: [...runtimeLogs]
      .sort((a, b) => a.time - b.time)
      .map((log, index) => ({
        id: `runtime-${index}`,
        offsetMs: log.time - origin,
        text: log.message,
        tone: severityTone[log.severity.toUpperCase()] ?? "plain",
      })),
  };
  return [...build, runtime];
}

export function formatOffset(ms: number): string {
  const total = Math.max(0, Math.floor(ms / 1000));
  const minutes = Math.floor(total / 60);
  const seconds = total % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

export function formatStageDuration(ms: number): string {
  return ms < 1000 ? "0s" : formatCompoundDuration(ms);
}

function sameStep(a: StepRecord | null | undefined, b: StepRecord | null | undefined): boolean {
  if (!a || !b) {
    return !a && !b;
  }
  return a.startedAt === b.startedAt && a.endedAt === b.endedAt && a.error === b.error;
}

export function nextRevealedSteps(revealed: StepRecords, target: StepRecords): StepRecords | null {
  const pending = STAGES.find(({ key }) => !sameStep(revealed[key], target[key]));
  return pending ? { ...revealed, [pending.key]: target[pending.key] } : null;
}

type EmptyLogCopy = { title: string; reason: string };

const emptyBuildLog: Record<RunOutcome, EmptyLogCopy> = {
  running: { title: "Waiting for logs…", reason: "Logs stream in as soon as the build starts." },
  live: {
    title: "No logs",
    reason: "The build finished without writing any logs.",
  },
  blocked: {
    title: "No logs yet",
    reason: "Logs appear after someone approves this deployment.",
  },
  failed: {
    title: "No logs",
    reason: "The build stopped before it wrote any logs.",
  },
};

export function emptyLogCopy(source: SourceKind, outcome: RunOutcome): EmptyLogCopy {
  if (source === "oci") {
    return {
      title: outcome === "running" ? "Waiting for logs…" : "No logs",
      reason: "Images skip the build step. Runtime logs show here once an instance starts.",
    };
  }
  return emptyBuildLog[outcome];
}
