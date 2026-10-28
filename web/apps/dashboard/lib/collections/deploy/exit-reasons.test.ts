import type { LastExit } from "@/lib/types/deploy";
import { describe, expect, test } from "vitest";
import { summarizeCrashLoop, summarizeExit } from "./exit-reasons";

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
  test("a crash loop names the last termination's cause and exit code", () => {
    expect(
      summarizeExit(exit({ statusReason: "CrashLoopBackOff", reason: "OOMKilled", exitCode: 137 })),
    ).toMatchObject({ severity: "error", label: "Out of memory", exit: "exit=137" });
    const error = summarizeExit(
      exit({ statusReason: "CrashLoopBackOff", reason: "Error", exitCode: 1 }),
    );
    expect(error).toMatchObject({ severity: "error", label: "Exited with error", exit: "exit=1" });
    expect(error.sentences).toContain("Unhandled error or failed startup check.");
    expect(error.raw).toBe("CrashLoopBackOff · Error · exit code 1");
  });

  test("a crash loop without a known termination reads as crashing with its exit code", () => {
    expect(summarizeExit(exit({ statusReason: "CrashLoopBackOff", exitCode: 1 }))).toMatchObject({
      severity: "error",
      label: "Crashing",
      exit: "exit=1",
      sentences: [
        "Unhandled error or failed startup check.",
        "Your app exits right after it starts. Check your logs.",
      ],
    });
    expect(summarizeExit(exit({ statusReason: "CrashLoopBackOff" }))).toMatchObject({
      label: "Crashing",
      exit: null,
    });
  });

  test("a crash loop only names causes the API also names", () => {
    expect(
      summarizeExit(
        exit({ statusReason: "CrashLoopBackOff", reason: "ContainerStatusUnknown", exitCode: 137 }),
      ),
    ).toMatchObject({ severity: "error", label: "Crashing", exit: "exit=137" });
    expect(
      summarizeExit(exit({ statusReason: "CrashLoopBackOff", reason: "Completed", exitCode: 0 })),
    ).toMatchObject({ severity: "error", label: "Exited cleanly", exit: null });
  });

  test("a stuck image or config error never shows an exit code", () => {
    for (const statusReason of ["ErrImagePull", "CreateContainerConfigError"]) {
      expect(summarizeExit(exit({ statusReason, reason: "Error", exitCode: 2 }))).toMatchObject({
        severity: "error",
        exit: null,
        sentences: [expect.any(String)],
      });
    }
  });

  test("an exit code above 128 with no signal names the signal", () => {
    expect(summarizeExit(exit({ reason: "Error", exitCode: 134 })).sentences).toEqual([
      "Killed by signal 6.",
      "Check your logs for the cause.",
    ]);
  });

  test("other fatal waiting reasons win over the last termination", () => {
    expect(
      summarizeExit(exit({ statusReason: "ImagePullBackOff", reason: "Error", exitCode: 1 })),
    ).toMatchObject({ severity: "error", label: "Can't pull image", exit: null });
  });

  test("a past termination is a warning and does not repeat the exit code", () => {
    const summary = summarizeExit(exit({ reason: "OOMKilled", exitCode: 137 }));
    expect(summary).toMatchObject({
      severity: "warning",
      label: "Out of memory",
      exit: "exit=137",
    });
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

  test("exit 137 points at a kill, not at memory", () => {
    expect(summarizeExit(exit({ reason: "Error", exitCode: 137 })).sentences[0]).toBe(
      "Killed with SIGKILL, often because it did not stop in time.",
    );
  });

  test("an exit code with no hint falls back to the signal or the app's docs", () => {
    expect(summarizeExit(exit({ reason: "Error", exitCode: 128 })).sentences[0]).toBe(
      "See your app's docs for this exit code.",
    );
    expect(summarizeExit(exit({ reason: "Error", exitCode: 42, signal: 9 })).sentences[0]).toBe(
      "Killed by signal 9.",
    );
  });

  test("an unknown reason keeps its raw label and still explains the exit code", () => {
    const summary = summarizeExit(exit({ reason: "RunContainerError", exitCode: 2 }));
    expect(summary).toMatchObject({ label: "RunContainerError", exit: "exit=2" });
    expect(summary.sentences).toEqual(["Crashed, usually a panic or uncaught exception."]);
  });

  test("a failed exit with no reason gets the error label and fix", () => {
    const { raw: _bare, ...bare } = summarizeExit(exit({ exitCode: 1 }));
    const { raw: _named, ...named } = summarizeExit(exit({ reason: "Error", exitCode: 1 }));
    expect(bare).toEqual(named);
    expect(summarizeExit(exit({ exitCode: 1 }))).toMatchObject({
      severity: "warning",
      label: "Exited with error",
      exit: "exit=1",
    });
  });

  test("a clean exit with no reason reads as a neutral exit", () => {
    expect(summarizeExit(exit({ exitCode: 0 }))).toMatchObject({
      label: "Exited",
      exit: null,
      sentences: [],
    });
    expect(summarizeExit(exit()).label).toBe("Exited");
  });
});

const crashLoop = {
  reason: "out_of_memory",
  message:
    "Your app used more memory than its limit and keeps stopping, so Unkey waits longer before each restart. Check its memory usage, or raise the limit.",
  exitCode: 137,
};

describe("summarizeCrashLoop", () => {
  test("labels the cause and shows the API message", () => {
    expect(summarizeCrashLoop(crashLoop)).toEqual({
      severity: "error",
      label: "Out of memory",
      exit: "exit=137",
      sentences: [crashLoop.message],
      raw: null,
    });
  });

  test("a crash loop with no recorded exit reads as crashing", () => {
    const summary = summarizeCrashLoop({ ...crashLoop, reason: null, exitCode: null });
    expect(summary.label).toBe("Crashing");
    expect(summary.exit).toBe(null);
  });

  test("a reason added later reads as crashing", () => {
    expect(summarizeCrashLoop({ ...crashLoop, reason: "KEBAP" }).label).toBe("Crashing");
  });
});
