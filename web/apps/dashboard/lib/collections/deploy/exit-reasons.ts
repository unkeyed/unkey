import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { LastExit } from "@/lib/types/deploy";
import { DeploymentExitReason } from "@unkey/api/models/components";

type ReasonCopy = { label: string; body: string; fatalWaiting?: true; explainExitCode?: true };

const CANT_PULL: ReasonCopy = {
  label: "Can't pull image",
  body: "Check the image name, tag, and registry access.",
  fatalWaiting: true,
};

const CANT_START: ReasonCopy = {
  label: "Couldn't start",
  body: "Check your start command and entrypoint.",
};

const EXIT_REASONS = {
  CrashLoopBackOff: {
    label: "Crashing",
    body: "Your app exits right after it starts. Check your logs.",
    fatalWaiting: true,
  },
  ImagePullBackOff: CANT_PULL,
  ErrImagePull: CANT_PULL,
  InvalidImageName: {
    label: "Invalid image name",
    body: "Check the image name in your build settings.",
    fatalWaiting: true,
  },
  CreateContainerConfigError: {
    label: "Config error",
    body: "An environment variable or secret is missing.",
    fatalWaiting: true,
  },
  OOMKilled: {
    label: "Out of memory",
    body: "Your app hit its memory limit. Fix a leak or raise the limit.",
  },
  Error: {
    label: "Exited with error",
    body: "Check your logs for the cause.",
    explainExitCode: true,
  },
  ContainerCannotRun: CANT_START,
  StartError: CANT_START,
  Completed: {
    label: "Exited cleanly",
    body: "Your app stopped on its own. A service should keep running.",
  },
  ContainerStatusUnknown: {
    label: "Lost contact",
    body: "The server it ran on went away.",
  },
} satisfies Record<string, ReasonCopy>;

type KnownReason = keyof typeof EXIT_REASONS;

function isKnownReason(reason: string): reason is KnownReason {
  return Object.hasOwn(EXIT_REASONS, reason);
}

function reasonCopy(reason: string | null): ReasonCopy | null {
  return reason !== null && isKnownReason(reason) ? EXIT_REASONS[reason] : null;
}

export function isFatalWaitingReason(reason: string | null): reason is string {
  return reasonCopy(reason)?.fatalWaiting === true;
}

const EXIT_CODE_HINTS = new Map<number, string>([
  [1, "Unhandled error or failed startup check."],
  [2, "Crashed, usually a panic or uncaught exception."],
  [126, "Start command found but not executable."],
  [127, "Start command not found."],
  [128, "Force-killed, usually out of memory."],
  [137, "Force-killed, usually out of memory."],
  [139, "Invalid memory access in your code."],
  [143, "Shut down on request. Normal during a redeploy."],
]);

function exitCodeHint(exitCode: number, signal: number | null): string {
  return (
    EXIT_CODE_HINTS.get(exitCode) ??
    (signal !== null && signal > 0
      ? "Force-killed, usually out of memory."
      : "See your app's docs for this exit code.")
  );
}

export type ExitSummary = {
  severity: "error" | "warning";
  label: string;
  exit: string | null;
  sentences: string[];
  raw: string | null;
};

function rawOf(lastExit: LastExit): string | null {
  const reasons = [...new Set([lastExit.statusReason, lastExit.reason])];
  const code = lastExit.exitCode !== null ? `exit code ${lastExit.exitCode}` : null;
  return [...reasons, code].filter(Boolean).join(" · ") || null;
}

export function summarizeExit(lastExit: LastExit): ExitSummary {
  const stuck = isFatalWaitingReason(lastExit.statusReason);
  const reason = stuck ? lastExit.statusReason : lastExit.reason;
  const copy = reasonCopy(reason);
  const failedCode =
    !stuck && lastExit.exitCode !== null && lastExit.exitCode !== 0 ? lastExit.exitCode : null;
  const hint =
    failedCode !== null && (copy === null || copy.explainExitCode === true)
      ? exitCodeHint(failedCode, lastExit.signal)
      : null;
  return {
    severity: stuck ? "error" : "warning",
    label: copy?.label ?? reason ?? "Exited",
    exit: failedCode !== null ? `exit=${failedCode}` : null,
    sentences: [hint, copy?.body ?? null].filter((sentence) => sentence !== null),
    raw: rawOf(lastExit),
  };
}

type KnownExitReason = (typeof DeploymentExitReason)[keyof typeof DeploymentExitReason];

const EXIT_LABELS = {
  crash_loop: "Crashing",
  out_of_memory: "Out of memory",
  exited_with_error: "Exited with error",
  start_failed: "Couldn't start",
  exited_cleanly: "Exited cleanly",
  unknown: "Exited",
} satisfies Record<KnownExitReason, string>;

function isKnownExitReason(reason: string): reason is KnownExitReason {
  return Object.hasOwn(EXIT_LABELS, reason);
}

export function isCrashLooping(lastExit: Deployment["lastExit"]): boolean {
  return lastExit?.reason === DeploymentExitReason.CrashLoop;
}

export function summarizeDeploymentExit(
  lastExit: NonNullable<Deployment["lastExit"]>,
): ExitSummary {
  const crashing = isCrashLooping(lastExit);
  const failedCode =
    !crashing && lastExit.exitCode !== null && lastExit.exitCode !== 0 ? lastExit.exitCode : null;
  return {
    severity: crashing ? "error" : "warning",
    label: isKnownExitReason(lastExit.reason) ? EXIT_LABELS[lastExit.reason] : EXIT_LABELS.unknown,
    exit: failedCode !== null ? `exit=${failedCode}` : null,
    sentences: [lastExit.message],
    raw: null,
  };
}
