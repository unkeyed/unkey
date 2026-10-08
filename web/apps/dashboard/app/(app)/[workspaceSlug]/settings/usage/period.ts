export type UsagePeriod = "current" | "previous";

export type UsagePeriodOption = {
  value: UsagePeriod;
  label: string;
};

const PERIODS: UsagePeriod[] = ["previous", "current"];

export function getUsagePeriodOptions(now: Date): UsagePeriodOption[] {
  return PERIODS.map((period) => {
    const { year, month } = usagePeriodMonth(period, now);
    const date = new Date(Date.UTC(year, month - 1, 1));

    return {
      value: period,
      label: date.toLocaleDateString("en-US", {
        month: "long",
        year: "numeric",
        timeZone: "UTC",
      }),
    };
  });
}

export function resolveUsagePeriod(value: string | null): UsagePeriod {
  return value === "previous" ? "previous" : "current";
}

export function usagePeriodMonth(period: UsagePeriod, now: Date): { year: number; month: number } {
  const date = new Date(
    Date.UTC(now.getUTCFullYear(), now.getUTCMonth() - (period === "previous" ? 1 : 0), 1),
  );
  return { year: date.getUTCFullYear(), month: date.getUTCMonth() + 1 };
}
