export type UsageMonthsAgo = 0 | 1;

export type UsagePeriod = {
  value: string;
  label: string;
  monthsAgo: UsageMonthsAgo;
  start: number;
  end: number;
};

const MONTHS_AGO: UsageMonthsAgo[] = [0, 1];

export function getUsagePeriods(now: Date): UsagePeriod[] {
  return MONTHS_AGO.map((monthsAgo) => {
    const start = Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo, 1);
    const end =
      monthsAgo === 0
        ? now.getTime()
        : Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - monthsAgo + 1, 1);
    const date = new Date(start);

    return {
      value: `${String(date.getUTCMonth() + 1).padStart(2, "0")}${date.getUTCFullYear()}`,
      label: monthsAgo === 0 ? "Current month" : "Last month",
      monthsAgo,
      start,
      end,
    };
  });
}

export function resolveUsagePeriod(value: string | null, periods: UsagePeriod[]): UsagePeriod {
  return periods.find((period) => period.value === value) ?? periods[0];
}
