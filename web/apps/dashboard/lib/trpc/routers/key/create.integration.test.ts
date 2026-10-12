import { db, eq, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { afterAll, describe, expect, it, onTestFinished, vi } from "vitest";
import { createKeyCore } from "./create";

const database = vi.hoisted(() => ({ close: async () => {} }));

vi.mock("@/lib/db", async () => {
  const { drizzle, schema, eq } = await import("@unkey/db");
  const { createPool } = await import("mysql2/promise");
  const pool = createPool(process.env.UNKEY_TEST_DATABASE_URL ?? "mysql://localhost/unkey");
  database.close = () => pool.end();
  return { db: drizzle(pool, { schema, mode: "default" }), schema, eq };
});

vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.create();
  return {
    workspaceProcedure: t.procedure,
    ratelimit: { create: undefined },
    withRatelimit: () => t.middleware(({ next }) => next()),
  };
});

vi.mock("@/lib/vault-client", () => ({ createVaultClient: () => ({}) }));

afterAll(() => database.close());

describe.skipIf(!process.env.UNKEY_TEST_DATABASE_URL)("key creation identity audit targets", () => {
  it.each([true, false])("records only an assigned identity (assigned: %s)", async (assigned) => {
    const workspaceId = newId("workspace");
    const projectId = newId("project");
    const keyAuthId = newId("keyAuth");
    const identityId = newId("identity");
    onTestFinished(async () => {
      await db
        .delete(schema.clickhouseOutbox)
        .where(eq(schema.clickhouseOutbox.workspaceId, workspaceId));
      await db.delete(schema.keys).where(eq(schema.keys.workspaceId, workspaceId));
      await db.delete(schema.identities).where(eq(schema.identities.workspaceId, workspaceId));
      await db.delete(schema.keyAuth).where(eq(schema.keyAuth.workspaceId, workspaceId));
      await db.delete(schema.projects).where(eq(schema.projects.workspaceId, workspaceId));
      await db.delete(schema.workspaces).where(eq(schema.workspaces.id, workspaceId));
    });
    await db.transaction(async (tx) => {
      await tx.insert(schema.workspaces).values({
        id: workspaceId,
        orgId: newId("test"),
        name: "Audit test",
        slug: workspaceId,
        betaFeatures: {},
      });
      await tx.insert(schema.projects).values({
        id: projectId,
        workspaceId,
        name: "Audit test",
        slug: projectId,
      });
      await tx.insert(schema.keyAuth).values({ id: keyAuthId, workspaceId, projectId });
      await tx.insert(schema.identities).values({
        id: identityId,
        workspaceId,
        projectId,
        externalId: newId("test"),
      });
    });
    const result = await db.transaction((tx) =>
      createKeyCore(
        {
          keyAuthId,
          identityId: assigned ? identityId : null,
          externalId: newId("test"),
          bytes: 16,
          enabled: true,
          storeEncryptedKeys: false,
        },
        {
          workspace: { id: workspaceId },
          user: { id: newId("test") },
          audit: { location: "test" },
        },
        tx,
      ),
    );
    const saved = await db.query.keys.findFirst({ where: eq(schema.keys.id, result.keyId) });
    expect(saved?.identityId).toBe(assigned ? identityId : null);
    const logs = await db.query.clickhouseOutbox.findMany({
      where: eq(schema.clickhouseOutbox.workspaceId, workspaceId),
    });
    expect(logs).toHaveLength(1);
    expect(logs[0]?.payload).toMatchObject({
      event: "key.create",
      targets: assigned
        ? [
            { type: "key", id: result.keyId },
            { type: "identity", id: identityId },
          ]
        : [{ type: "key", id: result.keyId }],
    });
  });
});
