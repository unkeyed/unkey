import { describe, expect, it } from "vitest";
import { priceUsageQuantitiesCents } from "./compute-tree";

describe("priceUsageQuantitiesCents", () => {
  it("prices the display units with the Deploy meter rates", () => {
    expect(
      priceUsageQuantitiesCents({
        cpuHours: 1,
        memoryGiBHours: 1,
        egressGiB: 1,
        diskGiBHours: 1,
      }),
    ).toEqual({
      cpu: 2.49984,
      memory: 1.24992,
      egress: 5,
      disk: 0.0216,
    });
  });
});
