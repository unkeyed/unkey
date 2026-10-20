import { describe, expect, test } from "vitest";
import { instanceStateOf, summarizeInstances } from "./instance-status";

const region = (replicas: number) => ({ replicas });

describe("summarizeInstances", () => {
  test("returns null when no instances are wanted", () => {
    expect(summarizeInstances({ instances: [], desiredInstanceCount: 0, targetRegions: [] })).toBe(
      null,
    );
  });

  test("a crash loop is unhealthy even while the instance reads pending", () => {
    const summary = summarizeInstances({
      instances: [{ status: "pending", statusReason: "CrashLoopBackOff" }],
      desiredInstanceCount: 1,
      targetRegions: [],
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
      targetRegions: [],
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
      targetRegions: [],
    });
    expect(summary).toEqual({ state: "running", running: 1, total: 3 });
  });

  test("the total is the sum of target region replicas when regions are set", () => {
    const summary = summarizeInstances({
      instances: [{ status: "running", statusReason: null }],
      desiredInstanceCount: 1,
      targetRegions: [region(2), region(3)],
    });
    expect(summary).toEqual({ state: "running", running: 1, total: 5 });
  });

  test("the total never drops below the instances that exist", () => {
    const summary = summarizeInstances({
      instances: [
        { status: "running", statusReason: null },
        { status: "running", statusReason: null },
        { status: "running", statusReason: null },
      ],
      desiredInstanceCount: 1,
      targetRegions: [region(2)],
    });
    expect(summary).toEqual({ state: "running", running: 3, total: 3 });
  });
});

describe("instanceStateOf", () => {
  test("only the crash-looping pod reads unhealthy", () => {
    expect(instanceStateOf({ status: "pending", statusReason: "CrashLoopBackOff" })).toBe(
      "unhealthy",
    );
    expect(instanceStateOf({ status: "running", statusReason: null })).toBe("running");
  });

  test.each([
    "CrashLoopBackOff",
    "ImagePullBackOff",
    "ErrImagePull",
    "CreateContainerConfigError",
    "InvalidImageName",
  ])("%s on a pending instance reads unhealthy", (statusReason) => {
    expect(instanceStateOf({ status: "pending", statusReason })).toBe("unhealthy");
  });

  test("a fatal reason on a disabled instance stays disabled", () => {
    expect(instanceStateOf({ status: "inactive", statusReason: "ImagePullBackOff" })).toBe(
      "inactive",
    );
  });

  test("a transient waiting reason does not read unhealthy", () => {
    expect(instanceStateOf({ status: "pending", statusReason: "ContainerCreating" })).toBe(
      "starting",
    );
  });
});
