export const alertThresholdSigma = 4;
export const alertMinimumLifetimeBuckets = 12;
export const alertMinimumStddevRatio = 0.1;
export const requestDropRecentLevelFraction = 0.25;
export const requestDropMinimumActiveBuckets = 9;
export const requestDropActivityFloor = 200;
export const requestDropMinimumAbsoluteLoss = 200;

export const alertStddevFloors = {
  error_5xx: 0.01,
  error_4xx: 0.01,
  requests: 20,
  egress_bytes: 1_048_576,
  cpu_seconds: 1,
} as const;

export const alertBaselineMinimums = {
  error_5xx: 12,
  error_4xx: 12,
  requests: 12,
  requests_drop: 72,
  egress_bytes: 12,
  cpu_seconds: 12,
} as const;

export const alertFixedThresholds = {
  memory_utilization: 0.9,
  oom_killed: 1,
  crash_loop: 1,
} as const;

export type SigmaAlertMetric = keyof typeof alertStddevFloors;

export type AlertExpectedBand = {
  lowerBound: number | null;
  upperBound: number;
};

export function calculateAlertExpectedBand(
  metric: SigmaAlertMetric,
  mean: number,
  stddev: number,
  lifetimeBuckets: number,
  recentMedian = mean,
  recentActiveBuckets = requestDropMinimumActiveBuckets,
): AlertExpectedBand | null {
  if (lifetimeBuckets < alertBaselineMinimums[metric]) {
    return null;
  }

  const effectiveStddev = Math.max(
    stddev,
    alertMinimumStddevRatio * mean,
    alertStddevFloors[metric],
  );
  const requestDropThreshold = Math.max(
    0,
    Math.min(
      recentMedian * requestDropRecentLevelFraction,
      recentMedian - requestDropMinimumAbsoluteLoss,
    ),
  );
  return {
    lowerBound:
      metric === "requests" &&
      lifetimeBuckets >= alertBaselineMinimums.requests_drop &&
      recentActiveBuckets >= requestDropMinimumActiveBuckets
        ? requestDropThreshold
        : null,
    upperBound: mean + alertThresholdSigma * effectiveStddev,
  };
}
