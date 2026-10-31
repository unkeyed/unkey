import type { LastExit } from "@/lib/types/deploy";
import type { DeploymentExitReason } from "@unkey/api/models/components";
import type { Deployment } from "./deployments";
import { type FatalWaitingReason, isFatalWaitingReason } from "./instance-status";

type ExitFacts = Pick<LastExit, "exitCode" | "signal" | "reason" | "statusReason">;

type ReasonCopy = { label: string; body: string; explainExitCode?: true };

const CANT_PULL: ReasonCopy = {
  label: "Can't pull image",
  body: "Check the image name, tag, and registry access.",
};

const CANT_START: ReasonCopy = {
  label: "Couldn't start",
  body: "Check your start command and entrypoint.",
};

const WAITING_REASONS = {
  CrashLoopBackOff: {
    label: "Crashing",
    body: "Your app exits right after it starts. Check your logs.",
    explainExitCode: true,
  },
  ImagePullBackOff: CANT_PULL,
  ErrImagePull: CANT_PULL,
  InvalidImageName: {
    label: "Invalid image name",
    body: "Check the image name in your build settings.",
  },
  CreateContainerConfigError: {
    label: "Config error",
    body: "An environment variable or secret is missing.",
  },
} satisfies Record<FatalWaitingReason, ReasonCopy>;

const CRASH_CAUSES = {
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
} satisfies Record<string, ReasonCopy>;

const TERMINATION_REASONS = {
  ...CRASH_CAUSES,
  ContainerStatusUnknown: {
    label: "Lost contact",
    body: "The server it ran on went away.",
  },
} satisfies Record<string, ReasonCopy>;

function hasKey<T extends object>(table: T, key: string | null): key is Extract<keyof T, string> {
  return key !== null && Object.hasOwn(table, key);
}

const EXIT_CODE_HINTS = new Map<number, string>([
  [1, "Unhandled error or failed startup check."],
  [2, "Crashed, usually a panic or uncaught exception."],
  [126, "Start command found but not executable."],
  [127, "Start command not found."],
  [137, "Killed with SIGKILL, often because it did not stop in time."],
  [139, "Invalid memory access in your code."],
  [143, "Shut down on request. Normal during a redeploy."],
]);

const SIGNAL_EXIT_BASE = 128;

function signalOf(exitCode: number, signal: number | null): number | null {
  if (signal !== null && signal > 0) {
    return signal;
  }
  return exitCode > SIGNAL_EXIT_BASE ? exitCode - SIGNAL_EXIT_BASE : null;
}

function exitCodeHint(exitCode: number, signal: number | null): string {
  const killedBy = signalOf(exitCode, signal);
  return (
    EXIT_CODE_HINTS.get(exitCode) ??
    (killedBy !== null
      ? `Killed by signal ${killedBy}.`
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

function rawOf(lastExit: ExitFacts): string | null {
  const reasons = [...new Set([lastExit.statusReason, lastExit.reason])];
  const code = lastExit.exitCode !== null ? `exit code ${lastExit.exitCode}` : null;
  return [...reasons, code].filter(Boolean).join(" · ") || null;
}

type Resolution = {
  copy: ReasonCopy | null;
  severity: ExitSummary["severity"];
  showsExitCode: boolean;
};

function resolve({ statusReason, reason, exitCode }: ExitFacts): Resolution {
  if (statusReason === "CrashLoopBackOff") {
    return {
      copy: hasKey(CRASH_CAUSES, reason) ? CRASH_CAUSES[reason] : WAITING_REASONS.CrashLoopBackOff,
      severity: "error",
      showsExitCode: true,
    };
  }
  if (isFatalWaitingReason(statusReason)) {
    return { copy: WAITING_REASONS[statusReason], severity: "error", showsExitCode: false };
  }
  if (hasKey(TERMINATION_REASONS, reason)) {
    return { copy: TERMINATION_REASONS[reason], severity: "warning", showsExitCode: true };
  }
  const failed = !reason && exitCode !== null && exitCode !== 0;
  return { copy: failed ? CRASH_CAUSES.Error : null, severity: "warning", showsExitCode: true };
}

export function summarizeExit(lastExit: ExitFacts): ExitSummary {
  const { copy, severity, showsExitCode } = resolve(lastExit);
  const failedCode =
    showsExitCode && lastExit.exitCode !== null && lastExit.exitCode !== 0
      ? lastExit.exitCode
      : null;
  const hint =
    failedCode !== null && (copy === null || copy.explainExitCode === true)
      ? exitCodeHint(failedCode, lastExit.signal)
      : null;
  return {
    severity,
    label: copy?.label ?? (lastExit.reason || "Exited"),
    exit: failedCode !== null ? `exit=${failedCode}` : null,
    sentences: [hint, copy?.body ?? null].filter((sentence) => sentence !== null),
    raw: rawOf(lastExit),
  };
}

type KnownExitReason = (typeof DeploymentExitReason)[keyof typeof DeploymentExitReason];

const EXIT_LABELS = {
  out_of_memory: CRASH_CAUSES.OOMKilled.label,
  exited_with_error: CRASH_CAUSES.Error.label,
  start_failed: CANT_START.label,
  exited_cleanly: CRASH_CAUSES.Completed.label,
  unknown: WAITING_REASONS.CrashLoopBackOff.label,
} satisfies Record<KnownExitReason, string>;

export function summarizeCrashLoop(crashLoop: NonNullable<Deployment["crashLoop"]>): ExitSummary {
  const failedCode =
    crashLoop.exitCode !== null && crashLoop.exitCode !== 0 ? crashLoop.exitCode : null;
  return {
    severity: "error",
    label: hasKey(EXIT_LABELS, crashLoop.reason)
      ? EXIT_LABELS[crashLoop.reason]
      : EXIT_LABELS.unknown,
    exit: failedCode !== null ? `exit=${failedCode}` : null,
    sentences: [crashLoop.message],
    raw: null,
  };
}
