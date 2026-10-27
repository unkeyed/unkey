import { describe, expect, test } from "vitest";
import type { Deployment } from "./deployments";
import {
  type InstancesSource,
  instanceStateOf,
  instancesText,
  summarizeInstances,
} from "./instance-status";

type Instance = Deployment["instances"][number];

const instance = (region: string, overrides: Partial<Instance> = {}): Instance => ({
  id: `ins_${region}`,
  region: { id: `reg_${region}`, name: region, platform: "aws" },
  status: "running",
  lastExit: null,
  ...overrides,
});

const target = (name: string, replicas: number) => ({
  region: { id: `reg_${name}`, name, platform: "aws" },
  replicas,
});

const waiting = (
  statusReason: string,
): NonNullable<Deployment["instances"][number]["lastExit"]> => ({
  restartCount: 0,
  exitCode: null,
  signal: null,
  reason: null,
  finishedAt: null,
  statusReason,
});

const source = (overrides: Partial<InstancesSource> = {}): InstancesSource => ({
  instances: [],
  targetRegions: [],
  desiredState: "running",
  ...overrides,
});

describe("summarizeInstances", () => {
  test("returns null when no instances exist or are wanted", () => {
    expect(summarizeInstances(source())).toBe(null);
  });

  test("a crash loop is unhealthy even while the instance reads pending", () => {
    const summary = summarizeInstances(
      source({
        instances: [
          instance("us-east-1", { status: "pending", lastExit: waiting("CrashLoopBackOff") }),
        ],
      }),
    );
    expect(summary).toEqual({ state: "unhealthy", running: 0, total: 1 });
  });

  test("a crash loop on disabled instances is not unhealthy", () => {
    const summary = summarizeInstances(
      source({
        instances: [
          instance("us-east-1", { status: "inactive", lastExit: waiting("CrashLoopBackOff") }),
          instance("eu-central-1", { status: "inactive" }),
        ],
      }),
    );
    expect(summary).toEqual({ state: "inactive", running: 0, total: 2 });
  });

  test("the total is the sum of target region replicas, and a short region is still starting", () => {
    const summary = summarizeInstances(
      source({
        instances: [instance("us-east-1")],
        targetRegions: [target("us-east-1", 2), target("eu-central-1", 3)],
      }),
    );
    expect(summary).toEqual({ state: "starting", running: 1, total: 5 });
  });

  test("the total never drops below the instances that exist", () => {
    const summary = summarizeInstances(
      source({
        instances: [
          instance("us-east-1", { id: "ins_a" }),
          instance("us-east-1", { id: "ins_b" }),
          instance("us-east-1", { id: "ins_c" }),
        ],
        targetRegions: [target("us-east-1", 2)],
      }),
    );
    expect(summary).toEqual({ state: "running", running: 3, total: 3 });
  });

  test("a stopped deployment wants no replicas, so only its rows count", () => {
    expect(
      summarizeInstances(
        source({ desiredState: "stopped", targetRegions: [target("us-east-1", 2)] }),
      ),
    ).toBe(null);
    expect(
      summarizeInstances(
        source({
          desiredState: "stopped",
          instances: [instance("us-east-1", { status: "inactive" })],
          targetRegions: [target("us-east-1", 2)],
        }),
      ),
    ).toEqual({ state: "inactive", running: 0, total: 1 });
  });
});

describe("summarizeInstances after a deployment stops serving", () => {
  test("a stopped deployment whose topology still lists regions counts only its rows", () => {
    expect(
      summarizeInstances(
        source({
          desiredState: "stopped",
          instances: [
            instance("us-east-1", { id: "ins_a", status: "inactive" }),
            instance("us-east-1", { id: "ins_b", status: "inactive" }),
          ],
          targetRegions: [target("us-east-1", 5)],
        }),
      ),
    ).toEqual({ state: "inactive", running: 0, total: 2 });
  });

  test("a stopped deployment with no topology and no rows has no summary", () => {
    expect(summarizeInstances(source({ desiredState: "stopped" }))).toBe(null);
  });

  test("a superseded deployment has no running topology, so it never borrows the app's replica count", () => {
    expect(
      summarizeInstances(source({ instances: [instance("us-east-1", { status: "inactive" })] })),
    ).toEqual({ state: "inactive", running: 0, total: 1 });
    expect(summarizeInstances(source())).toBe(null);
  });
});

describe("instancesText", () => {
  test("counts running instances, or words the empty case by the desired state", () => {
    expect(instancesText({ state: "running", running: 1, total: 3 }, "running")).toBe(
      "1 of 3 running",
    );
    expect(instancesText(null, "running")).toBe("No instances");
    expect(instancesText(null, "stopped")).toBe("No instances running");
  });
});

describe("instanceStateOf", () => {
  test("only the crash-looping pod reads unhealthy", () => {
    expect(instanceStateOf({ status: "pending", lastExit: waiting("CrashLoopBackOff") })).toBe(
      "unhealthy",
    );
    expect(instanceStateOf({ status: "running", lastExit: null })).toBe("running");
  });

  test.each([
    "CrashLoopBackOff",
    "ImagePullBackOff",
    "ErrImagePull",
    "CreateContainerConfigError",
    "InvalidImageName",
  ])("%s on a pending instance reads unhealthy", (statusReason) => {
    expect(instanceStateOf({ status: "pending", lastExit: waiting(statusReason) })).toBe(
      "unhealthy",
    );
  });

  test("a fatal reason on a disabled instance stays disabled", () => {
    expect(instanceStateOf({ status: "inactive", lastExit: waiting("ImagePullBackOff") })).toBe(
      "inactive",
    );
  });

  test("a transient waiting reason does not read unhealthy", () => {
    expect(instanceStateOf({ status: "pending", lastExit: waiting("ContainerCreating") })).toBe(
      "starting",
    );
  });
});
