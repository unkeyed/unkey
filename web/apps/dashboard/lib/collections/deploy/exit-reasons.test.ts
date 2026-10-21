import type { LastExit } from "@/lib/types/deploy";
import { describe, expect, test } from "vitest";
import { isFatalWaitingReason, reasonCopy, summarizeExit } from "./exit-reasons";

describe("reasonCopy", () => {
  test("gives plain copy for known reasons", () => {
    expect(reasonCopy("CrashLoopBackOff")?.label).toBe("Crashing");
    expect(reasonCopy("ErrImagePull")?.label).toBe("Can't pull image");
  });

  test("is null for unknown, inherited or missing reasons", () => {
    expect(reasonCopy("RunContainerError")).toBe(null);
    expect(reasonCopy("toString")).toBe(null);
    expect(reasonCopy(null)).toBe(null);
  });
});

describe("isFatalWaitingReason", () => {
  test("is true only for waiting reasons that stop an instance", () => {
    expect(isFatalWaitingReason("CrashLoopBackOff")).toBe(true);
    expect(isFatalWaitingReason("CreateContainerConfigError")).toBe(true);
    expect(isFatalWaitingReason("OOMKilled")).toBe(false);
    expect(isFatalWaitingReason("ContainerCreating")).toBe(false);
    expect(isFatalWaitingReason(null)).toBe(false);
  });
});

const exit = (overrides: Partial<LastExit> = {}): LastExit => ({
  restartCount: 0,
  exitCode: null,
  signal: null,
  reason: null,
  finishedAt: null,
  statusReason: null,
  ...overrides,
});

describe("summarizeExit", () => {
  test("a fatal waiting reason is an error and wins over the last termination", () => {
    const summary = summarizeExit(
      exit({ statusReason: "CrashLoopBackOff", reason: "Error", exitCode: 1 }),
    );
    expect(summary).toEqual({
      severity: "error",
      label: "Crashing",
      exit: null,
      sentences: ["Your app exits right after it starts. Check your logs."],
      raw: "CrashLoopBackOff · Error · exit code 1",
    });
  });

  test("a past termination is a warning and does not repeat the exit code", () => {
    const summary = summarizeExit(exit({ reason: "OOMKilled", exitCode: 137 }));
    expect(summary.severity).toBe("warning");
    expect(summary.exit).toBe("exit=137");
    expect(summary.sentences).toEqual([
      "Your app hit its memory limit. Fix a leak or raise the limit.",
    ]);
  });

  test("a generic exit puts the exit-code hint before the reason's fix", () => {
    expect(summarizeExit(exit({ reason: "Error", exitCode: 127 })).sentences).toEqual([
      "Start command not found.",
      "Check your logs for the cause.",
    ]);
  });

  test("an unknown reason keeps its raw label and still explains the exit code", () => {
    const summary = summarizeExit(exit({ reason: "RunContainerError", exitCode: 42, signal: 9 }));
    expect(summary.label).toBe("RunContainerError");
    expect(summary.sentences).toEqual(["Force-killed, usually out of memory."]);
  });

  test("no reason reads as a neutral exit", () => {
    expect(summarizeExit(exit({ exitCode: 0 }))).toMatchObject({
      label: "Exited",
      exit: null,
      sentences: [],
    });
  });
});
