import { describe, expect, it } from "vitest";
import { deploymentsPollIntervalMs } from "./poll-interval";

describe("deploymentsPollIntervalMs", () => {
  it("picks the fastest cadence that applies", () => {
    const idle = { awaiting: false, settling: false, requestedMs: null, unchangedPolls: 0 };
    expect(deploymentsPollIntervalMs(idle)).toBe(20_000);
    expect(deploymentsPollIntervalMs({ ...idle, requestedMs: 10_000 })).toBe(10_000);
    expect(deploymentsPollIntervalMs({ ...idle, requestedMs: 10_000, settling: true })).toBe(5_000);
    expect(
      deploymentsPollIntervalMs({ ...idle, requestedMs: 10_000, settling: true, awaiting: true }),
    ).toBe(2_000);
  });

  it("backs off idle polls that change nothing, up to a minute", () => {
    const idle = { awaiting: false, settling: false, requestedMs: null };
    expect(deploymentsPollIntervalMs({ ...idle, unchangedPolls: 1 })).toBe(40_000);
    expect(deploymentsPollIntervalMs({ ...idle, unchangedPolls: 2 })).toBe(60_000);
    expect(deploymentsPollIntervalMs({ ...idle, unchangedPolls: 9 })).toBe(60_000);
  });

  it("never backs off while something moves or a page asks for a cadence", () => {
    expect(
      deploymentsPollIntervalMs({
        awaiting: false,
        settling: true,
        requestedMs: null,
        unchangedPolls: 5,
      }),
    ).toBe(5_000);
    expect(
      deploymentsPollIntervalMs({
        awaiting: true,
        settling: false,
        requestedMs: null,
        unchangedPolls: 5,
      }),
    ).toBe(2_000);
    expect(
      deploymentsPollIntervalMs({
        awaiting: false,
        settling: false,
        requestedMs: 10_000,
        unchangedPolls: 5,
      }),
    ).toBe(10_000);
  });
});
