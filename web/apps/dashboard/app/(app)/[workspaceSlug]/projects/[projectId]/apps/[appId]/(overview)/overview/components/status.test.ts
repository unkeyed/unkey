import type { Deployment } from "@/lib/collections/deploy/deployments";
import { describe, expect, test } from "vitest";
import { deriveProductionStatus } from "./status";

type Instance = Deployment["instances"][number];

const instance = (overrides: Partial<Instance> = {}): Instance => ({
  id: "i-1",
  region: { id: "r-1", name: "us-east-1", platform: "aws" },
  status: "running",
  statusReason: null,
  lastExit: null,
  ...overrides,
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
        instances: [instance(), instance({ id: "i-2", statusReason: "CrashLoopBackOff" })],
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
          instance({ id: "i-2", status: "inactive", statusReason: "CrashLoopBackOff" }),
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
});
