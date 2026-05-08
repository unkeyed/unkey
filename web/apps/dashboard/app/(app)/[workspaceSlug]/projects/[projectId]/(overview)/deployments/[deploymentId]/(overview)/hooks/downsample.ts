export type TimeseriesData = {
  originalTimestamp: number;
  y: number;
  total: number;
};

const TARGET_BUCKETS = 24;

/** 6h window at 15s buckets produces ~360 points. Reduce to ~24 using peak (max) selection so spikes stay visible. */
export function downsample(points: TimeseriesData[]): TimeseriesData[] {
  if (points.length <= TARGET_BUCKETS) {
    return points;
  }

  const bucketSize = Math.ceil(points.length / TARGET_BUCKETS);
  const result: TimeseriesData[] = [];

  for (let i = 0; i < points.length; i += bucketSize) {
    const chunk = points.slice(i, i + bucketSize);
    let maxPoint = chunk[0];
    for (let j = 1; j < chunk.length; j++) {
      if (chunk[j].y > maxPoint.y) {
        maxPoint = chunk[j];
      }
    }
    result.push(maxPoint);
  }

  return result;
}
