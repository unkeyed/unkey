export type UsagePeriod = "current" | "previous";

export type UsagePeriodOption = {
  value: UsagePeriod;
  label: string;
};

const PERIODS: UsagePeriod[] = ["previous", "current"];

export function getUsagePeriodOptions(now: Date): UsagePeriodOption[] {
  return PERIODS.map((period) => {
    const month = now.getUTCMonth() - (period === "previous" ? 1 : 0);
    const date = new Date(Date.UTC(now.getUTCFullYear(), month, 1));

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
