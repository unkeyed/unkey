import type { InstanceStatus } from "@/lib/collections/deploy/instance-status";
import type { Loadable } from "@/lib/loadable/loadable";
import type { LastExit } from "@/lib/types/deploy";
import type { MetricSource, SourcePoint } from "@unkey/clickhouse/src/app-metrics/catalog";
import { DAY_MS, type Grid } from "@unkey/clickhouse/src/app-metrics/grid";
import type { ResourcePoint } from "@unkey/clickhouse/src/app-metrics/resources";
import type { TrafficPoint } from "@unkey/clickhouse/src/app-metrics/traffic";
import { z } from "zod";

export {
  METRIC_CATALOG,
  METRIC_IDS,
  type MetricId,
  type MetricSource,
  type SeriesKey,
  type SourcePoint,
  rollupsServe,
} from "@unkey/clickhouse/src/app-metrics/catalog";
export {
  DAY_MS,
  type Grid,
  RANGE_PLANS,
  RANGE_PRESETS,
  type RangePreset,
  clampedGrid,
  latestGrid,
  nextTickMs,
} from "@unkey/clickhouse/src/app-metrics/grid";
export type { ResourcePoint, TrafficPoint };

export const environmentRefSchema = z.union([
  z.object({ id: z.string() }),
  z.object({ kind: z.literal("production") }),
]);

export type EnvironmentRef = z.infer<typeof environmentRefSchema>;

export const PRODUCTION: EnvironmentRef = { kind: "production" };

// Production is addressed by kind, so the overview glance can query before the
// environment list loads and still share its cache with the metrics page.
export function environmentRefOf(environment: {
  id: string;
  kind: "production" | "preview";
}): EnvironmentRef {
  return environment.kind === "production" ? PRODUCTION : { id: environment.id };
}

export function environmentKeyOf(ref: EnvironmentRef): string {
  return "id" in ref ? ref.id : ref.kind;
}

export function spansDays(grid: Pick<Grid, "startMs" | "endMs">): boolean {
  return grid.endMs - grid.startMs >= DAY_MS;
}

export type SourceResult<S extends MetricSource> = { grid: Grid; points: SourcePoint<S>[] };

export type Sources = { [S in MetricSource]: Loadable<SourceResult<S>> };

export type InstanceSeries = {
  grid: Grid;
  instances: Array<{ instance: string; points: ResourcePoint[] }>;
};

export type DeploySource =
  | {
      kind: "git";
      sha: string;
      message: string | null;
      authorHandle: string | null;
      avatarUrl: string | null;
    }
  | { kind: "image"; ref: string };

// The first marker may start before grid.startMs: it was serving when the range began.
// serving is null in preview environments, where no single deploy serves the
// environment, so there is no span to shade or compare.
export type DeployMarker = {
  deploymentId: string;
  readyAtMs: number;
  serving: { untilMs: number | null } | null;
  source: DeploySource | null;
};

export type ReplicaTarget = {
  k8sName: string;
  cpuMillicores: number;
  memoryMib: number;
  storageMib: number;
};

export type DeployRole = "current" | "rolling-out" | "previous";

export type InstanceRow = {
  instanceId: string;
  region: { name: string };
  status: InstanceStatus;
  role: DeployRole;
  lastExit: LastExit | null;
  deployment: { id: string; source: DeploySource | null };
  replica: ReplicaTarget;
};

export type InstanceEventKind =
  | "crashed"
  | "oom"
  | "crashloop"
  | "image-pull"
  | "config"
  | "completed"
  | "started";

export type InstanceEvent = {
  id: string;
  atMs: number;
  instance: string;
  instanceId: string | null;
  region: string;
  deploymentId: string;
  kind: InstanceEventKind;
  reason: string;
  exitCode: number | null;
};

const CLAMPED_RETRY_MS = 2_000;
const MAX_CLAMPED_RETRIES = 2;

// The server clamps endMs to its own clock, so a browser clock a few seconds ahead
// gets the previous bucket back. Retries cover small skew; a larger skew keeps the
// previous bucket until the next tick instead of polling every 2s.
export function clampedRetryMs(asked: Grid, served: Grid, fetchCount: number): number | false {
  const behindMs = asked.endMs - served.endMs;
  const clamped = behindMs > 0 && behindMs <= asked.bucketMs;
  return clamped && fetchCount <= MAX_CLAMPED_RETRIES ? CLAMPED_RETRY_MS : false;
}
