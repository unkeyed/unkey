import { describe, expect, it } from "vitest";
import {
  type ComputeMeterQuantities,
  type DeployMeterRates,
  type DeployUsageQuantities,
  MICRO_CENTS_PER_CENT,
  microCentsToCents,
  priceComputeMeterMicroCents,
  priceDeployMetersCents,
  priceDeployUsageMicroCents,
  projectDeployUsage,
  sumDeployMeterCents,
} from "./deployPricing";

describe("priceDeployMetersCents", () => {
  it("prices each meter on its own", () => {
    const costs = priceDeployMetersCents({
      cpuSeconds: 3600,
      memoryGiBHours: 10,
      diskGiBHours: 100,
      egressGiB: 2,
      activeKeys: 5,
    });
    expect(costs.cpu).toBeCloseTo(3600 * 0.0006944, 6);
    expect(costs.memory).toBeCloseTo(10 * 3600 * 0.0003472, 6);
    expect(costs.egress).toBeCloseTo(2 * 5.0, 6);
    expect(costs.disk).toBeCloseTo(100 * 3600 * 0.000006, 6);
    expect(costs.activeKeys).toBeCloseTo(5 * 0.2, 6);
  });

  it("parts sum to the gross the spend bar shows, within sub-cent rounding", () => {
    const usage: DeployUsageQuantities = {
      cpuSeconds: 12_345,
      memoryGiBHours: 678.9,
      diskGiBHours: 2_596.7,
      egressGiB: 43.3,
      activeKeys: 7,
    };
    const costs = priceDeployMetersCents(usage);
    const partsSum = costs.cpu + costs.memory + costs.egress + costs.disk + costs.activeKeys;
    const gross = priceDeployUsageMicroCents(usage) / MICRO_CENTS_PER_CENT;
    expect(partsSum).toBeCloseTo(gross, 4);
  });
});

describe("sumDeployMeterCents", () => {
  /**
   * Quantities from a real Stripe invoice preview, whose usage lines rounded to
   * 794 + 15873 + 265 + 69 cents for a $170.01 subtotal. Rounding each meter
   * before summing reproduces that; summing the exact amounts and rounding once
   * lands on $170.00, a cent under the invoice.
   */
  const invoiceUsage: DeployUsageQuantities = {
    cpuSeconds: 1_142_941.51892,
    memoryGiBHours: 45_717_656.0512 / 3600,
    diskGiBHours: 11_429_400 / 3600,
    egressGiB: 52.9138888894,
    activeKeys: 0,
  };

  it("rounds each meter before summing, matching the invoice", () => {
    expect(sumDeployMeterCents(invoiceUsage)).toBe(17_001);
  });

  it("differs from rounding the exact total once", () => {
    expect(microCentsToCents(priceDeployUsageMicroCents(invoiceUsage))).toBe(16_999);
  });

  it("agrees with the per-meter figures shown beside it", () => {
    const costs = priceDeployMetersCents(invoiceUsage);
    const shown =
      Math.round(costs.cpu) +
      Math.round(costs.memory) +
      Math.round(costs.egress) +
      Math.round(costs.disk) +
      Math.round(costs.activeKeys);
    expect(sumDeployMeterCents(invoiceUsage)).toBe(shown);
  });

  it("prices zero usage as zero", () => {
    expect(
      sumDeployMeterCents({
        cpuSeconds: 0,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
        activeKeys: 0,
      }),
    ).toBe(0);
  });
});

describe("priceComputeMeterMicroCents", () => {
  const tinySlice: ComputeMeterQuantities = {
    cpuSeconds: 100,
    memoryGiBHours: 0.1,
    diskGiBHours: 1,
    egressGiB: 0.01,
  };

  it("keeps a sub-cent slice non-zero", () => {
    expect(priceComputeMeterMicroCents(tinySlice)).toBeGreaterThan(0);
    expect(sumDeployMeterCents({ ...tinySlice, activeKeys: 0 })).toBe(0);
  });

  it("sums across slices to the same price as the combined usage", () => {
    const a: ComputeMeterQuantities = {
      cpuSeconds: 1234.5,
      memoryGiBHours: 7.25,
      diskGiBHours: 3,
      egressGiB: 0.4,
    };
    const b: ComputeMeterQuantities = {
      cpuSeconds: 987.25,
      memoryGiBHours: 2.5,
      diskGiBHours: 11,
      egressGiB: 1.6,
    };
    const combined: ComputeMeterQuantities = {
      cpuSeconds: a.cpuSeconds + b.cpuSeconds,
      memoryGiBHours: a.memoryGiBHours + b.memoryGiBHours,
      diskGiBHours: a.diskGiBHours + b.diskGiBHours,
      egressGiB: a.egressGiB + b.egressGiB,
    };

    const summed = priceComputeMeterMicroCents(a) + priceComputeMeterMicroCents(b);
    expect(Math.abs(summed - priceComputeMeterMicroCents(combined))).toBeLessThanOrEqual(1);
  });

  it("excludes active keys, which have no project attribution", () => {
    expect(priceComputeMeterMicroCents({ ...tinySlice })).toBe(
      priceDeployUsageMicroCents({ ...tinySlice, activeKeys: 0 }),
    );
  });
});

describe("priceDeployUsageMicroCents", () => {
  it("prices zero usage as zero", () => {
    expect(
      priceDeployUsageMicroCents({
        cpuSeconds: 0,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
        activeKeys: 0,
      }),
    ).toBe(0);
  });

  it("matches the Go catalog rates per meter unit", () => {
    expect(
      priceDeployUsageMicroCents({
        cpuSeconds: 1,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
        activeKeys: 0,
      }),
    ).toBe(694);
    expect(
      priceDeployUsageMicroCents({
        cpuSeconds: 0,
        memoryGiBHours: 1 / 3600,
        diskGiBHours: 0,
        egressGiB: 0,
        activeKeys: 0,
      }),
    ).toBe(347);
    expect(
      priceDeployUsageMicroCents({
        cpuSeconds: 0,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 1,
        activeKeys: 0,
      }),
    ).toBe(5_000_000);
    expect(
      priceDeployUsageMicroCents({
        cpuSeconds: 0,
        memoryGiBHours: 0,
        diskGiBHours: 1 / 3600,
        egressGiB: 0,
        activeKeys: 0,
      }),
    ).toBe(6);
    expect(
      priceDeployUsageMicroCents({
        cpuSeconds: 0,
        memoryGiBHours: 0,
        diskGiBHours: 0,
        egressGiB: 0,
        activeKeys: 1,
      }),
    ).toBe(200_000);
  });

  it("sums meters like the spend-cap worker", () => {
    const micro = priceDeployUsageMicroCents({
      cpuSeconds: 0,
      memoryGiBHours: 0,
      diskGiBHours: 0,
      egressGiB: 10,
      activeKeys: 100,
    });
    expect(micro).toBe(70 * 1_000_000);
  });
});

describe("microCentsToCents", () => {
  it("truncates sub-cent fractions for display", () => {
    expect(microCentsToCents(4_711_499_999)).toBe(4_711);
  });
});

describe("projectDeployUsage", () => {
  const DAY = 24 * 60 * 60 * 1000;
  const trailingWindowMs = 7 * DAY;

  const mtd: DeployUsageQuantities = {
    cpuSeconds: 100,
    memoryGiBHours: 20,
    diskGiBHours: 10,
    egressGiB: 5,
    activeKeys: 42,
  };
  const trailing: DeployMeterRates = {
    cpuSeconds: 70,
    memoryGiBHours: 14,
    diskGiBHours: 7,
    egressGiB: 3.5,
  };

  it("adds the trailing run-rate applied over the remaining period", () => {
    // 21 days remaining is 3x the 7-day trailing window, so add 3x trailing.
    const projected = projectDeployUsage(mtd, trailing, trailingWindowMs, 21 * DAY);
    expect(projected.cpuSeconds).toBe(100 + 70 * 3);
    expect(projected.memoryGiBHours).toBe(20 + 14 * 3);
    expect(projected.diskGiBHours).toBe(10 + 7 * 3);
    expect(projected.egressGiB).toBe(5 + 3.5 * 3);
  });

  it("holds active keys flat, since a distinct count is not a rate", () => {
    const projected = projectDeployUsage(mtd, trailing, trailingWindowMs, 21 * DAY);
    expect(projected.activeKeys).toBe(42);
  });

  it("returns month-to-date unchanged when no time remains", () => {
    expect(projectDeployUsage(mtd, trailing, trailingWindowMs, 0)).toEqual(mtd);
    expect(projectDeployUsage(mtd, trailing, trailingWindowMs, -DAY)).toEqual(mtd);
  });

  it("returns month-to-date unchanged with no trailing window", () => {
    expect(projectDeployUsage(mtd, trailing, 0, 21 * DAY)).toEqual(mtd);
  });

  it("adds nothing when the trailing window saw no usage", () => {
    const idle: DeployMeterRates = {
      cpuSeconds: 0,
      memoryGiBHours: 0,
      diskGiBHours: 0,
      egressGiB: 0,
    };
    expect(projectDeployUsage(mtd, idle, trailingWindowMs, 21 * DAY)).toEqual(mtd);
  });
});
