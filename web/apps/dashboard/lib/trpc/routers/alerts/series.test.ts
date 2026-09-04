import { beforeEach, describe, expect, it, vi } from "vitest";

type SeriesInput = {
  appId: string;
  environmentId: string;
  metric:
    | "error_5xx"
    | "error_4xx"
    | "requests"
    | "egress_bytes"
    | "cpu_seconds"
    | "memory_utilization"
    | "health";
  resolution: "5m" | "1h";
  startMs: number;
  endMs: number;
};

type Resolver = (opts: {
  ctx: { workspace: { id: string } };
  input: SeriesInput;
}) => Promise<unknown>;

const state = vi.hoisted(() => ({
  app: undefined as { createdAt: number } | undefined,
  resolver: null as Resolver | null,
  seriesArgs: undefined as Record<string, unknown> | undefined,
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      apps: {
        findFirst: () => Promise.resolve(state.app),
      },
    },
  },
}));

vi.mock("@/lib/clickhouse", () => ({
  clickhouse: {
    alerts: {
      series: (args: Record<string, unknown>) => {
        state.seriesArgs = args;
        return Promise.resolve({ val: [], err: undefined });
      },
    },
  },
}));

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

import "./series";

function callSeries() {
  if (!state.resolver) {
    throw new Error("getAlertSeries resolver was not registered");
  }
  return state.resolver({
    ctx: { workspace: { id: "ws_1" } },
    input: {
      appId: "app_1",
      environmentId: "env_1",
      metric: "requests",
      resolution: "5m",
      startMs: Date.UTC(2026, 8, 4, 10),
      endMs: Date.UTC(2026, 8, 4, 11),
    },
  });
}

describe("getAlertSeries", () => {
  beforeEach(() => {
    vi.setSystemTime(Date.UTC(2026, 8, 4, 12));
    state.app = { createdAt: Date.UTC(2026, 8, 1) };
    state.seriesArgs = undefined;
  });

  it("passes the workspace app creation time to the ClickHouse series", async () => {
    await callSeries();

    expect(state.seriesArgs).toMatchObject({
      workspaceId: "ws_1",
      appId: "app_1",
      appCreatedAtMs: Date.UTC(2026, 8, 1),
      environmentId: "env_1",
    });
  });

  it("rejects an app outside the workspace", async () => {
    state.app = undefined;

    await expect(callSeries()).rejects.toMatchObject({ code: "NOT_FOUND" });
    expect(state.seriesArgs).toBeUndefined();
  });
});
