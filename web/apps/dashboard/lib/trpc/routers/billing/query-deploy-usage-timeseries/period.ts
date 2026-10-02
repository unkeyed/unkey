const DAY_MS = 24 * 60 * 60 * 1000;

export function getDeployUsageQueryPeriod({
  now,
  period,
  dayStart,
}: {
  now: Date;
  period: "current" | "previous";
  dayStart?: number;
}): { start: number; end: number } | null {
  const month = now.getUTCMonth() - (period === "previous" ? 1 : 0);
  const monthStart = Date.UTC(now.getUTCFullYear(), month, 1);
  const monthEnd =
    period === "current" ? now.getTime() : Date.UTC(now.getUTCFullYear(), month + 1, 1);

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
