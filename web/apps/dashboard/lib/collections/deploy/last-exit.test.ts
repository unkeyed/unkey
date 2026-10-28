import type { LastExit } from "@/lib/types/deploy";
import type { ContainerStatus } from "@unkey/db/src/schema";
import { describe, expect, it, test } from "vitest";
import { exitSummary, lastExitOf, latestExit } from "./last-exit";

const exit = (overrides: Partial<LastExit> = {}): LastExit => ({
  restartCount: 0,
  exitCode: null,
  signal: null,
  reason: null,
  finishedAt: null,
  statusReason: null,
  ...overrides,
});

describe("lastExitOf", () => {
  const crashed = {
    restartCount: 3,
    lastTerminationState: { exitCode: 137, signal: 9, reason: "OOMKilled", finishedAt: 1_000 },
  };

  test("has no last exit for a container that is still starting", () => {
    expect(lastExitOf({ restartCount: 0, waiting: { reason: "ContainerCreating" } })).toBe(null);
    expect(lastExitOf({ restartCount: 0 })).toBe(null);
    expect(lastExitOf(null)).toBe(null);
  });

  test("keeps a fatal waiting reason before the first exit", () => {
    expect(lastExitOf({ restartCount: 0, waiting: { reason: "ImagePullBackOff" } })).toEqual(
      exit({ statusReason: "ImagePullBackOff" }),
    );
  });

  test("keeps a termination behind a waiting reason that is not fatal", () => {
    expect(lastExitOf({ ...crashed, waiting: { reason: "ContainerCreating" } })).toEqual(
      exit({
        restartCount: 3,
        exitCode: 137,
        signal: 9,
        reason: "OOMKilled",
        finishedAt: 1_000,
        statusReason: null,
      }),
    );
  });

  test("keeps the waiting reason next to a termination", () => {
    expect(lastExitOf({ ...crashed, waiting: { reason: "CrashLoopBackOff" } })).toEqual(
      exit({
        restartCount: 3,
        exitCode: 137,
        signal: 9,
        reason: "OOMKilled",
        finishedAt: 1_000,
        statusReason: "CrashLoopBackOff",
      }),
    );
  });
});

const recoveredOom = {
  containerStatus: {
    restartCount: 1,
    lastTerminationState: { exitCode: 137, signal: 9, reason: "OOMKilled", finishedAt: 2_000 },
  },
};

describe("latestExit", () => {
  const pickFromRows = (rows: { containerStatus: ContainerStatus | null }[]) =>
    latestExit(rows.map((row) => lastExitOf(row.containerStatus)));

  it("ranks a pod in a crash loop above a newer exit that recovered", () => {
    const crashLoop = {
      containerStatus: {
        restartCount: 4,
        lastTerminationState: { exitCode: 1, signal: 0, reason: "Error", finishedAt: 1_500 },
        waiting: { reason: "CrashLoopBackOff" },
      },
    };
    expect(pickFromRows([recoveredOom, crashLoop])?.statusReason).toBe("CrashLoopBackOff");
  });

  it("ranks a fatal wait with no termination above a terminated pod", () => {
    const imagePull = {
      containerStatus: { restartCount: 0, waiting: { reason: "ImagePullBackOff" } },
    };
    expect(pickFromRows([recoveredOom, imagePull])).toEqual({
      restartCount: 0,
      exitCode: null,
      signal: null,
      reason: null,
      finishedAt: null,
      statusReason: "ImagePullBackOff",
    });
  });

  it("picks the most recent exit when no pod is stuck", () => {
    const older = {
      containerStatus: {
        restartCount: 3,
        lastTerminationState: { exitCode: 1, signal: 0, reason: "Error", finishedAt: 1_000 },
      },
    };
    expect(pickFromRows([older, recoveredOom])?.reason).toBe("OOMKilled");
    expect(pickFromRows([{ containerStatus: null }])).toBe(null);
  });

  function crashLoop(finishedAt: number, restartCount: number) {
    return {
      containerStatus: {
        restartCount,
        lastTerminationState: { exitCode: 1, signal: 0, reason: "Error", finishedAt },
        waiting: { reason: "CrashLoopBackOff" },
      },
    };
  }

  it("picks the later exit between two fatal pods", () => {
    const early = crashLoop(1_000, 9);
    const late = crashLoop(3_000, 2);
    expect(pickFromRows([early, late])?.finishedAt).toBe(3_000);
    expect(pickFromRows([late, early])?.finishedAt).toBe(3_000);
  });

  it("breaks a finishedAt tie by the higher restart count", () => {
    const fewer = crashLoop(1_000, 2);
    const more = crashLoop(1_000, 5);
    expect(pickFromRows([fewer, more])?.restartCount).toBe(5);
    expect(pickFromRows([more, fewer])?.restartCount).toBe(5);
  });

  it("picks the fatal pod in either order", () => {
    const stuck = crashLoop(1_500, 4);
    expect(pickFromRows([stuck, recoveredOom])?.statusReason).toBe("CrashLoopBackOff");
    expect(pickFromRows([recoveredOom, stuck])?.statusReason).toBe("CrashLoopBackOff");
  });
});

describe("exitSummary", () => {
  test("names the cause, the exit code and the restarts", () => {
    expect(
      exitSummary(
        exit({ restartCount: 7, reason: "Error", exitCode: 1, statusReason: "CrashLoopBackOff" }),
      ),
    ).toBe("Exited with error · exit=1 · 7 restarts");
    expect(exitSummary(exit({ restartCount: 1, reason: "OOMKilled", exitCode: 137 }))).toBe(
      "Out of memory · exit=137 · 1 restart",
    );
  });

  test("reads a failed exit without a reason as an error and a clean one as neutral", () => {
    expect(exitSummary(exit({ reason: "", exitCode: 1, restartCount: 2 }))).toBe(
      "Exited with error · exit=1 · 2 restarts",
    );
    expect(exitSummary(exit({ exitCode: 0 }))).toBe("Exited");
  });
});
