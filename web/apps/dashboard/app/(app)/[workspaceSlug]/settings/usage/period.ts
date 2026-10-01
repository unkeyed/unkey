export type UsageMonthsAgo = 0 | 1;
export type UsagePeriodValue = "current" | "previous";

export type UsagePeriod = {
  value: UsagePeriodValue;
  label: string;
  monthsAgo: UsageMonthsAgo;
  start: number;
  end: number;
};

const MONTHS_AGO: UsageMonthsAgo[] = [1, 0];

export function getUsagePeriods(now: Date): UsagePeriod[] {
  return MONTHS_AGO.map((monthsAgo) => {
    const start = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo, 1);
    const end =
      monthsAgo === 0
        ? now.getTime()
        : Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo + 1, 1);
    return {
      value: monthsAgo === 0 ? "current" : "previous",
      label: new Date(start).toLocaleDateString("en-US", {
        month: "long",
        year: "numeric",
        timeZone: "UTC",
      }),
      monthsAgo,
      start,
      end,
    };
  });
}

export function resolveUsagePeriod(value: string | null, periods: UsagePeriod[]): UsagePeriod {
  return (
    periods.find((period) => period.value === value) ??
    periods.find((period) => period.monthsAgo === 0) ??
    periods[0]
  );
}
