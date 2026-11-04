import {
  type InstanceState,
  groupInstancesByRegion,
  instanceStateOf,
  instancesText,
} from "@/lib/collections/deploy/instance-status";
import { exitSummary } from "@/lib/collections/deploy/last-exit";
import type { Deployment } from "@/lib/deploy-data";
import type { StepResolution } from "./stages-model";

type DeploymentInstance = Deployment["instances"][number];

export type DiagramInstance = {
  id: string;
  state: InstanceState;
  detail: string | null;
  restarts: number;
};

export type RegionContent =
  | { kind: "instances"; instances: DiagramInstance[]; waiting: number }
  | { kind: "waiting" }
  | { kind: "skeleton"; count: number };

export type RegionGroup = {
  key: string;
  name: string | null;
  state: InstanceState;
  running: number;
  total: number;
  content: RegionContent;
};

export type InstancesView =
  | { kind: "diagram"; regions: RegionGroup[] }
  | { kind: "empty"; message: string }
  | { kind: "none" };

export type InstancesInput = Pick<
  Deployment,
  "instances" | "desiredRegions" | "targetRegions" | "desiredState" | "desiredInstanceCount"
> & { stage: StepResolution["status"] };

export function instanceDetail(instance: DeploymentInstance): string | null {
  if (instanceStateOf(instance) !== "unhealthy" || !instance.lastExit) {
    return null;
  }
  return exitSummary(instance.lastExit);
}

export function resolveInstancesView(input: InstancesInput): InstancesView {
  if (input.stage === "pending") {
    return { kind: "diagram", regions: skeletonRegions(input) };
  }
  if (input.instances.length === 0) {
    if (input.stage === "started" && input.desiredState === "running") {
      return { kind: "diagram", regions: skeletonRegions(input) };
    }
    if (input.stage === "error") {
      return { kind: "none" };
    }
    return { kind: "empty", message: instancesText(null, input.desiredState) };
  }
  return { kind: "diagram", regions: liveRegions(input) };
}

function regionOrder(desiredRegions: InstancesInput["desiredRegions"]) {
  const desiredOrder = desiredRegions.map((d) => d.region.name);
  const rank = (name: string) => {
    const index = desiredOrder.indexOf(name);
    return index === -1 ? desiredOrder.length : index;
  };
  return (a: string, b: string) => rank(a) - rank(b) || a.localeCompare(b);
}

function skeletonRegions({
  targetRegions,
  desiredRegions,
  desiredInstanceCount,
}: InstancesInput): RegionGroup[] {
  if (targetRegions.length === 0) {
    return [skeletonRegion(null, desiredInstanceCount)];
  }
  const byOrder = regionOrder(desiredRegions);
  return targetRegions
    .toSorted((a, b) => byOrder(a.region.name, b.region.name))
    .map((target) => skeletonRegion(target.region.name, target.replicas));
}

function skeletonRegion(name: string | null, replicas: number): RegionGroup {
  const count = Math.max(replicas, 1);
  return {
    key: name ?? "expected",
    name,
    state: "starting",
    running: 0,
    total: count,
    content: { kind: "skeleton", count },
  };
}

function liveRegions(input: InstancesInput): RegionGroup[] {
  const byOrder = regionOrder(input.desiredRegions);
  return groupInstancesByRegion(input)
    .toSorted((a, b) => byOrder(a.name, b.name))
    .map(({ name, instances, state, running, total, waiting }) => ({
      key: name,
      name,
      state,
      running,
      total,
      content:
        instances.length === 0
          ? { kind: "waiting" }
          : {
              kind: "instances",
              instances: instances
                .toSorted((a, b) => a.id.localeCompare(b.id))
                .map((i) => ({
                  id: i.id,
                  state: instanceStateOf(i),
                  detail: instanceDetail(i),
                  restarts: i.lastExit?.restartCount ?? 0,
                })),
              waiting,
            },
    }));
}

export type EnteredTrack = {
  state: InstanceState;
  restarts: number;
  entered: InstanceState | null;
};

export function trackEntered(
  prev: EnteredTrack,
  next: { state: InstanceState; restarts: number },
): EnteredTrack {
  if (prev.state === next.state) {
    return prev.restarts === next.restarts ? prev : { ...prev, restarts: next.restarts };
  }
  const flappedBetweenBackoffs = next.state === "running" && next.restarts > prev.restarts;
  return { ...next, entered: flappedBetweenBackoffs ? null : next.state };
}
