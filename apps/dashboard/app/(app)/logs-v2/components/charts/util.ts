/**
 * Generates mock time series data for logs visualization
 * @param hours Number of hours of data to generate
 * @param intervalMinutes Interval between data points in minutes
 * @returns Array of LogsTimeseriesDataPoint
 */
export function generateMockLogsData(hours = 24, intervalMinutes = 5) {
  const now = new Date();
  const points = Math.floor((hours * 60) / intervalMinutes);
  const data = [];

  for (let i = points - 1; i >= 0; i--) {
    const timestamp = new Date(now.getTime() - i * intervalMinutes * 60 * 1000);

    const success = Math.floor(Math.random() * 50) + 20;
    const error = Math.floor(Math.random() * 30);
    const warning = Math.floor(Math.random() * 25);

    data.push({
      x: Math.floor(timestamp.getTime()),
      displayX: timestamp.toISOString(),
      originalTimestamp: timestamp.toISOString(),
      success,
      error,
      warning,
      total: success + error + warning,
    });
  }

  return data;
}
