import { describe, expect, it } from "vitest";
import {
  type InstanceSnapshot,
  crashExplanation,
  isDeploymentSettled,
  lastExitDetail,
  summarizeInstances,
} from "./instance-state";

const snapshot = (
  statuses: InstanceSnapshot["instances"][number]["status"][],
  lastExit: InstanceSnapshot["lastExit"] = null,
): InstanceSnapshot => ({
  instances: statuses.map((status) => ({ status })),
  desiredInstanceCount: 1,
  lastExit,
});

const crashLoop = {
  restartCount: 0,
  exitCode: null,
  reason: null,
  statusReason: "CrashLoopBackOff",
};

describe("summarizeInstances", () => {
  it("uses the deployment page words for each state", () => {
    expect(summarizeInstances(snapshot(["pending"]))?.text).toBe("0 of 1 starting");
    expect(summarizeInstances(snapshot(["running"]))?.text).toBe("1 of 1 running");
    expect(summarizeInstances(snapshot(["failed"]))?.text).toBe(
      "Unhealthy · Instance stopped unexpectedly.",
    );
  });

  it("counts a crash loop as unhealthy even while the instance reads pending", () => {
    const summary = summarizeInstances(snapshot(["pending"], crashLoop));
    expect(summary?.state).toBe("unhealthy");
    expect(summary?.tone).toBe("error");
    expect(summary?.lastExit).toBe("Last exit: Keeps crashing · 0 restarts");
  });
});

describe("lastExitDetail", () => {
  it("keeps raw reasons as secondary detail", () => {
    expect(
      lastExitDetail({ restartCount: 3, exitCode: 137, reason: "OOMKilled", statusReason: null }),
    ).toBe("Last exit: Out of memory · Exit code 137 · 3 restarts");
    expect(lastExitDetail(null)).toBeNull();
  });
});

describe("crashExplanation", () => {
  it("only names an exit code when the data has one", () => {
    expect(crashExplanation(crashLoop, false)).toBe(
      "Your app stopped right after it started and printed nothing.",
    );
    expect(
      crashExplanation(
        { restartCount: 1, exitCode: 0, reason: "Completed", statusReason: null },
        true,
      ),
    ).toBe("Your app exited right after it started (exit code 0).");
  });
});

describe("isDeploymentSettled", () => {
  it("keeps polling until a ready deployment has a running or unhealthy instance", () => {
    expect(isDeploymentSettled("building", null)).toBe(false);
    expect(isDeploymentSettled("failed", null)).toBe(true);
    expect(isDeploymentSettled("ready", summarizeInstances(snapshot(["pending"])))).toBe(false);
    expect(isDeploymentSettled("ready", summarizeInstances(snapshot(["running"])))).toBe(true);
    expect(isDeploymentSettled("ready", summarizeInstances(snapshot(["pending"], crashLoop)))).toBe(
      true,
    );
  });
});
