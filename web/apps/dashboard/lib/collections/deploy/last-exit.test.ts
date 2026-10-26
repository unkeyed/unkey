import type { LastExit } from "@/lib/types/deploy";
import { describe, expect, test } from "vitest";
import { describeLastExit, latestExitAcross } from "./last-exit";

const exit = (overrides: Partial<LastExit> = {}): LastExit => ({
  restartCount: 0,
  exitCode: null,
  signal: null,
  reason: null,
  finishedAt: null,
  statusReason: null,
  ...overrides,
});

describe("describeLastExit", () => {
  test("a fatal waiting reason wins over the last termination", () => {
    expect(
      describeLastExit(exit({ statusReason: "CrashLoopBackOff", reason: "Error", exitCode: 1 })),
    ).toEqual({ kind: "waiting", reason: "CrashLoopBackOff" });
    expect(describeLastExit(exit({ statusReason: "ImagePullBackOff" }))).toEqual({
      kind: "waiting",
      reason: "ImagePullBackOff",
    });
  });

  test("a transient waiting reason reads as the last exit", () => {
    expect(
      describeLastExit(
        exit({ statusReason: "ContainerCreating", reason: "OOMKilled", exitCode: 137 }),
      ),
    ).toEqual({ kind: "exit", reason: "OOMKilled", exit: "exit=137" });
  });

  test("a zero exit code is not shown", () => {
    expect(describeLastExit(exit({ reason: "Completed", exitCode: 0 }))).toEqual({
      kind: "exit",
      reason: "Completed",
      exit: null,
    });
  });
});

describe("latestExitAcross", () => {
  test("is null when no instance has exited", () => {
    expect(latestExitAcross([])).toBe(null);
    expect(latestExitAcross([{ lastExit: null }])).toBe(null);
  });

  test("picks the most recent exit, then the most restarts", () => {
    const old = exit({ reason: "Error", finishedAt: 1_000, restartCount: 9 });
    const recent = exit({ reason: "OOMKilled", finishedAt: 2_000, restartCount: 1 });
    expect(latestExitAcross([{ lastExit: old }, { lastExit: null }, { lastExit: recent }])).toBe(
      recent,
    );

    const fewer = exit({ statusReason: "CrashLoopBackOff", restartCount: 2 });
    const more = exit({ statusReason: "CrashLoopBackOff", restartCount: 5 });
    expect(latestExitAcross([{ lastExit: more }, { lastExit: fewer }])).toBe(more);
  });
});
