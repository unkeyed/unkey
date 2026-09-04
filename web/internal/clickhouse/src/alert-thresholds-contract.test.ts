import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { z } from "zod";
import {
  alertBaselineMinimums,
  alertFixedThresholds,
  alertMinimumLifetimeBuckets,
  alertMinimumStddevRatio,
  alertStddevFloors,
  alertThresholdSigma,
  requestDropActivityFloor,
  requestDropMinimumAbsoluteLoss,
  requestDropMinimumActiveBuckets,
  requestDropRecentLevelFraction,
} from "./alert-thresholds";

const detectorThresholdsSchema = z.strictObject({
  sigmaK: z.number(),
  sensitivitySigmaOffsets: z.strictObject({
    low: z.number(),
    high: z.number(),
  }),
  minimumLifetimeBuckets: z.number(),
  minimumStddevRatio: z.number(),
  stddevFloors: z.strictObject({
    error_5xx: z.number(),
    error_4xx: z.number(),
    requests: z.number(),
    egress_bytes: z.number(),
    cpu_seconds: z.number(),
  }),
  baselineMinimums: z.strictObject({
    error_5xx: z.number(),
    error_4xx: z.number(),
    requests: z.number(),
    requests_drop: z.number(),
    egress_bytes: z.number(),
    cpu_seconds: z.number(),
  }),
  activityFloors: z.strictObject({
    errorExcessFailures: z.number(),
    requests: z.number(),
    egressBytes: z.number(),
    cpuSeconds: z.number(),
    memoryUtilization: z.number(),
    oomKilled: z.number(),
    crashLoop: z.number(),
  }),
  requestDrop: z.strictObject({
    recentLevelFraction: z.number(),
    activityPerBucket: z.number(),
    minimumActiveBuckets: z.number(),
    minimumAbsoluteLoss: z.number(),
  }),
  catastrophic: z.strictObject({
    error5xxRatio: z.number(),
    error5xxFailures: z.number(),
  }),
  recovery: z.strictObject({
    memoryUtilization: z.number(),
    sigmaReduction: z.number(),
    consecutiveWindows: z.number(),
  }),
  maxOpenDurationSeconds: z.number(),
});

describe("deploy anomaly threshold contract", () => {
  it("strictly validates every detector threshold and matches dashboard values", () => {
    const path = resolve(
      __dirname,
      "../../../../svc/ctrl/worker/cron/deployanomaly/thresholds.json",
    );
    const decoded: unknown = JSON.parse(readFileSync(path, "utf8"));
    const detectorThresholds = detectorThresholdsSchema.parse(decoded);

    expect({
      sigmaK: alertThresholdSigma,
      minimumLifetimeBuckets: alertMinimumLifetimeBuckets,
      minimumStddevRatio: alertMinimumStddevRatio,
      stddevFloors: alertStddevFloors,
      baselineMinimums: alertBaselineMinimums,
      activityFloors: {
        memoryUtilization: alertFixedThresholds.memory_utilization,
        oomKilled: alertFixedThresholds.oom_killed,
        crashLoop: alertFixedThresholds.crash_loop,
      },
      requestDrop: {
        recentLevelFraction: requestDropRecentLevelFraction,
        activityPerBucket: requestDropActivityFloor,
        minimumActiveBuckets: requestDropMinimumActiveBuckets,
        minimumAbsoluteLoss: requestDropMinimumAbsoluteLoss,
      },
    }).toEqual({
      sigmaK: detectorThresholds.sigmaK,
      minimumLifetimeBuckets: detectorThresholds.minimumLifetimeBuckets,
      minimumStddevRatio: detectorThresholds.minimumStddevRatio,
      stddevFloors: detectorThresholds.stddevFloors,
      baselineMinimums: detectorThresholds.baselineMinimums,
      activityFloors: {
        memoryUtilization: detectorThresholds.activityFloors.memoryUtilization,
        oomKilled: detectorThresholds.activityFloors.oomKilled,
        crashLoop: detectorThresholds.activityFloors.crashLoop,
      },
      requestDrop: detectorThresholds.requestDrop,
    });
  });
});
