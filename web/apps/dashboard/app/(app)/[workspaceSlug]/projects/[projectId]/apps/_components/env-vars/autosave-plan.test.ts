import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { describe, expect, it } from "vitest";
import {
  applyChange,
  hydrateTable,
  planChanges,
  resolveStatus,
  snapshotFromServer,
  visibleRowErrors,
} from "./autosave-plan";

function envVar(environmentId: string, key: string, value: string, type: EnvVar["type"]): EnvVar {
  return {
    id: `${environmentId}:${key}`,
    key,
    value,
    type,
    description: null,
    createdAt: 0,
    environmentId,
    projectId: "proj",
    appId: "app",
  };
}

const foo = { key: "FOO", value: "1", kind: "recoverable" } as const;
const bar = { key: "BAR", value: "2", kind: "recoverable" } as const;
const environmentIds = ["prod", "preview"];

describe("planChanges", () => {
  it("writes nothing when the server already matches", () => {
    const server = snapshotFromServer([envVar("prod", "FOO", "1", "recoverable")]);
    expect(
      planChanges({
        server,
        environmentIds,
        targetIds: ["prod"],
        variables: [foo],
      }),
    ).toEqual([]);
  });

  it("sets new and changed keys and removes dropped keys per environment", () => {
    const server = snapshotFromServer([
      envVar("prod", "FOO", "0", "recoverable"),
      envVar("preview", "OLD", "x", "recoverable"),
    ]);
    expect(
      planChanges({
        server,
        environmentIds,
        targetIds: environmentIds,
        variables: [foo, bar],
      }),
    ).toEqual([
      { environmentId: "prod", set: [foo, bar], remove: [] },
      { environmentId: "preview", set: [foo, bar], remove: ["OLD"] },
    ]);
  });

  it("rewrites every key a failed change touched", () => {
    const change = { environmentId: "prod", set: [foo], remove: ["OLD"] };
    const after = applyChange(new Map([["OLD", "x"]]), change, false);
    expect(
      planChanges({
        server: new Map([["prod", after]]),
        environmentIds: ["prod"],
        targetIds: ["prod"],
        variables: [foo],
      }),
    ).toEqual([{ environmentId: "prod", set: [foo], remove: ["OLD"] }]);
  });

  it("writes nothing after a change succeeds", () => {
    const change = { environmentId: "prod", set: [foo], remove: ["OLD"] };
    const after = applyChange(new Map([["OLD", "x"]]), change, true);
    expect(
      planChanges({
        server: new Map([["prod", after]]),
        environmentIds: ["prod"],
        targetIds: ["prod"],
        variables: [foo],
      }),
    ).toEqual([]);
  });
});

describe("hydrateTable", () => {
  it("loads rows that plan no writes and picks the single environment in use", () => {
    const table = hydrateTable(
      [envVar("prod", "FOO", "1", "recoverable"), envVar("prod", "BAR", "2", "recoverable")],
      environmentIds,
    );
    expect(
      table.kind === "table" ? table.rows.map(({ key, value }) => ({ key, value })) : table,
    ).toEqual([
      { key: "FOO", value: "1" },
      { key: "BAR", value: "2" },
    ]);
    expect(table.kind === "table" ? table.environmentId : table).toBe("prod");
  });

  it("uses all environments when every key is in each one", () => {
    const table = hydrateTable(
      [envVar("prod", "FOO", "1", "recoverable"), envVar("preview", "FOO", "1", "recoverable")],
      environmentIds,
    );
    expect(table.kind === "table" ? table.environmentId : table).toBe("all");
  });

  it.each([
    ["a writeonly value", [envVar("prod", "SECRET", "", "writeonly")]],
    [
      "values that differ per environment",
      [envVar("prod", "FOO", "a", "recoverable"), envVar("preview", "FOO", "b", "recoverable")],
    ],
    [
      "keys split unevenly across environments",
      [
        envVar("prod", "FOO", "1", "recoverable"),
        envVar("preview", "FOO", "1", "recoverable"),
        envVar("prod", "BAR", "2", "recoverable"),
      ],
    ],
    ["a value the table would trim", [envVar("prod", "FOO", " padded ", "recoverable")]],
    ["a key the table rejects", [envVar("prod", "1BAD", "x", "recoverable")]],
  ])("refuses %s instead of rewriting it", (_, vars) => {
    expect(hydrateTable(vars, environmentIds)).toEqual({ kind: "unrepresentable" });
  });
});

describe("resolveStatus", () => {
  const editing = {
    phase: "editing",
    failed: false,
    pending: false,
    invalid: 0,
    count: 2,
  } as const;

  it("reports the load phase before anything else", () => {
    expect(resolveStatus({ ...editing, phase: "load-failed", failed: true })).toEqual({
      kind: "load-failed",
    });
  });

  it("puts a failed save ahead of invalid rows and pending writes", () => {
    expect(resolveStatus({ ...editing, failed: true, invalid: 1, pending: true })).toEqual({
      kind: "failed",
    });
  });

  it("puts invalid rows ahead of pending writes", () => {
    expect(resolveStatus({ ...editing, invalid: 1, pending: true })).toEqual({
      kind: "invalid",
      invalid: 1,
    });
  });

  it("counts saved variables when nothing is pending", () => {
    expect(resolveStatus(editing)).toEqual({ kind: "saved", count: 2 });
  });
});

describe("visibleRowErrors", () => {
  const badKey = { id: "r1", key: "1BAD", value: "x", sensitive: false };
  const halfRow = { id: "r2", key: "HALF", value: "", sensitive: false };
  const errors = new Map([
    ["r1", { key: "bad key" }],
    ["r2", { value: "required" }],
  ]);

  it("hides required errors while the table has focus", () => {
    const rows = [badKey, halfRow];
    expect([
      ...visibleRowErrors({ errors, rows, settledRows: rows, tableFocused: true }).keys(),
    ]).toEqual(["r1"]);
  });

  it("shows every error once focus leaves the table", () => {
    const rows = [badKey, halfRow];
    expect([
      ...visibleRowErrors({ errors, rows, settledRows: rows, tableFocused: false }).keys(),
    ]).toEqual(["r1", "r2"]);
  });

  it("hides errors on a row edited since the last settle", () => {
    const edited = { ...badKey, key: "1BADD" };
    expect([
      ...visibleRowErrors({
        errors,
        rows: [edited, halfRow],
        settledRows: [badKey, halfRow],
        tableFocused: false,
      }).keys(),
    ]).toEqual(["r2"]);
  });
});
