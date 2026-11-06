import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import type { CallerEnvironment } from "./callers";
import type { ConnectionOp } from "./plan";
import { failureDescription, needsRedeploy, redeployEnvironmentIds, runInOrder } from "./run";

const production: CallerEnvironment = { id: newId("environment"), kind: "production" };
const preview: CallerEnvironment = { id: newId("environment"), kind: "preview" };

const deleteIn = (environment: CallerEnvironment): ConnectionOp => ({
  type: "delete",
  environment,
  id: newId("connection"),
});

describe("runInOrder", () => {
  it("stops at the first failing op and reports what applied", async () => {
    const first = deleteIn(production);
    const second = deleteIn(preview);
    const error = new Error("boom");
    const calls: ConnectionOp[] = [];
    const result = await runInOrder([first, second], async (op) => {
      calls.push(op);
      if (op === second) {
        throw error;
      }
    });
    expect(result).toEqual({ applied: [first], failure: { op: second, error } });
    expect(calls).toEqual([first, second]);
  });

  it("applies every op when none fails", async () => {
    const ops = [deleteIn(preview), deleteIn(production)];
    expect(await runInOrder(ops, async () => {})).toEqual({ applied: ops, failure: null });
  });
});

describe("needsRedeploy", () => {
  it("is true only when a production op applied", () => {
    expect(needsRedeploy([deleteIn(production)])).toBe(true);
    expect(needsRedeploy([deleteIn(preview)])).toBe(false);
    expect(needsRedeploy([])).toBe(false);
  });
});

describe("redeployEnvironmentIds", () => {
  it("names each environment whose applied ops need a redeploy once", () => {
    const ops = [deleteIn(production), deleteIn(preview), deleteIn(production)];
    expect(redeployEnvironmentIds(ops)).toEqual([production.id]);
    expect(redeployEnvironmentIds([deleteIn(preview)])).toEqual([]);
  });
});

describe("failureDescription", () => {
  it("names what saved and what failed", () => {
    expect(failureDescription([deleteIn(production)], deleteIn(preview), "saved", "boom")).toBe(
      "Production saved. Preview failed: boom",
    );
    expect(failureDescription([], deleteIn(production), "saved", "boom")).toBe("boom");
  });
});
