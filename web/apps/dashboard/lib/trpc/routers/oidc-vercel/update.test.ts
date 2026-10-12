import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { updateOIDCVercelIntegration } from "./update";

const insertedRows = vi.hoisted(() => vi.fn());
const deletedTables = vi.hoisted(() => vi.fn());

vi.mock("@/lib/db", async () => {
  const { and, eq, schema } = await import("@unkey/db");
  return {
    and,
    eq,
    schema,
    db: {
      transaction: async (run: (tx: unknown) => Promise<unknown>) =>
        run({
          query: {
            oidcVercel: {
              findFirst: async () => ({ id: "vco_test" }),
            },
          },
          update: () => ({
            set: () => ({ where: async () => undefined }),
          }),
          delete: (table: unknown) => ({
            where: async () => {
              deletedTables(table);
            },
          }),
          insert: (table: unknown) => ({
            values: async (rows: unknown) => {
              insertedRows(table, rows);
            },
          }),
        }),
    },
  };
});

vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return {
    workspaceProcedure: initTRPC
      .context<{
        workspace: { id: string };
        user: { id: string };
        audit: { location: string; userAgent: string };
      }>()
      .create().procedure,
  };
});

function caller() {
  const t = initTRPC
    .context<{
      workspace: { id: string };
      user: { id: string };
      audit: { location: string; userAgent: string };
    }>()
    .create();
  return t.router({ update: updateOIDCVercelIntegration }).createCaller({
    workspace: { id: "ws_test" },
    user: { id: "user_test" },
    audit: { location: "", userAgent: "test" },
  });
}

beforeEach(() => {
  insertedRows.mockClear();
  deletedTables.mockClear();
});

it("replaces integration permissions from root-key permission slugs", async () => {
  await caller().update({
    id: "vco_test",
    name: "Production API",
    vercelTeamSlug: "acme",
    vercelProjectName: "storefront",
    vercelEnvironment: "production",
    enabled: true,
    permissions: ["api.*.read_api", "api.*.create_key"],
  });

  expect(deletedTables).toHaveBeenCalledTimes(1);
  expect(insertedRows.mock.calls[0]?.[1]).toMatchObject([
    {
      principalId: "vco_test",
      principalType: "oidc",
      slug: "api.*.read_api",
      workspaceId: "ws_test",
    },
    {
      principalId: "vco_test",
      principalType: "oidc",
      slug: "api.*.create_key",
      workspaceId: "ws_test",
    },
  ]);
});
