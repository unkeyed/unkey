import {
  type DeploymentStatus,
  isDeploymentInFlight,
} from "@/lib/collections/deploy/deployment-status";
import type { LogTone } from "./run-model";

type InstanceState = "starting" | "running" | "unhealthy" | "inactive";

type InstanceCopy = { label: string; description: string; tone: LogTone };

const instanceCopy: Record<InstanceState, InstanceCopy> = {
  starting: { label: "Starting", description: "Instance is starting up.", tone: "plain" },
  running: { label: "Running", description: "Instance is running.", tone: "plain" },
  unhealthy: { label: "Unhealthy", description: "Instance stopped unexpectedly.", tone: "error" },
  inactive: {
    label: "Inactive",
    description: "Instance is inactive or unavailable.",
    tone: "plain",
  },
};

export type LastExit = {
  restartCount: number;
  exitCode: number | null;
  reason: string | null;
  statusReason: string | null;
};

export type InstanceSnapshot = {
  instances: readonly { status: "inactive" | "pending" | "running" | "failed" }[];
  desiredInstanceCount: number;
  lastExit: LastExit | null;
};

export type InstanceSummary = InstanceCopy & {
  state: InstanceState;
  running: number;
  total: number;
  text: string;
  lastExit: string | null;
};

const stateByStatus: Record<InstanceSnapshot["instances"][number]["status"], InstanceState> = {
  pending: "starting",
  running: "running",
  failed: "unhealthy",
  inactive: "inactive",
};

function aggregateState(snapshot: InstanceSnapshot): InstanceState {
  if (snapshot.lastExit?.statusReason === "CrashLoopBackOff") {
    return "unhealthy";
  }
  const states = snapshot.instances.map((instance) => stateByStatus[instance.status]);
  if (states.includes("unhealthy")) {
    return "unhealthy";
  }
  if (states.includes("running")) {
    return "running";
  }
  if (states.length > 0 && states.every((state) => state === "inactive")) {
    return "inactive";
  }
  return "starting";
}

const exitReasonLabels: Record<string, string> = {
  CrashLoopBackOff: "Keeps crashing",
  OOMKilled: "Out of memory",
  Completed: "Exited",
};

function exitReason(reason: string | null): string {
  return reason === null ? "Unknown" : (exitReasonLabels[reason] ?? reason);
}

export function lastExitDetail(exit: LastExit | null): string | null {
  if (!exit || (exit.statusReason === null && exit.reason === null && exit.exitCode === null)) {
    return null;
  }
  const parts = [
    `Last exit: ${exitReason(exit.statusReason ?? exit.reason)}`,
    exit.exitCode === null ? null : `Exit code ${exit.exitCode}`,
    `${exit.restartCount} ${exit.restartCount === 1 ? "restart" : "restarts"}`,
  ];
  return parts.filter((part) => part !== null).join(" · ");
}

export function summarizeInstances(snapshot: InstanceSnapshot): InstanceSummary | null {
  const total = Math.max(snapshot.desiredInstanceCount, snapshot.instances.length);
  if (total === 0) {
    return null;
  }
  const state = aggregateState(snapshot);
  const copy = instanceCopy[state];
  const running = snapshot.instances.filter((i) => i.status === "running").length;
  const text =
    state === "unhealthy"
      ? `${copy.label} · ${copy.description}`
      : `${running} of ${total} ${copy.label.toLowerCase()}`;
  return { ...copy, state, running, total, text, lastExit: lastExitDetail(snapshot.lastExit) };
}

export function crashExplanation(exit: LastExit | null, printedOutput: boolean): string {
  const exited = exit?.exitCode !== null && exit?.exitCode !== undefined;
  const started = exited
    ? `Your app exited right after it started (exit code ${exit?.exitCode})`
    : "Your app stopped right after it started";
  const output = printedOutput ? "." : " and printed nothing.";
  return `${started}${output}`;
}

const settledStates: Record<InstanceState, boolean> = {
  starting: false,
  running: true,
  unhealthy: true,
  inactive: false,
};

export function isDeploymentSettled(
  status: DeploymentStatus,
  instances: InstanceSummary | null,
): boolean {
  if (isDeploymentInFlight(status)) {
    return false;
  }
  if (status !== "ready") {
    return true;
  }
  return instances === null || settledStates[instances.state];
}
