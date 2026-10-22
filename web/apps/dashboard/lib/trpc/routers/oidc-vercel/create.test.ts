import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { createOIDCVercelIntegration } from "./create";
import type { OIDCVercelInput } from "./validation";

const insertedRows = vi.hoisted(() => vi.fn());

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
  expect(insertedRows).toHaveBeenCalledTimes(2);
  expect(insertedRows.mock.calls[0]?.[1]).toMatchObject({
    id: result.id,
    audience: result.audience,
  });
  expect(insertedRows.mock.calls[1]?.[1]).toMatchObject([
    {
      principalId: result.id,
      principalType: "oidc",
      slug: "api.*.read_api",
      workspaceId: "ws_test",
    },
    {
      principalId: result.id,
      principalType: "oidc",
      slug: "api.*.create_key",
      workspaceId: "ws_test",
    },
  ]);
});
