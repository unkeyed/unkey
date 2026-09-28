import { deriveProductionStatus } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/[appId]/(overview)/overview/components/status";
import type { Deployment } from "@/lib/collections/deploy/deployments";
import type { ContainerStatus } from "@unkey/db/src/schema";
import { describe, expect, it } from "vitest";
import { computeLastExit } from "./deployment-query-helpers";

const termination = {
  reason: "OOMKilled",
  exitCode: 137,
  signal: 9,
  finishedAt: 1_000,
};

describe("deployment diagnostics", () => {
  it.each([false, true])(
    "keeps active Kubernetes errors ahead of history (reverse=%s)",
    (reverse) => {
      const rows: { containerStatus: ContainerStatus | null }[] = [
        { containerStatus: { restartCount: 8, lastTerminationState: termination } },
        {
          containerStatus: {
            restartCount: 0,
            waiting: { reason: "ErrImagePull", message: "registry denied access" },
          },
        },
        { containerStatus: null },
      ];

      expect(computeLastExit(reverse ? rows.toReversed() : rows)).toEqual({
        restartCount: 0,
        exitCode: null,
        signal: null,
        reason: null,
        finishedAt: null,
        statusReason: "ErrImagePull",
        statusMessage: "registry denied access",
      });
    },
  );

  it("stops showing Crashing after waiting clears without erasing the last exit", () => {
    const runningInstance: Deployment["instances"][number] = {
      id: "i_restarted",
      region: { id: "r_test", name: "us-east-1", platform: "kubernetes" },
      flagCode: "us",
      status: "running",
    };
    const before: ContainerStatus = {
      restartCount: 3,
      lastTerminationState: termination,
      waiting: { reason: "CrashLoopBackOff", message: "back-off restarting failed container" },
    };
    const after: ContainerStatus = { restartCount: 3, lastTerminationState: termination };
    const lastExit = computeLastExit([{ containerStatus: after }]);

    expect(
      deriveProductionStatus({
        status: "ready",
        instances: [runningInstance],
        lastExit: computeLastExit([{ containerStatus: before }]),
      }),
    ).toBe("crashing");
    expect(lastExit).toEqual({
      restartCount: 3,
      ...termination,
      statusReason: null,
      statusMessage: null,
    });
    expect(
      deriveProductionStatus({ status: "ready", instances: [runningInstance], lastExit }),
    ).toBe("live");
    expect(
      deriveProductionStatus({
        status: "ready",
        instances: [{ ...runningInstance, status: "failed" }],
        lastExit,
      }),
    ).toBe("crashing");
  });
});
