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
