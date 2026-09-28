import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { updateOIDCVercelIntegration } from "./update";

const insertedRows = vi.hoisted(() => vi.fn());
const deletedTables = vi.hoisted(() => vi.fn());
const upsertPermissions = vi.hoisted(() => vi.fn());

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

vi.mock("@/lib/env", () => ({
  env: () => ({ UNKEY_WORKSPACE_ID: "ws_unkey" }),
}));

vi.mock("../rbac", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../rbac")>()),
  upsertPermissions,
}));

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
  upsertPermissions.mockReset();
  upsertPermissions.mockResolvedValue({
    permissions: [
      { id: "perm_read", slug: "api.*.read_api" },
      { id: "perm_create", slug: "api.*.create_key" },
    ],
    auditLogs: [],
  });
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

  expect(upsertPermissions).toHaveBeenCalledWith(
    expect.anything(),
    expect.objectContaining({ workspace: { id: "ws_test" } }),
    "ws_unkey",
    ["api.*.read_api", "api.*.create_key"],
  );
  expect(deletedTables).toHaveBeenCalledTimes(1);
  expect(insertedRows).toHaveBeenCalledWith(expect.anything(), [
    { oidcVercelId: "vco_test", permissionId: "perm_read", workspaceId: "ws_test" },
    { oidcVercelId: "vco_test", permissionId: "perm_create", workspaceId: "ws_test" },
  ]);
});
