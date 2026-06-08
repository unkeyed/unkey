export function intervalToSeconds(interval: string): number {
  const num = Number.parseInt(interval, 10);
  if (interval.endsWith("h")) {
    return num * 3600;
  }
  if (interval.endsWith("m")) {
    return num * 60;
  }
  return num;
}

export function secondsToInterval(seconds: number): string {
  if (seconds % 3600 === 0) {
    return `${seconds / 3600}h`;
  }
  if (seconds % 60 === 0) {
    return `${seconds / 60}m`;
  }
  return `${seconds}s`;
}
