import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { z } from "zod";
import {
  alertBaselineMinimums,
  alertMinimumLifetimeBuckets,
  alertMinimumStddevRatio,
  alertStddevFloors,
  alertThresholdSigma,
  requestDropMedianFraction,
} from "./alert-thresholds";

const detectorThresholdsSchema = z
  .object({
    sigmaK: z.number(),
    minimumLifetimeBuckets: z.number(),
    minimumStddevRatio: z.number(),
    requestDropMedianFraction: z.number(),
    stddevFloors: z
      .object({
        error_5xx: z.number(),
        error_4xx: z.number(),
        requests: z.number(),
        egress_bytes: z.number(),
        cpu_seconds: z.number(),
      })
      .strict(),
    baselineMinimums: z
      .object({
        error_5xx: z.number(),
        error_4xx: z.number(),
        requests: z.number(),
        requests_drop: z.number(),
        egress_bytes: z.number(),
        cpu_seconds: z.number(),
      })
      .strict(),
  })
  .strict();

describe("deploy anomaly threshold contract", () => {
  it("matches the detector's embedded production defaults", () => {
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
      requestDropMedianFraction,
      stddevFloors: alertStddevFloors,
      baselineMinimums: alertBaselineMinimums,
    }).toEqual(detectorThresholds);
  });
});
