export function generateRealisticChartData({
  count = 90,
  baseValue = 200,
  variance = 100,
  trend = 0,
  spikeProbability = 0.05,
  spikeMultiplier = 3,
  startTime = Date.now() - 90 * 24 * 60 * 60 * 1000,
  intervalMs = 24 * 60 * 60 * 1000,
  dataKey = "value",
}: {
  count?: number;
  baseValue?: number;
  variance?: number;
  trend?: number;
  spikeProbability?: number;
  spikeMultiplier?: number;
  startTime?: number;
  intervalMs?: number;
  dataKey?: string;
} = {}): Array<{
  originalTimestamp: number;
  [key: string]: number;
}> {
  return Array.from({ length: count }, (_, i) => {
    const trendValue = baseValue + trend * i;
    const randomVariance = (Math.random() - 0.5) * variance * 2;
    const hasSpike = Math.random() < spikeProbability;
    const spikeValue = hasSpike ? randomVariance * spikeMultiplier : 0;
    const value = Math.max(0, Math.floor(trendValue + randomVariance + spikeValue));

    return {
      originalTimestamp: startTime + i * intervalMs,
      [dataKey]: value,
    };
  });
}
