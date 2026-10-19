import { describe, expect, test } from "vitest";
import { resolvePageStatus } from "./deployment-health";

describe("resolvePageStatus", () => {
  test("a ready deployment with unhealthy instances reads Unhealthy", () => {
    const status = resolvePageStatus("ready", { state: "unhealthy", running: 0, total: 1 });
    expect(status.label).toBe("Unhealthy");
    expect(status.dotClass).toBe("bg-error-9");
  });

  test("a ready deployment with running instances reads Ready", () => {
    expect(resolvePageStatus("ready", { state: "running", running: 1, total: 1 }).label).toBe(
      "Ready",
    );
  });

  test("building statuses are in flight", () => {
    expect(resolvePageStatus("building", null).inFlight).toBe(true);
  });
});
