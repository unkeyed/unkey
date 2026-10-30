import { describe, expect, test } from "vitest";
import { findReplacedDeployment, previousRollbackTarget, rollbackCandidates } from "./rollback";

type Row = {
  id: string;
  environmentId: string;
  status: "ready" | "stopped" | "failed";
  desiredState: "running" | "stopped";
  createdAt: number;
};

const row = (id: string, createdAt: number, overrides: Partial<Row> = {}): Row => ({
  id,
  environmentId: "env_prod",
  status: "ready",
  desiredState: "running",
  createdAt,
  ...overrides,
});

const current = row("d_current", 100);

describe("rollbackCandidates", () => {
  test("keeps running ready siblings, newest first", () => {
    const result = rollbackCandidates([row("d_a", 10), row("d_b", 50), current], current);
    expect(result.map((d) => d.id)).toEqual(["d_b", "d_a"]);
  });

  test("drops deployments that are scaling down, stopped or failed", () => {
    const result = rollbackCandidates(
      [
        row("d_draining", 90, { desiredState: "stopped" }),
        row("d_stopped", 80, { status: "stopped", desiredState: "stopped" }),
        row("d_failed", 70, { status: "failed" }),
      ],
      current,
    );
    expect(result).toEqual([]);
  });

  test("drops other environments and the current deployment", () => {
    const result = rollbackCandidates(
      [row("d_preview", 90, { environmentId: "env_preview" }), current],
      current,
    );
    expect(result).toEqual([]);
  });
});

describe("previousRollbackTarget", () => {
  test("picks the newest running deployment older than the current one", () => {
    const result = previousRollbackTarget(
      [row("d_newer", 150), row("d_old", 10), row("d_prev", 60)],
      current,
    );
    expect(result?.id).toBe("d_prev");
  });

  test("returns undefined once the previous deployment is scaling down", () => {
    const result = previousRollbackTarget(
      [row("d_prev", 60, { desiredState: "stopped" })],
      current,
    );
    expect(result).toBeUndefined();
  });
});

describe("findReplacedDeployment", () => {
  const newest = row("d_newest", 300);
  const replaced = row("d_replaced", 200);

  test("uses the deployment the rollback event names", () => {
    expect(findReplacedDeployment([newest, replaced, current], current, "d_replaced")?.id).toBe(
      "d_replaced",
    );
  });

  test("returns nothing when the named deployment is not loaded", () => {
    expect(findReplacedDeployment([newest, current], current, "d_replaced")).toBeUndefined();
  });

  test("infers the newest later ready deployment without an event", () => {
    expect(findReplacedDeployment([replaced, newest, current], current, null)?.id).toBe("d_newest");
  });
});
