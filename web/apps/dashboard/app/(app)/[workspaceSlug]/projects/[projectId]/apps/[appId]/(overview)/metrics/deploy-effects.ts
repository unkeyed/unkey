import type {
  DeployMarker,
  Grid,
  ResourcePoint,
  SourceResult,
  TrafficPoint,
} from "@/lib/metrics/app-metrics";
import { errorRate } from "@/lib/metrics/views";

export const EFFECT_METRICS = ["errors", "p95", "cpu"] as const;
export type EffectMetric = (typeof EFFECT_METRICS)[number];

export type EffectTone = "worse" | "better" | "neutral";

export type EffectChange = { kind: "percent"; percent: number } | { kind: "from-zero" };

export type DeployEffect = {
  metric: EffectMetric;
  label: string;
  change: EffectChange;
  tone: EffectTone;
};

export type EffectSamples = {
  traffic: SourceResult<"traffic"> | null;
  resources: SourceResult<"resources"> | null;
};

type Span = { readyAtMs: number; untilMs: number | null };

type Effect = {
  label: string;
  measure: (samples: EffectSamples, span: Span) => number | null;
};

function weightedP95(points: ReadonlyArray<TrafficPoint>): number | null {
  let weighted = 0;
  let requests = 0;
  for (const p of points) {
    if (p.p95 !== null) {
      weighted += p.p95 * p.requests;
      requests += p.requests;
    }
  }
  return requests > 0 ? weighted / requests : null;
}

function meanCpu(points: ReadonlyArray<ResourcePoint>): number | null {
  const values = points.flatMap((p) => (p.cpu === null ? [] : [p.cpu]));
  return values.length > 0 ? values.reduce((sum, v) => sum + v, 0) / values.length : null;
}

const EFFECTS: Record<EffectMetric, Effect> = {
  errors: {
    label: "5xx",
    measure: ({ traffic }, span) =>
      traffic && errorRate(pointsIn(traffic, span), traffic.grid.bucketMs / 1000),
  },
  p95: {
    label: "p95",
    measure: ({ traffic }, span) => traffic && weightedP95(pointsIn(traffic, span)),
  },
  cpu: {
    label: "CPU",
    measure: ({ resources }, span) => resources && meanCpu(pointsIn(resources, span)),
  },
};

export const EFFECT_THRESHOLD_PERCENT = 5;

function bucketStart(ms: number, bucketMs: number): number {
  return Math.floor(ms / bucketMs) * bucketMs;
}

// A bucket that holds a deploy's ready time mixes the old and the new deploy,
// so it counts toward neither side.
function pointsIn<P extends { t: number }>(
  result: { grid: Grid; points: ReadonlyArray<P> },
  { readyAtMs, untilMs }: Span,
): P[] {
  const { startMs, endMs, bucketMs } = result.grid;
  const lo = readyAtMs >= startMs ? bucketStart(readyAtMs, bucketMs) + bucketMs : startMs;
  const hi = untilMs === null ? endMs : bucketStart(untilMs, bucketMs);
  return result.points.filter((p) => p.t >= lo && p.t < hi);
}

function changeOf(before: number, after: number): EffectChange {
  if (before === 0) {
    return after === 0 ? { kind: "percent", percent: 0 } : { kind: "from-zero" };
  }
  return { kind: "percent", percent: ((after - before) / before) * 100 };
}

function toneOf(change: EffectChange): EffectTone {
  const rise = change.kind === "from-zero" ? Number.POSITIVE_INFINITY : change.percent;
  if (Math.abs(rise) <= EFFECT_THRESHOLD_PERCENT) {
    return "neutral";
  }
  return rise > 0 ? "worse" : "better";
}

export function deployEffects(
  deploys: ReadonlyArray<DeployMarker>,
  samples: EffectSamples,
): Map<string, DeployEffect[]> {
  return new Map(
    deploys.map((deploy, i) => {
      const previous = deploys[i - 1];
      if (!previous?.serving || !deploy.serving) {
        return [deploy.deploymentId, []];
      }
      const beforeSpan = { readyAtMs: previous.readyAtMs, untilMs: previous.serving.untilMs };
      const afterSpan = { readyAtMs: deploy.readyAtMs, untilMs: deploy.serving.untilMs };
      const effects = EFFECT_METRICS.flatMap((metric): DeployEffect[] => {
        const { label, measure } = EFFECTS[metric];
        const before = measure(samples, beforeSpan);
        const after = measure(samples, afterSpan);
        if (before === null || after === null) {
          return [];
        }
        const change = changeOf(before, after);
        return [{ metric, label, change, tone: toneOf(change) }];
      });
      return [deploy.deploymentId, effects];
    }),
  );
}
