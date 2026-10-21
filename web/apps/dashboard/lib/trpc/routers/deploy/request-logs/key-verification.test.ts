import { initTRPC } from "@trpc/server";
import { QueryError } from "@unkey/clickhouse/src/client/error";
import type { RequestDetailsRequest } from "@unkey/clickhouse/src/frontline";
import { Err, Ok } from "@unkey/error";
import { newId } from "@unkey/id";
import { beforeEach, expect, it, vi } from "vitest";
import { queryRequestKeyVerification } from "./key-verification";

const mocks = vi.hoisted(() => ({
  verification: vi.fn(),
  key: vi.fn(),
  api: vi.fn(),
}));

vi.mock("@/lib/clickhouse", () => ({
  clickhouse: { frontline: { keyVerification: mocks.verification } },
}));
vi.mock("@/lib/db", () => ({
  db: { query: { keys: { findFirst: mocks.key }, apis: { findFirst: mocks.api } } },
}));
vi.mock("@/lib/trpc/trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.context<{ workspace: { id: string } }>().create();
  return {
    workspaceProcedure: t.procedure,
    ratelimit: { read: {} },
    withRatelimit: () => t.middleware(({ next }) => next()),
  };
});

const workspaceId = newId("workspace");
const request = { requestId: newId("request"), startTime: 1, endTime: 2 };
const verification = {
  key_id: newId("key"),
  key_space_id: newId("keyAuth"),
  outcome: "RATE_LIMITED",
};
const api = { id: newId("api"), projectId: newId("project") };
const t = initTRPC.context<{ workspace: { id: string } }>().create();
const caller = t.router({ verification: queryRequestKeyVerification }).createCaller({
  workspace: { id: workspaceId },
});

beforeEach(() => {
  vi.resetAllMocks();
  mocks.verification.mockImplementation(async (params: RequestDetailsRequest) =>
    Ok(params.workspaceId === workspaceId ? [verification] : []),
  );
  mocks.key.mockResolvedValue({ name: "Orders backend" });
  mocks.api.mockResolvedValue(api);
});

it("attributes the request in the session workspace, not a caller-supplied workspace", async () => {
  const input = { ...request, workspaceId: newId("workspace") };
  const result = await caller.verification(input);
  expect(result).toEqual({
    verification: {
      ...verification,
      key: { name: "Orders backend", apiId: api.id, projectId: api.projectId },
    },
  });
});

it.each(["key", "api"] as const)(
  "keeps the historical key ID and outcome when the %s no longer exists",
  async (missing) => {
    mocks[missing].mockResolvedValue(undefined);
    expect(await caller.verification(request)).toEqual({
      verification: { ...verification, key: null },
    });
  },
);

it("returns no attribution when no matching verification was recorded", async () => {
  mocks.verification.mockResolvedValue(Ok([]));
  expect(await caller.verification(request)).toEqual({ verification: null });
});

it("preserves an outcome outside the dashboard's known labels", async () => {
  mocks.verification.mockResolvedValue(Ok([{ ...verification, outcome: "WORKSPACE_DISABLED" }]));
  expect((await caller.verification(request)).verification?.outcome).toBe("WORKSPACE_DISABLED");
});

it("does not report a lookup failure as an absent verification", async () => {
  mocks.verification.mockResolvedValue(Err(new QueryError("unavailable", { query: "" })));
  await expect(caller.verification(request)).rejects.toMatchObject({
    code: "INTERNAL_SERVER_ERROR",
  });
});
