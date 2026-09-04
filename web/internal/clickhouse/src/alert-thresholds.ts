export const deployAnomalyThresholds = {
  sigmaK: 4,
  sensitivitySigmaOffsets: {
    low: 1,
    high: -1,
  },
  minimumStddevRatio: 0.1,
  stddevFloors: {
    error_5xx: 0.01,
    error_4xx: 0.01,
    requests: 20,
    egress_bytes: 1_048_576,
    cpu_seconds: 1,
  },
  baselineMinimums: {
    error_5xx: 12,
    error_4xx: 12,
    requests: 12,
    requests_drop: 72,
    egress_bytes: 12,
    cpu_seconds: 12,
  },
  activityFloors: {
    errorExcessFailures: 20,
    requests: 200,
    egressBytes: 104_857_600,
    cpuSeconds: 60,
    memoryUtilization: 0.9,
    oomKilled: 1,
    crashLoop: 1,
  },
  requestDrop: {
    recentLevelFraction: 0.25,
    activityPerBucket: 200,
    minimumActiveBuckets: 9,
    minimumAbsoluteLoss: 200,
  },
  catastrophic: {
    error5xxRatio: 0.5,
    error5xxFailures: 50,
  },
  recovery: {
    memoryUtilization: 0.85,
    sigmaReduction: 1,
    consecutiveWindows: 3,
  },
  maxOpenDurationSeconds: 86_400,
} as const;

export const alertThresholdSigma = deployAnomalyThresholds.sigmaK;
export const alertMinimumStddevRatio = deployAnomalyThresholds.minimumStddevRatio;
export const requestDropRecentLevelFraction =
  deployAnomalyThresholds.requestDrop.recentLevelFraction;
export const requestDropMinimumActiveBuckets =
  deployAnomalyThresholds.requestDrop.minimumActiveBuckets;
export const requestDropActivityFloor = deployAnomalyThresholds.requestDrop.activityPerBucket;
export const requestDropMinimumAbsoluteLoss =
  deployAnomalyThresholds.requestDrop.minimumAbsoluteLoss;

export const alertStddevFloors = deployAnomalyThresholds.stddevFloors;

export const alertBaselineMinimums = deployAnomalyThresholds.baselineMinimums;

export const alertFixedThresholds = {
  memory_utilization: deployAnomalyThresholds.activityFloors.memoryUtilization,
  oom_killed: deployAnomalyThresholds.activityFloors.oomKilled,
  crash_loop: deployAnomalyThresholds.activityFloors.crashLoop,
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
