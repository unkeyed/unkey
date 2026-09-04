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
}));

vi.mock("@/lib/db", async () => {
  const actual = await vi.importActual<typeof import("@unkey/db")>("@unkey/db");
  const query = {
    innerJoin: () => query,
    where: () => query,
    orderBy: () => query,
    limit: async () => [],
  };
  return {
    ...actual,
    eq: (_column: unknown, value: unknown) => {
      state.equalityValues.push(value);
      return actual.sql`true`;
    },
    db: {
      select: () => ({ from: () => query }),
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

function listAlerts(includeResolved: boolean) {
  if (!state.resolver) {
    throw new Error("listAlerts resolver was not registered");
  }
  return state.resolver({
    ctx: { workspace: { id: "ws_1" } },
    input: { includeResolved, limit: 50 },
  });
}

describe("listAlerts", () => {
  beforeEach(() => {
    state.equalityValues = [];
  });

  it("limits the workspace inbox to open alerts", async () => {
    await listAlerts(false);

    expect(state.equalityValues).toContain("open");
  });

  it("includes resolved alerts when the app view opts in", async () => {
    await listAlerts(true);

    expect(state.equalityValues).not.toContain("open");
  });
});
