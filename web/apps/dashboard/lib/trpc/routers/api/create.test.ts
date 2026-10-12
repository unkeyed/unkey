import { db, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { expect, it, vi } from "vitest";
import { createApiCore } from "./create";

const database = vi.hoisted(() => ({
  projectId: "",
  rows: new Map<unknown, unknown>(),
}));

vi.mock("@/lib/db", async () => {
  const { schema } = await import("@unkey/db");
  return {
    schema,
    db: {
      transaction: async (run: (tx: unknown) => Promise<unknown>) =>
        run({
          query: {
            projects: { findFirst: async () => ({ id: database.projectId }) },
          },
          insert: (table: unknown) => ({
            values: async (rows: unknown) => {
              database.rows.set(table, rows);
            },
          }),
        }),
    },
  };
});

it("audits the created API and its keyAuth in the same outbox event", async () => {
  database.rows.clear();
  database.projectId = newId("project");
  const workspaceId = newId("workspace");
  const userId = newId("test");
  const name = "Customer API";

  const api = await db.transaction((tx) =>
    createApiCore(
      { name },
      {
        workspace: { id: workspaceId },
        user: { id: userId },
        audit: { location: "127.0.0.1", userAgent: "audit-regression" },
      },
      tx,
    ),
  );

  expect(database.rows.get(schema.apis)).toMatchObject({
    id: api.id,
    keyAuthId: api.keyAuthId,
    name,
    workspaceId,
  });
  expect(database.rows.get(schema.keyAuth)).toMatchObject({
    id: api.keyAuthId,
    workspaceId,
  });
  expect(database.rows.get(schema.clickhouseOutbox)).toMatchObject([
    {
      version: "audit_log.v1",
      workspaceId,
      payload: {
        workspace_id: workspaceId,
        event: "api.create",
        actor: { type: "user", id: userId },
        targets: [
          { type: "api", id: api.id, name },
          { type: "keyAuth", id: api.keyAuthId },
        ],
      },
    },
  ]);
});
