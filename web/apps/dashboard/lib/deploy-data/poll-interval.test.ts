import { describe, expect, it } from "vitest";
import { deploymentsPollIntervalMs } from "./poll-interval";

describe("deploymentsPollIntervalMs", () => {
  it("picks the fastest cadence that applies", () => {
    expect(deploymentsPollIntervalMs({ awaiting: false, settling: false, requestedMs: null })).toBe(
      20_000,
    );
    expect(
      deploymentsPollIntervalMs({ awaiting: false, settling: false, requestedMs: 10_000 }),
    ).toBe(10_000);
    expect(
      deploymentsPollIntervalMs({ awaiting: false, settling: true, requestedMs: 10_000 }),
    ).toBe(5_000);
    expect(deploymentsPollIntervalMs({ awaiting: true, settling: true, requestedMs: 10_000 })).toBe(
      2_000,
    );
  });
});
