/** Mirrors tools/pricing/catalog.go CentsPerUnit; keep in sync with deploybilling/billing.go. */
const CENTS_PER_CPU_SECOND = 0.0006944;
const CENTS_PER_MEMORY_GIB_SECOND = 0.0003472;
const CENTS_PER_EGRESS_GIB = 5.0;
const CENTS_PER_DISK_GIB_SECOND = 0.000006;
const CENTS_PER_ACTIVE_KEY = 0.2;

const SECONDS_PER_HOUR = 3600;

/** Same fixed-point scale as deploybilling.MicroCentsPerCent. */
export const MICRO_CENTS_PER_CENT = 1_000_000;

export type DeployUsageQuantities = {
  cpuSeconds: number;
  memoryGiBHours: number;
  diskGiBHours: number;
  egressGiB: number;
  activeKeys: number;
};

/**
 * Prices month-to-date Deploy usage in micro-cents, matching the spend-cap
 * worker's PriceMicroCents so the dashboard bar tracks what alerts enforce.
 */
export function priceDeployUsageMicroCents(usage: DeployUsageQuantities): number {
  const memoryGiBSeconds = usage.memoryGiBHours * SECONDS_PER_HOUR;
  const diskGiBSeconds = usage.diskGiBHours * SECONDS_PER_HOUR;

  const cents =
    usage.cpuSeconds * CENTS_PER_CPU_SECOND +
    memoryGiBSeconds * CENTS_PER_MEMORY_GIB_SECOND +
    usage.egressGiB * CENTS_PER_EGRESS_GIB +
    diskGiBSeconds * CENTS_PER_DISK_GIB_SECOND +
    usage.activeKeys * CENTS_PER_ACTIVE_KEY;

  return Math.round(cents * MICRO_CENTS_PER_CENT);
}

/** Converts micro-cents to whole cents for formatDollars and the spend budget bar. */
export function microCentsToCents(microCents: number): number {
  return Math.floor(microCents / MICRO_CENTS_PER_CENT);
}

/** Per-meter spend in cents, so the card can show what each usage line costs. */
export type DeployMeterCostsCents = {
  cpu: number;
  memory: number;
  egress: number;
  disk: number;
  activeKeys: number;
};

/**
 * Prices each meter on its own, using the same per-unit rates as
 * priceDeployUsageMicroCents, so the usage row can attribute a dollar figure to
 * every line. The parts sum to the gross the spend bar shows within sub-cent
 * rounding. Cents are kept fractional; formatDollars rounds them for display.
 */
export function priceDeployMetersCents(usage: DeployUsageQuantities): DeployMeterCostsCents {
  return {
    cpu: usage.cpuSeconds * CENTS_PER_CPU_SECOND,
    memory: usage.memoryGiBHours * SECONDS_PER_HOUR * CENTS_PER_MEMORY_GIB_SECOND,
    egress: usage.egressGiB * CENTS_PER_EGRESS_GIB,
    disk: usage.diskGiBHours * SECONDS_PER_HOUR * CENTS_PER_DISK_GIB_SECOND,
    activeKeys: usage.activeKeys * CENTS_PER_ACTIVE_KEY,
  };
}

/** The duration meters, which accrue at a rate set by what's currently deployed. */
export type DeployMeterRates = {
  cpuSeconds: number;
  memoryGiBHours: number;
  diskGiBHours: number;
  egressGiB: number;
};

/**
 * Projects usage to the end of the billing period: what has accrued so far plus
 * a trailing-window run-rate applied over the time remaining. The rate comes
 * from a recent window rather than the whole period so a mid-period scale-up
 * isn't diluted by idle early days, and an almost-empty period start doesn't
 * blow the number up.
 *
 * The four duration meters (cpu, memory, disk, egress) project on their rate.
 * Active keys is a distinct-key count, not a rate, so it is held flat at its
 * month-to-date value. Returns month-to-date unchanged when there is no time
 * left or no trailing window to derive a rate from.
 */
export function projectDeployUsage(
  monthToDate: DeployUsageQuantities,
  trailing: DeployMeterRates,
  trailingWindowMs: number,
  remainingMs: number,
): DeployUsageQuantities {
  if (trailingWindowMs <= 0 || remainingMs <= 0) {
    return monthToDate;
  }
  // Fraction of the trailing window that the remaining period represents: the
  // trailing usage scaled to that span is the projected additional usage.
  const scale = remainingMs / trailingWindowMs;
  return {
    cpuSeconds: monthToDate.cpuSeconds + trailing.cpuSeconds * scale,
    memoryGiBHours: monthToDate.memoryGiBHours + trailing.memoryGiBHours * scale,
    diskGiBHours: monthToDate.diskGiBHours + trailing.diskGiBHours * scale,
    egressGiB: monthToDate.egressGiB + trailing.egressGiB * scale,
    activeKeys: monthToDate.activeKeys,
  };
}
