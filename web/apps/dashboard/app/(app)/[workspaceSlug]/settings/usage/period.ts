import { formatPeriod } from "@/lib/fmt";

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

export type UsagePeriodRange = {
  label: string;
  daysLeft: number | null;
};

const DAY_MS = 24 * 60 * 60 * 1000;

export function usagePeriodRange(period: UsagePeriod, now: Date): UsagePeriodRange {
  const { year, month } = usagePeriodMonth(period, now);
  const start = Date.UTC(year, month - 1, 1);
  const reset = Date.UTC(year, month, 1);

  return {
    label: formatPeriod(start, reset - DAY_MS),
    daysLeft: period === "current" ? Math.ceil((reset - now.getTime()) / DAY_MS) : null,
  };
}
