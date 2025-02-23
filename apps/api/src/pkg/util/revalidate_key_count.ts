import { type Database, and, eq, isNull, schema, sql } from "@unkey/db";

/**
 * Counts the keys in a keyspace and writes it back to `sizeApprox`
 */
export async function revalidateKeyCount(db: Database, keyAuthId: string): Promise<void> {
  const rows = await db
    .select({ count: sql<string>`count(*)` })
    .from(schema.keys)
    .where(and(eq(schema.keys.keyAuthId, keyAuthId), isNull(schema.keys.deletedAtM)));

  await db
    .update(schema.keyAuth)
    .set({
      /**
       * I'm pretty sure it will always return 1 row, but in case it doesn't, we fall back to 0
       */
      sizeApprox: Number.parseInt(rows.at(0)?.count ?? "0"),
      sizeLastUpdatedAt: Date.now(),
    })
    .where(eq(schema.keyAuth.id, keyAuthId));
}
