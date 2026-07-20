import { describe, expect, it } from "vitest";
import { microCentsToCents, priceDeployUsageMicroCents } from "./deployPricing";

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
