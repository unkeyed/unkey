export const INSTANCE_STATUSES = ["inactive", "pending", "running", "failed"] as const;

export type InstanceStatus = (typeof INSTANCE_STATUSES)[number];

type InstanceState = "starting" | "running" | "unhealthy" | "inactive";

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

export type InstanceHealth = { status: InstanceStatus; statusReason: string | null };

type InstanceSnapshot = {
  instances: readonly InstanceHealth[];
  desiredInstanceCount: number;
};

export type InstanceSummary = {
  state: InstanceState;
  running: number;
  total: number;
};

export function instanceStateOf({ status, statusReason }: InstanceHealth): InstanceState {
  if (statusReason === "CrashLoopBackOff" && status !== "inactive") {
    return "unhealthy";
  }
  return stateByStatus[status];
}

export function summarizeInstances(snapshot: InstanceSnapshot): InstanceSummary | null {
  const total = Math.max(snapshot.desiredInstanceCount, snapshot.instances.length);
  if (total === 0) {
    return null;
  }
  const states = snapshot.instances.map(instanceStateOf);
  const running = snapshot.instances.filter((i) => i.status === "running").length;
  return { state: aggregate(states), running, total };
}

function aggregate(states: InstanceState[]): InstanceState {
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
