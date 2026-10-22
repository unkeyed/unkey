import type { Deployment } from "@/lib/collections/deploy/deployments";
import { describe, expect, test } from "vitest";
import { deriveProductionStatus } from "./status";

type Instance = Deployment["instances"][number];

const instance = (overrides: Partial<Instance> = {}): Instance => ({
  id: "i-1",
  region: { id: "r-1", name: "us-east-1", platform: "aws" },
  status: "running",
  lastExit: null,
  ...overrides,
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

describe("deriveProductionStatus", () => {
  test("a ready deployment with healthy instances is live", () => {
    expect(
      deriveProductionStatus({
        status: "ready",
        desiredState: "running",
        targetRegions: [],
        instances: [instance()],
      }),
    ).toBe("live");
  });

  test("one crash-looping pod makes the deployment crash", () => {
    expect(
      deriveProductionStatus({
        status: "ready",
        desiredState: "running",
        targetRegions: [],
        instances: [instance(), instance({ id: "i-2", lastExit: waiting("CrashLoopBackOff") })],
      }),
    ).toBe("crashing");
  });

  test("a crash loop on a disabled pod does not crash the deployment", () => {
    expect(
      deriveProductionStatus({
        status: "ready",
        desiredState: "running",
        targetRegions: [],
        instances: [
          instance(),
          instance({ id: "i-2", status: "inactive", lastExit: waiting("CrashLoopBackOff") }),
        ],
      }),
    ).toBe("live");
  });

  test("a failed deployment reads failed before instance health", () => {
    expect(
      deriveProductionStatus({
        status: "failed",
        desiredState: "running",
        targetRegions: [],
        instances: [instance({ status: "failed" })],
      }),
    ).toBe("failed");
  });

  test("a pod stuck pulling its image makes the deployment crash", () => {
    expect(
      deriveProductionStatus({
        status: "ready",
        desiredState: "running",
        targetRegions: [],
        instances: [instance({ lastExit: waiting("ImagePullBackOff") })],
      }),
    ).toBe("crashing");
  });

  test("a pod that recovered from an exit stays live", () => {
    expect(
      deriveProductionStatus({
        status: "ready",
        desiredState: "running",
        targetRegions: [],
        instances: [
          instance({
            lastExit: { ...waiting("CrashLoopBackOff"), statusReason: null, reason: "OOMKilled" },
          }),
        ],
      }),
    ).toBe("live");
  });
});
