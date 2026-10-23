import type { InstanceStatus } from "@/lib/collections/deploy/instance-status";
import type { Loadable } from "@/lib/loadable/loadable";
import type { LastExit } from "@/lib/types/deploy";
import type { MetricSource, SourcePoint } from "@unkey/clickhouse/src/app-metrics/catalog";
import type { Grid } from "@unkey/clickhouse/src/app-metrics/grid";
import type { ResourcePoint } from "@unkey/clickhouse/src/app-metrics/resources";
import type { TrafficPoint } from "@unkey/clickhouse/src/app-metrics/traffic";

// Pod names travel in a GET query string, so the series names at most this many
export const INSTANCE_SERIES_MAX = 200;

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
