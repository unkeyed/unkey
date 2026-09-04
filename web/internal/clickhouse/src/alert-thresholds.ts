export const alertThresholdSigma = 4;
export const alertMinimumLifetimeBuckets = 12;
export const alertMinimumStddevRatio = 0.1;
export const requestDropMedianFraction = 0.25;

export const alertStddevFloors = {
  error_5xx: 0.01,
  error_4xx: 0.01,
  requests: 20,
  egress_bytes: 1_048_576,
  cpu_seconds: 1,
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
): AlertExpectedBand | null {
  if (lifetimeBuckets < alertMinimumLifetimeBuckets) {
    return null;
  }

  const effectiveStddev = Math.max(
    stddev,
    alertMinimumStddevRatio * mean,
    alertStddevFloors[metric],
  );
  return {
    lowerBound:
      metric === "requests" ? Math.max(0, recentMedian * requestDropMedianFraction) : null,
    upperBound: mean + alertThresholdSigma * effectiveStddev,
  };
}
