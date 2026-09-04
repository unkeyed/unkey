import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { z } from "zod";
import { deployAnomalyThresholds } from "./alert-thresholds";

const detectorThresholdsSchema = z.strictObject({
  sigmaK: z.number(),
  sensitivitySigmaOffsets: z.strictObject({
    low: z.number(),
    high: z.number(),
  }),
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

const thresholdsPath = resolve(
  __dirname,
  "../../../../svc/ctrl/worker/cron/deployanomaly/thresholds.json",
);

describe("deploy anomaly threshold contract", () => {
  it("strictly validates and mirrors every detector threshold", () => {
    const decoded: unknown = JSON.parse(readFileSync(thresholdsPath, "utf8"));

    expect(deployAnomalyThresholds).toEqual(detectorThresholdsSchema.parse(decoded));
  });

  it("rejects a nested detector threshold difference", () => {
    const decoded: unknown = JSON.parse(readFileSync(thresholdsPath, "utf8"));
    const detectorThresholds = detectorThresholdsSchema.parse(decoded);
    const mutatedThresholds = detectorThresholdsSchema.parse({
      ...detectorThresholds,
      catastrophic: {
        ...detectorThresholds.catastrophic,
        error5xxRatio: 0.51,
      },
    });

    expect(deployAnomalyThresholds).not.toEqual(mutatedThresholds);
  });
});
