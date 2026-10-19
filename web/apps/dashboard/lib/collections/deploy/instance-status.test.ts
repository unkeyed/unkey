import { describe, expect, test } from "vitest";
import { instanceStateOf, summarizeInstances } from "./instance-status";

describe("summarizeInstances", () => {
  test("returns null when no instances are wanted", () => {
    expect(summarizeInstances({ instances: [], desiredInstanceCount: 0 })).toBe(null);
  });

  test("a crash loop is unhealthy even while the instance reads pending", () => {
    const summary = summarizeInstances({
      instances: [{ status: "pending", statusReason: "CrashLoopBackOff" }],
      desiredInstanceCount: 1,
    });
    expect(summary).toEqual({ state: "unhealthy", running: 0, total: 1 });
  });

  test("a crash loop on disabled instances is not unhealthy", () => {
    const summary = summarizeInstances({
      instances: [
        { status: "inactive", statusReason: "CrashLoopBackOff" },
        { status: "inactive", statusReason: null },
      ],
      desiredInstanceCount: 2,
    });
    expect(summary).toEqual({ state: "inactive", running: 0, total: 2 });
  });

  test("counts running instances against the desired count", () => {
    const summary = summarizeInstances({
      instances: [
        { status: "running", statusReason: null },
        { status: "pending", statusReason: null },
      ],
      desiredInstanceCount: 3,
    });
    expect(summary).toEqual({ state: "running", running: 1, total: 3 });
  });
});

describe("instanceStateOf", () => {
  test("only the crash-looping pod reads unhealthy", () => {
    expect(instanceStateOf({ status: "pending", statusReason: "CrashLoopBackOff" })).toBe(
      "unhealthy",
    );
    expect(instanceStateOf({ status: "running", statusReason: null })).toBe("running");
  });
});
