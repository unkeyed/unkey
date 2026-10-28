import type { Deployment } from "./deployments";

export const INSTANCE_STATUSES = ["inactive", "pending", "running", "failed"] as const;

export type InstanceStatus = (typeof INSTANCE_STATUSES)[number];

export function toInstanceKey(k8sPodName: string, region: string): string {
  return `${k8sPodName}::${region}`;
}

export type InstanceState = "starting" | "running" | "unhealthy" | "inactive";

export const instanceLabel: Record<InstanceState, string> = {
  starting: "Starting",
  running: "Running",
  unhealthy: "Unhealthy",
  inactive: "Disabled",
};

const stateByStatus: Record<InstanceStatus, InstanceState> = {
  pending: "starting",
  running: "running",
  failed: "unhealthy",
  inactive: "inactive",
};

type DeploymentInstance = Deployment["instances"][number];

export type InstanceHealth = Pick<DeploymentInstance, "status" | "lastExit">;

export type InstancesSource = Pick<Deployment, "instances" | "targetRegions" | "desiredState">;

export type InstanceCount = { running: number; total: number };

export type InstanceSummary = InstanceCount & { state: InstanceState };

export type RegionInstances = InstanceSummary & {
  name: string;
  instances: DeploymentInstance[];
  waiting: number;
};

const FATAL_WAITING_REASONS = [
  "CrashLoopBackOff",
  "ImagePullBackOff",
  "ErrImagePull",
  "CreateContainerConfigError",
  "InvalidImageName",
] as const;

export type FatalWaitingReason = (typeof FATAL_WAITING_REASONS)[number];

const fatalWaitingReasons: ReadonlySet<string> = new Set(FATAL_WAITING_REASONS);

export function isFatalWaitingReason(
  statusReason: string | null,
): statusReason is FatalWaitingReason {
  return statusReason !== null && fatalWaitingReasons.has(statusReason);
}

export function instanceStateOf({ status, lastExit }: InstanceHealth): InstanceState {
  if (isFatalWaitingReason(lastExit?.statusReason ?? null) && status !== "inactive") {
    return "unhealthy";
  }
  return stateByStatus[status];
}

export function groupInstancesByRegion({
  instances,
  targetRegions,
  desiredState,
}: InstancesSource): RegionInstances[] {
  const byRegion = Map.groupBy(instances, (i) => i.region.name);
  const wantedByRegion = new Map(
    desiredState === "running" ? targetRegions.map((t) => [t.region.name, t.replicas]) : [],
  );
  const names = new Set([...wantedByRegion.keys(), ...byRegion.keys()]);
  return [...names].map((name) => {
    const rows = byRegion.get(name) ?? [];
    const states = rows.map(instanceStateOf);
    const total = Math.max(wantedByRegion.get(name) ?? 0, rows.length);
    const waiting = total - rows.length;
    const state = aggregateInstanceStates(states);
    return {
      name,
      instances: rows,
      state: waiting > 0 && state === "running" ? "starting" : state,
      running: states.filter((s) => s === "running").length,
      total,
      waiting,
    };
  });
}

export function summarizeInstances(source: InstancesSource): InstanceSummary | null {
  const regions = groupInstancesByRegion(source);
  const total = regions.reduce((sum, r) => sum + r.total, 0);
  if (total === 0) {
    return null;
  }
  return {
    state: aggregateInstanceStates(regions.map((r) => r.state)),
    running: regions.reduce((sum, r) => sum + r.running, 0),
    total,
  };
}

export function runningCountText({ running, total }: InstanceCount): string {
  return `${running} of ${total} running`;
}

export function instancesText(
  summary: InstanceSummary | null,
  desiredState: Deployment["desiredState"],
): string {
  if (summary) {
    return runningCountText(summary);
  }
  return desiredState === "stopped" ? "No instances running" : "No instances";
}

export function aggregateInstanceStates(states: readonly InstanceState[]): InstanceState {
  const allInactive = states.length > 0 && states.every((s) => s === "inactive");
  if (states.includes("unhealthy")) {
    return "unhealthy";
  }
  if (states.includes("running")) {
    return "running";
  }
  if (allInactive) {
    return "inactive";
  }
  return "starting";
}
