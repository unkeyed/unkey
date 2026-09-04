import { beforeEach, describe, expect, it, vi } from "vitest";

type ListInput = {
  includeResolved: boolean;
  metric?:
    | "error_5xx"
    | "error_4xx"
    | "requests"
    | "requests_drop"
    | "egress_bytes"
    | "cpu_seconds"
    | "memory_utilization"
    | "oom_killed"
    | "crash_loop";
  appId?: string;
  environmentId?: string;
  startMs?: number;
  endMs?: number;
  cursor?: string;
  limit: number;
};

type Resolver = (opts: {
  ctx: { workspace: { id: string } };
  input: ListInput;
}) => Promise<{ alerts: unknown[]; nextCursor: string | undefined }>;

const state = vi.hoisted(() => ({
  resolver: null as Resolver | null,
  equalityValues: [] as unknown[],
  lessThanValues: [] as unknown[],
  rows: [] as Array<Array<{ firedAt: number; id: string; pk: number }>>,
  selectCalls: 0,
}));

vi.mock("@/lib/db", async () => {
  const actual = await vi.importActual<typeof import("@unkey/db")>("@unkey/db");
  const query = {
    innerJoin: () => query,
    where: () => query,
    orderBy: () => query,
    limit: async () => state.rows.shift() ?? [],
  };
  return {
    ...actual,
    eq: (_column: unknown, value: unknown) => {
      state.equalityValues.push(value);
      return actual.sql`true`;
    },
    lt: (_column: unknown, value: unknown) => {
      state.lessThanValues.push(value);
      return actual.sql`true`;
    },
    db: {
      select: () => {
        state.selectCalls += 1;
        return { from: () => query };
      },
    },
  };
});

vi.mock("../../trpc", () => ({
  workspaceProcedure: {
    input: () => ({
      query: (resolver: Resolver) => {
        state.resolver = resolver;
        return {};
      },
    }),
  },
}));

import "./list";

function listAlerts(input: Partial<ListInput> = {}) {
  if (!state.resolver) {
    throw new Error("listAlerts resolver was not registered");
  }
  return state.resolver({
    ctx: { workspace: { id: "ws_1" } },
    input: { includeResolved: false, limit: 50, ...input },
  });
}

describe("listAlerts", () => {
  beforeEach(() => {
    state.equalityValues = [];
    state.lessThanValues = [];
    state.rows = [];
    state.selectCalls = 0;
  });

  it("limits the workspace inbox to open alerts", async () => {
    await listAlerts();

    expect(state.equalityValues).toContain("open");
  });

  it("includes resolved alerts when the app view opts in", async () => {
    await listAlerts({ includeResolved: true });

    expect(state.equalityValues).not.toContain("open");
  });

  it("continues from an immutable cursor after the boundary alert resolves", async () => {
    state.rows = [
      [
        { firedAt: 2_000, id: "alert_boundary", pk: 20 },
        { firedAt: 1_000, id: "alert_next", pk: 10 },
      ],
    ];

    const firstPage = await listAlerts({ limit: 1 });

    expect(firstPage.nextCursor).toBeTypeOf("string");
    state.rows = [[{ firedAt: 1_000, id: "alert_next", pk: 10 }]];

    const secondPage = await listAlerts({ cursor: firstPage.nextCursor });

    expect(secondPage.alerts).toEqual([{ firedAt: 1_000, id: "alert_next" }]);
    expect(state.selectCalls).toBe(2);
    expect(state.equalityValues).toContain(2_000);
    expect(state.lessThanValues).toEqual([2_000, 20]);
  });

  it("rejects malformed cursors", async () => {
    await expect(listAlerts({ cursor: "not-a-cursor" })).rejects.toMatchObject({
      code: "BAD_REQUEST",
      message: "Invalid alert cursor",
    });
    expect(state.selectCalls).toBe(0);
  });
});
