import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { createOIDCVercelIntegration } from "./create";
import type { OIDCVercelInput } from "./validation";

const insertedRows = vi.hoisted(() => vi.fn());
const upsertPermissions = vi.hoisted(() => vi.fn());

vi.mock("@/lib/db", async () => {
  const { schema } = await import("@unkey/db");
  return {
    db: {
      transaction: async (run: (tx: unknown) => Promise<unknown>) =>
        run({
          insert: (table: unknown) => ({
            values: (rows: unknown) => {
              insertedRows(table, rows);
              return Promise.resolve();
            },
          }),
        }),
    },
    schema,
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

const input: OIDCVercelInput = {
  name: "Production API",
  vercelTeamSlug: "acme",
  vercelProjectName: "storefront",
  vercelEnvironment: "production",
  permissions: ["api.*.read_api", "api.*.create_key"],
};

function caller() {
  const t = initTRPC
    .context<{
      workspace: { id: string };
      user: { id: string };
      audit: { location: string; userAgent: string };
    }>()
    .create();
  return t.router({ create: createOIDCVercelIntegration }).createCaller({
    workspace: { id: "ws_test" },
    user: { id: "user_test" },
    audit: { location: "", userAgent: "test" },
  });
}

beforeEach(() => {
  insertedRows.mockClear();
  upsertPermissions.mockReset();
  upsertPermissions.mockResolvedValue({
    permissions: [
      { id: "perm_read", slug: "api.*.read_api" },
      { id: "perm_create", slug: "api.*.create_key" },
    ],
    auditLogs: [],
  });
});

it("rejects URN grants for another workspace", async () => {
  await expect(
    caller().create({
      ...input,
      permissions: ["unkey:v1:ws_other:projects/*/keyspaces/*/keys/*#verify"],
    }),
  ).rejects.toMatchObject({ code: "BAD_REQUEST" });
  expect(insertedRows).not.toHaveBeenCalled();
});

it("creates the integration with root-key permission slugs", async () => {
  const result = await caller().create(input);

  expect(result.id).toMatch(/^vco_/);
  expect(upsertPermissions).toHaveBeenCalledWith(
    expect.anything(),
    expect.objectContaining({ workspace: { id: "ws_test" } }),
    "ws_unkey",
    input.permissions,
  );
  expect(insertedRows).toHaveBeenCalledTimes(2);
  expect(insertedRows.mock.calls[0]?.[1]).toMatchObject({
    id: result.id,
    audience: result.audience,
  });
  expect(insertedRows.mock.calls[1]?.[1]).toEqual([
    { oidcVercelId: result.id, permissionId: "perm_read", workspaceId: "ws_test" },
    { oidcVercelId: result.id, permissionId: "perm_create", workspaceId: "ws_test" },
  ]);
});
