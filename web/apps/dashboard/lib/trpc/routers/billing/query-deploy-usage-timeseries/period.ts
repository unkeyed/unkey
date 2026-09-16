const DAY_MS = 24 * 60 * 60 * 1000;

export function getDeployUsageQueryPeriod({
  now,
  monthsAgo,
  dayStart,
}: {
  now: Date;
  monthsAgo: 0 | 1 | 2;
  dayStart?: number;
}): { start: number; end: number } | null {
  const monthStart = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo, 1);
  const monthEnd =
    monthsAgo === 0
      ? now.getTime()
      : Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo + 1, 1);

  if (dayStart === undefined) {
    return { start: monthStart, end: monthEnd };
  }
  if (!Number.isInteger(dayStart) || dayStart % DAY_MS !== 0) {
    return null;
  }
  if (dayStart < monthStart || dayStart >= monthEnd) {
    return null;
  }

  return {
    start: dayStart,
    end: Math.min(dayStart + DAY_MS, monthEnd),
  };
}
