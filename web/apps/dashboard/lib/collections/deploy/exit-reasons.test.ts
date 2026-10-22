import type { LastExit } from "@/lib/types/deploy";
import { describe, expect, test } from "vitest";
import { isFatalWaitingReason, summarizeDeploymentExit, summarizeExit } from "./exit-reasons";

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

describe("summarizeDeploymentExit", () => {
  const apiExit = {
    reason: "exited_with_error",
    message: "Exit code 1 usually means an unhandled error. Check your logs for the cause.",
    exitCode: 1,
    restartCount: 2,
    finishedAt: 1_000,
  };

  test("labels the reason and shows the API message", () => {
    expect(summarizeDeploymentExit(apiExit)).toEqual({
      severity: "warning",
      label: "Exited with error",
      exit: "exit=1",
      sentences: [apiExit.message],
      raw: "exit code 1",
    });
  });

  test("a crash loop is an error and does not repeat the exit code", () => {
    const summary = summarizeDeploymentExit({ ...apiExit, reason: "crash_loop" });
    expect(summary.severity).toBe("error");
    expect(summary.label).toBe("Crashing");
    expect(summary.exit).toBe(null);
  });

  test("a reason added later reads as a neutral exit", () => {
    expect(summarizeDeploymentExit({ ...apiExit, reason: "KEBAP" }).label).toBe("Exited");
  });

  test("no recorded exit has no exit code line", () => {
    const summary = summarizeDeploymentExit({ ...apiExit, reason: "crash_loop", exitCode: null });
    expect(summary.raw).toBe(null);
  });
});
